package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avpavlov-cloud/wallet-api/internal/handlers"
	"github.com/avpavlov-cloud/wallet-api/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/avpavlov-cloud/wallet-api/docs"
	swaggerFiles "github.com/swaggo/files"     // статические файлы Swagger UI
	ginSwagger "github.com/swaggo/gin-swagger" // адаптер для Gin
)

func SetupRouter(pool *pgxpool.Pool) *gin.Engine {
	server := handlers.NewServer(pool)
	// 2. Инициализация Gin
	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(middleware.JSONLogger())

	protected := r.Group("/")
	protected.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/accounts", server.CreateAccountHandler)
		protected.POST("/transfer", server.TransferHandler)
		protected.GET("/accounts/:id", server.GetAccountHandlerfunc)
	}

	return r
}

func main() {
	// Настраиваем JSON-обработчик: логи будут выходить в одну строку JSON
	// nil в опциях означает уровень по умолчанию (Info)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Устанавливаем его как глобальный логгер для всего приложения
	slog.SetDefault(logger)

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
		logger.Error("Database unreachable", "error", err)
		os.Exit(1)
	}

	r := SetupRouter(dbPool) // Используем общую настройку
	// --- GRACEFUL SHUTDOWN LOGIC ---

	srv := &http.Server{
		Addr:    ":8000",
		Handler: r,
	}

	// Запуск сервера в отдельной горутине
	go func() {
		slog.Info("Server is starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Канал для системных сигналов
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Блокировка до получения сигнала (Ctrl+C или docker stop)

	slog.Info("shutting down server...")

	// Контекст ожидания завершения (5 секунд)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Останавливаем HTTP сервер
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server exiting gracefully")
}
