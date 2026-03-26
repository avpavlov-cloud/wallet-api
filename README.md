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
 curl -X POST http://localhost:8000/transfer      -H "Content-Type: application/json"      -d '{"from_account_id": 1, "to_account_id": 2, "amount": 50.0}'
 ```