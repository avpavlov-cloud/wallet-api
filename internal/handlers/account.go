package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/avpavlov-cloud/wallet-api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	DbPool *pgxpool.Pool // Или *sql.DB, если используете стандартный драйвер
}

// Конструктор
func NewServer(db *pgxpool.Pool) *Server {
	return &Server{DbPool: db}
}

// CreateAccountHandler godoc
// @Summary      Создать новый счет
// @Description  Создает банковский счет для пользователя с начальным балансом
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        request body model.CreateAccountRequest true "Данные для создания счета"
// @Success      201  {object}  map[string]interface{} "Успешное создание"
// @Failure      400  {object}  map[string]string      "Ошибка валидации"
// @Failure      500  {object}  map[string]string      "Внутренняя ошибка сервера"
// @Router       /accounts [post]
func (s *Server) CreateAccountHandler(c *gin.Context) {
	var req model.CreateAccountRequest
	// Валидация JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SQL запрос на вставку
	query := `INSERT INTO accounts (owner_name, balance, currency) 
                  VALUES ($1, $2, $3) RETURNING id, created_at`

	var id int64
	var createdAt time.Time
	err := s.DbPool.QueryRow(context.Background(), query, req.OwnerName, req.Balance, req.Currency).Scan(&id, &createdAt)

	if err != nil {
		slog.Error("failed to create account",
			"error", err,
			"owner", req.OwnerName,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         id,
		"owner_name": req.OwnerName,
		"balance":    req.Balance,
		"currency":   req.Currency,
		"created_at": createdAt,
	})

	slog.Info("account created successfully", "id", id)
}

func (s *Server) runTransferTx(ctx context.Context, req model.TransferRequest) error {
	txOptions := pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadWrite,
	}

	tx, err := s.DbPool.BeginTx(ctx, txOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Сортировка для защиты от Deadlock
	firstID, secondID := req.FromAccountID, req.ToAccountID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	// Явная блокировка строк
	if _, err := tx.Exec(ctx, "SELECT id FROM accounts WHERE id = $1 FOR UPDATE", firstID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SELECT id FROM accounts WHERE id = $1 FOR UPDATE", secondID); err != nil {
		return err
	}

	// Списание
	res, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1", req.Amount, req.FromAccountID)
	if err != nil || res.RowsAffected() == 0 {
		return fmt.Errorf("insufficient funds or sender error")
	}

	// Зачисление
	res, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", req.Amount, req.ToAccountID)
	if err != nil || res.RowsAffected() == 0 {
		return fmt.Errorf("receiver error")
	}

	// Лог
	_, err = tx.Exec(ctx, "INSERT INTO transactions (from_account_id, to_account_id, amount) VALUES ($1, $2, $3)", req.FromAccountID, req.ToAccountID, req.Amount)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// TransferHandler godoc
// @Summary      Перевод денежных средств
// @Description  Выполняет перевод денег между двумя счетами в рамках одной транзакции
// @Tags         transfers
// @Accept       json
// @Produce      json
// @Param        request body model.TransferRequest true "Данные перевода"
// @Success      200  {object}  map[string]string "Успешный перевод"
// @Failure      400  {object}  map[string]string "Недостаточно средств или неверные ID"
// @Failure      500  {object}  map[string]string "Ошибка транзакции"
// @Router       /transfer [post]
func (s *Server) TransferHandler(c *gin.Context) {
	var req model.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.FromAccountID == req.ToAccountID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot transfer to the same account"})
		return
	}

	const maxRetries = 3
	var err error

	for i := 0; i < maxRetries; i++ {
		err = s.runTransferTx(c.Request.Context(), req)

		if err == nil {
			c.JSON(http.StatusOK, gin.H{"message": "transfer successful"})
			return
		}

		// Проверяем, является ли ошибка конфликтом сериализации (40001)
		if isSerializationError(err) {
			slog.Warn("Retry transfer due to serialization conflict", "attempt", i+1, "from", req.FromAccountID)
			time.Sleep(time.Millisecond * 50) // Небольшая пауза перед повтором
			continue
		}

		// Если это ошибка бизнес-логики (нет денег), выходим из цикла сразу
		break
	}

	// Если мы здесь, значит все попытки провалены или ошибка фатальна
	slog.Error("Transfer finally failed", "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// Вспомогательная функция проверки кода ошибки
func isSerializationError(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "40001"
	}
	return false
}

// GetAccountHandlerfunc godoc
// @Summary      Получить информацию о счете
// @Description  Возвращает данные счета и историю последних 5 транзакций
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "ID счета"
// @Success      200  {object}  map[string]interface{} "Данные счета и история"
// @Failure      404  {object}  map[string]string      "Счет не найден"
// @Failure      500  {object}  map[string]string      "Ошибка базы данных"
// @Router       /accounts/{id} [get]
func (s *Server) GetAccountHandlerfunc(c *gin.Context) {
	id := c.Param("id")

	// Используем транзакцию только для чтения (Read Committed),
	// чтобы гарантировать Консистентность (C)
	tx, err := s.DbPool.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer tx.Rollback(context.Background())

	// 1. Получаем данные счета
	var acc model.Account
	err = tx.QueryRow(context.Background(),
		"SELECT id, owner_name, balance, currency, created_at FROM accounts WHERE id = $1", id).
		Scan(&acc.ID, &acc.OwnerName, &acc.Balance, &acc.Currency, &acc.CreatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	// 2. Получаем последние 5 транзакций (где этот счет был отправителем или получателем)
	rows, err := tx.Query(context.Background(), `
        SELECT id, from_account_id, to_account_id, amount, created_at 
        FROM transactions 
        WHERE from_account_id = $1 OR to_account_id = $1 
        ORDER BY created_at DESC 
        LIMIT 5`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions"})
		return
	}
	defer rows.Close()

	var history []gin.H
	for rows.Next() {
		var tid, from, to int64
		var amt float64
		var cat time.Time
		if err := rows.Scan(&tid, &from, &to, &amt, &cat); err == nil {
			history = append(history, gin.H{
				"id": tid, "from": from, "to": to, "amount": amt, "at": cat,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"account": acc,
		"history": history,
	})
}
