package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// JSONLogger возвращает Middleware для Gin, который пишет логи в формате JSON
func JSONLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Передаем управление следующему обработчику (эндпоинту)
		c.Next()

		// Данные собираются ПОСЛЕ выполнения запроса
		end := time.Since(start)
		status := c.Writer.Status()

		// Используем структурированный логгер slog
		slog.Info("http_request",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", query),
			slog.Int("status", status),
			slog.Duration("duration", end),
			slog.String("ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		)
	}
}
