SERVICE=notification-service
BIN_DIR=./bin

.PHONY: build test test-race lint fmt run migrate migrate-down proto tidy

build:
	go build -o $(BIN_DIR)/$(SERVICE) ./cmd/notification-service

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	go vet ./...

fmt:
	gofmt -w cmd internal

run:
	go run ./cmd/notification-service -config ./configs/config.yaml

migrate:
	migrate -path ./migrations -database "$${NOTIFICATION_DB_DSN}" up

migrate-down:
	migrate -path ./migrations -database "$${NOTIFICATION_DB_DSN}" down 1

proto:
	go run github.com/bufbuild/buf/cmd/buf@v1.46.0 generate

tidy:
	go mod tidy
