package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avpavlov-cloud/wallet-api/internal/handlers"
	"github.com/avpavlov-cloud/wallet-api/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	ginprometheus "github.com/zsais/go-gin-prometheus"

	_ "github.com/avpavlov-cloud/wallet-api/docs"
	swaggerFiles "github.com/swaggo/files"     // статические файлы Swagger UI
	ginSwagger "github.com/swaggo/gin-swagger" // адаптер для Gin
)

func SetupRouter(pool *pgxpool.Pool) *gin.Engine {
	server := handlers.NewServer(pool)
	r := gin.Default()

	// Публичный эндпоинт для Docker/Kubernetes
	r.GET("/health", func(c *gin.Context) {
		// Здесь можно добавить проверку связи с БД: pool.Ping()
		c.Status(http.StatusOK)
	})

	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

	// 1. ПУБЛИЧНЫЕ РОУТЫ (Без авторизации)
	// Swagger должен быть доступен всем
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 2. ЗАЩИЩЕННЫЕ РОУТЫ
	authGroup := r.Group("/")

	// Сначала авторизация, потом лимитер
	authGroup.Use(middleware.AuthMiddleware())

	limiter := middleware.NewIPlimiter()
	authGroup.Use(middleware.RateLimitMiddleware(limiter))

	{
		// Используем именно authGroup, чтобы работали Middleware
		authGroup.POST("/accounts", server.CreateAccountHandler)
		authGroup.POST("/transfer", server.TransferHandler)
		authGroup.GET("/accounts/:id", server.GetAccountHandlerfunc)

		// НОВЫЙ РОУТ: Аналитический отчет
		authGroup.GET("/reports/volume", server.GetVolumeReportHandler)
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

	config, _ := pgxpool.ParseConfig(os.Getenv("DB_SOURCE"))
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement

	// Используем пул соединений для высокой нагрузки
	dbPool, err := pgxpool.NewWithConfig(context.Background(), config)
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

	// ФОНОВЫЙ ВОРКЕР (Background Process)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			// Используем фоновый контекст с таймаутом
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)

			// Вызываем REFRESH MATERIALIZED VIEW
			_, err := dbPool.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY daily_volume_report")
			if err != nil {
				log.Printf("Ошибка обновления отчета: %v", err)
			}
			cancel()
		}
	}()

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
