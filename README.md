Поднять docker
```bash
docker-compose up -d
```

Миграции 
```bash
make migrate-up
make migrate-down
```

Создать счёт
```bash
 curl -X POST http://localhost:8000/accounts      -H "Content-Type: application/json"      -d '{"owner_name": "Aleksey", "balance": 1000.50, "currency": "USD"}'
 ```

 Показать логи
 ```bash
 docker compose logs app --tail 20
 ```

 Перевод денег с одного счёта на другой счёт
 ```bash
curl -X POST http://localhost:8000/transfer      -H "Content-Type: application/json"      -H "X-API-KEY: super-secret-token-123"      -d '{
       "from_account_id": 1, 
       "to_account_id": 2, 
       "amount": 50.0, 
       "idempotency_key": "550e8400-e29b-41d4-a716-446655440001"
     }'
 ```

 Запрос с авторизациями после включения middleware
 ```bash
curl http://localhost:8000/accounts/1      -H "X-API-KEY: super-secret-token-123"
{"account":{"id":1,"owner_name":"Aleksey","balance":850,"currency":"USD","created_at":"2026-03-26T07:03:54.291795Z"},"history":[{"amount":50,"at":"2026-03-26T07:48:36.954406Z","from":1,"id":3,"to":2},{"amount":50,"at":"2026-03-26T07:33:14.026734Z","from":1,"id":2,"to":2},{"amount":50,"at":"2026-03-26T07:28:52.577251Z","from":1,"id":1,"to":2}]}
```  

Посмотреть логи относящиеся только к приложению
```bash
docker compose logs -f app
```

Выполнение тестов
```bash
docker compose exec app go test -v ./...
```

Выполнение тестов покрытия тестами
```bash
make test-coverage
```

Генерация swagger
```bash
swag init -g cmd/api/main.go --parseDependency --parseInternal
```