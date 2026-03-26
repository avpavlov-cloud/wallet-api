package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/avpavlov-cloud/wallet-api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	// Рекомендую использовать ://github.com
)

func TestTransferIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Подключаемся к БД для теста (берем строку из окружения Docker)
	dbConnStr := os.Getenv("DB_SOURCE")
	pool, err := pgxpool.New(context.Background(), dbConnStr)
	if err != nil {
		t.Fatalf("Не удалось подключиться к БД: %v", err)
	}
	defer pool.Close()

	// 2. Инициализируем роутер со всеми эндпоинтами
	r := SetupRouter(pool)

	t.Run("Success Transfer", func(t *testing.T) {
		transferReq := model.TransferRequest{
			FromAccountID: 1, // Убедитесь, что такие ID есть в вашей БД или создайте их в тесте
			ToAccountID:   2,
			Amount:        10.0,
		}
		jsonData, _ := json.Marshal(transferReq)

		req, _ := http.NewRequest("POST", "/transfer", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-KEY", os.Getenv("API_KEY"))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Проверяем статус и сообщение
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "transfer successful", response["message"])
	})

	t.Run("Insufficient Funds", func(t *testing.T) {
		transferReq := model.TransferRequest{
			FromAccountID: 1,
			ToAccountID:   2,
			Amount:        1000000.0, // Сумма заведомо больше баланса
		}
		jsonData, _ := json.Marshal(transferReq)

		req, _ := http.NewRequest("POST", "/transfer", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-KEY", os.Getenv("API_KEY"))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Ожидаем 400 Bad Request из-за ошибки в транзакции (недостаточно средств)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
