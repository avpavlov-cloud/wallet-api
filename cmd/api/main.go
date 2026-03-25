package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Подключение к БД (в реальном проекте берется из env)
	dbConnStr := "postgres://user:password@localhost:5432/wallet_db?sslmode=disable"

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

	// Тестовый роут (Health Check)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "up",
			"db":     "connected",
		})
	})

	// 3. Запуск сервера
	log.Println("Server starting on :8000")
	r.Run(":8000")
}
