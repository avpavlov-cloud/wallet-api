# 🏦 Wallet Service API (Industrial Grade)

Высоконагруженная банковская система управления кошельками и переводами.
Проект демонстрирует реализацию принципов **ACID** в PostgreSQL, устойчивость к нагрузкам и современные паттерны разработки на Go.

---

## 🛠 Технологический стек

* **Язык:** Go (Golang) + Gin Framework
* **База данных:** PostgreSQL 15 (pgxpool)
* **Инфраструктура:** Docker, Docker Compose, Air (Hot Reload)
* **Документация:** Swagger (swaggo)
* **Мониторинг:** Prometheus, Grafana
* **Тестирование:** Testify (Integration & Unit tests)

---

## 🏗 Ключевые архитектурные решения (Industrial Standards)

1. **ACID Transactions**
   Гарантированная целостность данных при переводах

2. **Deadlock Prevention**
   Сортировка ID аккаунтов при блокировке (`FOR UPDATE`) для предотвращения взаимных блокировок

3. **Idempotency Key**
   Защита от двойных списаний при повторных запросах (UUID)

4. **Transactional Outbox Pattern**
   Надежная доставка событий (уведомлений) через отдельную таблицу в той же транзакции

5. **Retry Logic**
   Автоматический повтор транзакций при ошибках сериализации (`40001`)

6. **Rate Limiting**
   Защита API от спама и DoS-атак (алгоритм Token Bucket)

7. **Structured Logging**
   Логирование в формате JSON (`slog`) для анализа в ELK / Loki

---

## 🚀 Быстрый старт

### 1. Запуск инфраструктуры

```bash
docker-compose up -d
```

### 2. Управление миграциями

```bash
make migrate-up    # Накатить таблицы и индексы
make migrate-down  # Откатить изменения
```

### 3. Генерация Swagger документации

```bash
swag init -g cmd/api/main.go --parseDependency --parseInternal
```

Документация доступна по адресу:
👉 [http://localhost:8000/swagger/index.html](http://localhost:8000/swagger/index.html)

---

## 📑 Примеры использования API

### Создание счета

```bash
curl -X POST http://localhost:8000/accounts \
  -H "Content-Type: application/json" \
  -d '{"owner_name": "Aleksey", "balance": 1000.50, "currency": "USD"}'
```

---

### Перевод денег (с защитой UUID)

⚠️ Критически важно передавать уникальный `idempotency_key` для предотвращения дублей

```bash
curl -X POST http://localhost:8000/transfer \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: super-secret-token-123" \
  -d '{
    "from_account_id": 1,
    "to_account_id": 2,
    "amount": 50.0,
    "idempotency_key": "550e8400-e29b-41d4-a716-446655440001"
  }'
```

---

### Проверка баланса и истории

```bash
curl http://localhost:8000/accounts/1 \
  -H "X-API-KEY: super-secret-token-123"
```

---

## 🧪 Тестирование и мониторинг

### Запуск тестов

```bash
# Обычные тесты
docker compose exec app go test -v ./...

# Проверка покрытия
make test-coverage
```

---

### Проверка Rate Limit (стресс-тест)

```bash
for i in {1..10}; do 
  curl -s -o /dev/null -w "Запрос $i: %{http_code}\n" \
    -X POST http://localhost:8000/transfer \
    -H "Content-Type: application/json" \
    -H "X-API-KEY: super-secret-token-123" \
    -d "{\"from_account_id\": 1, \"to_account_id\": 2, \"amount\": 1.0, \"idempotency_key\": \"uuid-$i\"}"
done
```

---

## 🔍 Инспекция базы данных (Deep Dive)

### Проверка балансов напрямую в PostgreSQL

```bash
docker exec -it wallet-api-postgres-1 \
psql -U user -d wallet_db \
-c "SELECT id, owner_name, balance, currency FROM accounts ORDER BY id;"
```

---

### Просмотр Outbox событий (JSONB + GIN)

```bash
docker exec -it wallet-api-postgres-1 \
psql -U user -d wallet_db \
-c "SELECT id, event_type, payload, status FROM outbox_events;"
```

---

### Мониторинг состояния контейнеров

```bash
docker compose ps
```

Ищите статус **(healthy)** у сервисов `app` и `postgres`.



