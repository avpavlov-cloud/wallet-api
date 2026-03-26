package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/avpavlov-cloud/wallet-api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

	r.POST("/transfer", func(c *gin.Context) {
		var req model.TransferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Защита от перевода самому себе
		if req.FromAccountID == req.ToAccountID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot transfer to the same account"})
			return
		}

		tx, err := dbPool.Begin(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "tx failed"})
			return
		}
		defer tx.Rollback(context.Background())

		// --- МЕХАНИЗМ СОРТИРОВКИ ID ---
		firstID, secondID := req.FromAccountID, req.ToAccountID
		if firstID > secondID {
			firstID, secondID = secondID, firstID
		}

		// Сначала блокируем (SELECT FOR UPDATE) "меньший" ID, затем "больший"
		// Это гарантирует, что любая другая транзакция по этим ID пойдет по тому же пути
		_, err = tx.Exec(context.Background(), "SELECT id FROM accounts WHERE id = $1 FOR UPDATE", firstID)
		_, err = tx.Exec(context.Background(), "SELECT id FROM accounts WHERE id = $1 FOR UPDATE", secondID)
		// ------------------------------

		// Теперь выполняем саму логику списания/зачисления
		// Снимаем деньги
		res, err := tx.Exec(context.Background(),
			"UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1",
			req.Amount, req.FromAccountID)

		if err != nil || res.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient funds or sender error"})
			return
		}

		// Зачисляем деньги
		res, err = tx.Exec(context.Background(),
			"UPDATE accounts SET balance = balance + $1 WHERE id = $2",
			req.Amount, req.ToAccountID)

		if err != nil || res.RowsAffected() == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "receiver error"})
			return
		}

		// Записываем лог
		_, _ = tx.Exec(context.Background(),
			"INSERT INTO transactions (from_account_id, to_account_id, amount) VALUES ($1, $2, $3)",
			req.FromAccountID, req.ToAccountID, req.Amount)

		if err := tx.Commit(context.Background()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "transfer successful"})
	})

	r.GET("/accounts/:id", func(c *gin.Context) {
		id := c.Param("id")

		// Используем транзакцию только для чтения (Read Committed),
		// чтобы гарантировать Консистентность (C)
		tx, err := dbPool.Begin(context.Background())
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
			var cat string
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
	})

	// 3. Запуск сервера
	log.Println("Server starting on :8000")
	r.Run(":8000")
}
