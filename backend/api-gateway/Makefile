.PHONY: build run test test-race lint tidy gen migrate docker

GOCACHE ?= ../../.gocache
GOPATH ?= ../../.gopath

build:
	GOCACHE=$(GOCACHE) GOPATH=$(GOPATH) go build ./cmd/api-gateway

run:
	GOCACHE=$(GOCACHE) GOPATH=$(GOPATH) go run ./cmd/api-gateway

test:
	GOCACHE=$(GOCACHE) GOPATH=$(GOPATH) go test ./...

test-race:
	GOCACHE=$(GOCACHE) GOPATH=$(GOPATH) go test -race ./...

lint:
	golangci-lint run ./...

tidy:
	GOCACHE=$(GOCACHE) GOPATH=$(GOPATH) go mod tidy

gen:
	buf generate proto/petmatch/notification/v1/notification.proto

migrate:
	@echo "api-gateway has no database migrations"

docker:
	docker build -t petmatch/api-gateway:local .
