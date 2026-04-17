SHELL := /bin/sh

APP := animal-service
VERSION ?= $(shell cat VERSION)
DATABASE_URL ?= postgres://animal:animal@localhost:5432/animal?sslmode=disable
GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/go-mod

.PHONY: help build run test test-race lint fmt tidy proto migrate-up migrate-down docker-build compose-up compose-down tag

help:
	@printf '%s\n' 'Targets: build run test test-race lint fmt tidy proto migrate-up migrate-down docker-build compose-up compose-down tag'

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go build -trimpath -o bin/$(APP) ./cmd/animal-service

run:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go run ./cmd/animal-service -config configs/config.yaml

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test ./...

test-race:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test -race ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w $$(find cmd internal gen -name '*.go')

tidy:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go mod tidy

proto:
	cd .. && buf generate proto --template animal-service/buf.gen.yaml -o animal-service --path proto/petmatch/common/v1/common.proto --path proto/petmatch/animal/v1/animal.proto --path proto/petmatch/billing/v1/billing.proto --path proto/petmatch/matching/v1/matching.proto

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

docker-build:
	docker build -t petmatch/animal-service:$(VERSION) .

tag:
	git tag v$(VERSION)

compose-up:
	docker compose up --build

compose-down:
	docker compose down -v
