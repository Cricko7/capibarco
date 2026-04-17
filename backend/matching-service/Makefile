SHELL := /bin/sh

BINARY := matching-service
BUILD_DIR := bin
GO_ENV := GOTELEMETRY=off

.PHONY: build test test-race test-integration run tidy fmt lint migrate-up migrate-down docker-build proto clean

build:
	$(GO_ENV) go build -trimpath -o $(BUILD_DIR)/$(BINARY) ./cmd/matching-service

test:
	$(GO_ENV) go test -race -coverprofile=coverage.out ./...

test-race:
	$(GO_ENV) go test -race ./...

test-integration:
	$(GO_ENV) go test -tags=integration ./...

run:
	$(GO_ENV) go run ./cmd/matching-service -config configs/config.yaml

tidy:
	$(GO_ENV) go mod tidy

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

migrate-up:
	migrate -path migrations -database "$$MATCHING_DB_DSN" up

migrate-down:
	migrate -path migrations -database "$$MATCHING_DB_DSN" down

docker-build:
	docker build -t petmatch/matching-service:local .

proto:
	cd .. && buf generate proto --template matching-service/buf.gen.yaml --path proto/petmatch/common/v1/common.proto --path proto/petmatch/matching/v1/matching.proto --path proto/petmatch/animal/v1/animal.proto --path proto/petmatch/chat/v1/chat.proto -o matching-service

clean:
	rm -rf $(BUILD_DIR) coverage.out
