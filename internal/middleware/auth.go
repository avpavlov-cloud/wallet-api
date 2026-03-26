package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем ключ из заголовка X-API-KEY
		apiKey := c.GetHeader("X-API-KEY")
		expectedKey := os.Getenv("API_KEY")

		// Если ключа нет или он не совпадает — прерываем запрос
		if apiKey == "" || apiKey != expectedKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Неавторизованный доступ"})
			c.Abort() // Критически важно: это не дает запросу идти дальше
			return
		}
		c.Next() // Переход к следующему обработчику
	}
}
