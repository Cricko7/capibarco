.PHONY: generate fmt test vet build run

GO ?= go
BUF ?= buf

generate:
	$(BUF) generate

fmt:
	$(GO) fmt ./...

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

build:
	$(GO) build ./cmd/feed-service

run:
	$(GO) run ./cmd/feed-service
