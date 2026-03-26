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