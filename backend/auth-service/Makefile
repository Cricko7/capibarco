SERVICE=auth-service
BINARY=authsvc

.PHONY: build test lint run migrate-up migrate-down docker-up docker-down

build:
	go build -o bin/$(BINARY) ./cmd/authsvc

test:
	go test ./...

lint:
	golangci-lint run ./...

run:
	go run ./cmd/authsvc

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v
