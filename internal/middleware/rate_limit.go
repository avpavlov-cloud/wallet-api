package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPlimiter хранит ограничители для каждого IP адреса
type IPlimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.Mutex
}

func NewIPlimiter() *IPlimiter {
	return &IPlimiter{ips: make(map[string]*rate.Limiter)}
}

func (l *IPlimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.ips[ip]
	if !exists {
		// Разрешаем 2 запроса в секунду, с возможностью накопить до 5 (Burst)
		limiter = rate.NewLimiter(2, 5)
		l.ips[ip] = limiter
	}
	return limiter
}

func RateLimitMiddleware(l *IPlimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := l.getLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Слишком много запросов. Попробуйте позже.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
