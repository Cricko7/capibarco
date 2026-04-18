APP=analytics-service

.PHONY: build run test lint tidy migrate-up migrate-down docker-build

build:
	go build -o bin/$(APP) ./cmd/analytics-service

run:
	go run ./cmd/analytics-service --config ./configs/config.yaml

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

migrate-up:
	migrate -path migrations -database "$$ANALYTICS_DSN" up

migrate-down:
	migrate -path migrations -database "$$ANALYTICS_DSN" down 1

docker-build:
	docker build -t $(APP):local .
