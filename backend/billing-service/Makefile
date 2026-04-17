SHELL := powershell.exe
.SHELLFLAGS := -NoProfile -Command

GOCACHE_DIR := $(CURDIR)\.gocache
GOMODCACHE_DIR := $(CURDIR)\.gomodcache
BUF_CACHE_DIR := $(CURDIR)\.bufcache
POSTGRES_DSN ?= postgres://billing:billing@localhost:5432/billing?sslmode=disable

.PHONY: generate tidy build test test-race vet lint run migrate-up migrate-down kafka-topics compose-up compose-down

generate:
	$$env:BUF_CACHE_DIR='$(BUF_CACHE_DIR)'; buf generate

tidy:
	$$env:GOCACHE='$(GOCACHE_DIR)'; $$env:GOMODCACHE='$(GOMODCACHE_DIR)'; go mod tidy

build:
	New-Item -ItemType Directory -Force bin | Out-Null; $$env:GOCACHE='$(GOCACHE_DIR)'; $$env:GOMODCACHE='$(GOMODCACHE_DIR)'; go build -o bin/billing-service.exe ./cmd/billing-service

test:
	$$env:GOCACHE='$(GOCACHE_DIR)'; $$env:GOMODCACHE='$(GOMODCACHE_DIR)'; go test ./...

test-race:
	$$env:GOCACHE='$(GOCACHE_DIR)'; $$env:GOMODCACHE='$(GOMODCACHE_DIR)'; go test -race ./internal/...

vet:
	$$env:GOCACHE='$(GOCACHE_DIR)'; $$env:GOMODCACHE='$(GOMODCACHE_DIR)'; go vet ./...

lint:
	golangci-lint run

run:
	$$env:GOCACHE='$(GOCACHE_DIR)'; $$env:GOMODCACHE='$(GOMODCACHE_DIR)'; go run ./cmd/billing-service -config configs/config.yaml

migrate-up:
	goose -dir migrations postgres "$(POSTGRES_DSN)" up

migrate-down:
	goose -dir migrations postgres "$(POSTGRES_DSN)" down

kafka-topics:
	docker compose run --rm kafka-init

compose-up:
	docker compose up --build

compose-down:
	docker compose down -v
