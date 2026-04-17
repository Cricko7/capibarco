SERVICE=user-service

.PHONY: build test lint run migrate proto tidy

build:
	go build ./cmd/user-service

test:
	go test ./...

lint:
	go vet ./...

run:
	go run ./cmd/user-service -config ./configs/config.yaml

migrate:
	migrate -path ./migrations -database "$${USER_DB_DSN}" up

proto:
	buf generate

tidy:
	go mod tidy
