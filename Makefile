include .env
export

.PHONY: build run dev tidy test vet migrate-up migrate-down migrate-status docker-up docker-down docker-reset seed validate

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

tidy:
	go mod tidy

test:
	go test ./... -count=1 -race

vet:
	go vet ./...

migrate-up:
	goose -dir migrations/sql postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations/sql postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir migrations/sql postgres "$(DATABASE_URL)" status

dev:
	go run ./cmd/server

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-reset: docker-down
	docker compose down -v
	docker compose up -d

seed:
	go run ./cmd/seed

validate:
	go run ./cmd/validate
