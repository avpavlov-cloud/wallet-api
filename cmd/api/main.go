package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Модель данных для входящего JSON (с валидацией Gin)
type CreateAccountRequest struct {
	OwnerName string  `json:"owner_name" binding:"required"`
	Currency  string  `json:"currency" binding:"required,oneof=USD EUR RUB"`
	Balance   float64 `json:"balance" binding:"required,gte=0"`
}

func main() {
	dbConnStr := os.Getenv("DB_SOURCE")
	if dbConnStr == "" {
		// Это сработает, только если вы запускаете без Докера
		dbConnStr = "postgres://user:password@localhost:5432/wallet_db?sslmode=disable"
	}

	// Используем пул соединений для высокой нагрузки
	dbPool, err := pgxpool.New(context.Background(), dbConnStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Проверка связи
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Database unreachable: %v", err)
	}

	// 2. Инициализация Gin
	r := gin.Default()

	// Эндпоинт создания счета
	r.POST("/accounts", func(c *gin.Context) {
		var req CreateAccountRequest
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
		err := dbPool.QueryRow(context.Background(), query, req.OwnerName, req.Balance, req.Currency).Scan(&id, &createdAt)

		if err != nil {
			log.Printf("Error creating account: %v", err)
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
	})

	type TransferRequest struct {
		FromAccountID int64   `json:"from_account_id" binding:"required"`
		ToAccountID   int64   `json:"to_account_id" binding:"required"`
		Amount        float64 `json:"amount" binding:"required,gt=0"`
	}

	r.POST("/transfer", func(c *gin.Context) {
		var req TransferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Начинаем транзакцию (Atomicity)
		tx, err := dbPool.Begin(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not start transaction"})
			return
		}
		// Если функция завершится с ошибкой, транзакция откатится автоматически
		defer tx.Rollback(context.Background())

		// 1. Снимаем деньги (Postgres проверит CHECK(balance >= 0) автоматически)
		// Используем FOR UPDATE, чтобы заблокировать строку от изменений другими запросами (Isolation)
		res, err := tx.Exec(context.Background(),
			"UPDATE accounts SET balance = balance - $1 WHERE id = $2", req.Amount, req.FromAccountID)

		if err != nil || res.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient funds or sender not found"})
			return
		}

		// 2. Зачисляем деньги
		res, err = tx.Exec(context.Background(),
			"UPDATE accounts SET balance = balance + $1 WHERE id = $2", req.Amount, req.ToAccountID)

		if err != nil || res.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "receiver not found"})
			return
		}

		// 3. Записываем историю (Durability)
		_, err = tx.Exec(context.Background(),
			"INSERT INTO transactions (from_account_id, to_account_id, amount) VALUES ($1, $2, $3)",
			req.FromAccountID, req.ToAccountID, req.Amount)

		// Фиксируем все изменения (Commit)
		if err := tx.Commit(context.Background()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "transfer successful"})
	})

	// 3. Запуск сервера
	log.Println("Server starting on :8000")
	r.Run(":8000")
}
