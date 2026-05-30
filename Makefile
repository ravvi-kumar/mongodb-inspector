.PHONY: build run migrate-up migrate-down tidy

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

tidy:
	go mod tidy

migrate-up:
	goose -dir migrations/sql postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations/sql postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir migrations/sql postgres "$(DATABASE_URL)" status

dev:
	go run ./cmd/server
