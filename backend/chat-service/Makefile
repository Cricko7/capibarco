APP := chat-service
PKG := ./...
BIN_DIR := bin
GOCACHE ?= $(CURDIR)/tmp/gocache
DATABASE_URL ?= postgres://chat:chat@localhost:5432/chat?sslmode=disable

.PHONY: build run test test-race lint fmt tidy generate migrate-up migrate-down docker-build compose-up compose-down clean

build:
	GOCACHE=$(GOCACHE) go build -trimpath -o $(BIN_DIR)/$(APP) ./cmd/chat-service

run:
	GOCACHE=$(GOCACHE) go run ./cmd/chat-service

test:
	GOCACHE=$(GOCACHE) go test -coverprofile=coverage.out $(PKG)

test-race:
	GOCACHE=$(GOCACHE) go test -race $(PKG)

lint:
	golangci-lint run $(PKG)

fmt:
	gofmt -w ./cmd ./internal

tidy:
	GOCACHE=$(GOCACHE) go mod tidy

generate:
	buf generate

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

docker-build:
	docker build -t $(APP):latest .

compose-up:
	docker compose up --build

compose-down:
	docker compose down -v

clean:
	Remove-Item -Recurse -Force $(BIN_DIR), tmp -ErrorAction SilentlyContinue
	Remove-Item -Force coverage.out -ErrorAction SilentlyContinue
