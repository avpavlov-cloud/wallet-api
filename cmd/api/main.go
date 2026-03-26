package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avpavlov-cloud/wallet-api/internal/handlers"
	"github.com/avpavlov-cloud/wallet-api/internal/middleware"
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

	server := handlers.NewServer(dbPool)

	// 2. Инициализация Gin
	r := gin.Default()

	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/accounts", server.CreateAccountHandler)
		protected.POST("/transfer", server.TransferHandler)
		protected.GET("/accounts/:id", server.GetAccountHandlerfunc)
	}
	// --- GRACEFUL SHUTDOWN LOGIC ---

	srv := &http.Server{
		Addr:    ":8000",
		Handler: r,
	}

	// Запуск сервера в отдельной горутине
	go func() {
		log.Println("Server starting on :8000")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Канал для системных сигналов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Блокировка до получения сигнала (Ctrl+C или docker stop)

	log.Println("Shutting down server...")

	// Контекст ожидания завершения (5 секунд)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Останавливаем HTTP сервер
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting gracefully")
}
