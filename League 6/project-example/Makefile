.PHONY: run build migrate-up migrate-down compose-up compose-down

run:
	go run ./cmd

build:
	go build -o bin/snapbin ./cmd

migrate-up:
	goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	goose -dir migrations postgres "$$DATABASE_URL" down

compose-up:
	docker compose up --build

compose-down:
	docker compose down
