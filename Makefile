# Переменная для удобства (используем localhost, т.к. запускаем с хоста)
DB_URL=postgres://user:password@postgres:5432/wallet_db?sslmode=disable

migrate-up:
	docker compose run --rm migrate -path=migrations/ -database "$(DB_URL)" up


migrate-down:
	docker compose run --rm migrate -path=/migrations/ -database "$(DB_URL)" down 1

test:
	docker compose run --rm app go test -v ./...
