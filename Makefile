# Переменная для удобства (используем localhost, т.к. запускаем с хоста)
DB_URL=postgres://user:password@postgres:5432/wallet_db?sslmode=disable

migrate-up:
	docker compose run --rm migrate -path=migrations/ -database "$(DB_URL)" up


migrate-down:
	docker compose run --rm migrate -path=/migrations/ -database "$(DB_URL)" down 1

test:
	docker compose run --rm app go test -v ./...

# Запуск тестов с генерацией профиля покрытия
test-coverage:
	docker compose exec app go test -v -coverprofile=coverage.out ./...
	# Превращаем бинарный отчет в HTML (опционально, если есть go локально)
	go tool cover -html=coverage.out -o coverage.html

# Создание новой миграции. Использование: make migrate-create name=add_idempotency_key
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Ошибка: укажите имя миграции. Пример: make migrate-create name=my_migration"; \
		exit 1; \
	fi
	docker run --rm \
		-v $(shell pwd)/migrations:/migrations \
		--user $(shell id -u):$(shell id -g) \
		migrate/migrate create -ext sql -dir /migrations/ -seq $(name)
