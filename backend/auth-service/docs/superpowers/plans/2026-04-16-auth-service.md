# Auth Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-oriented multi-tenant Go auth microservice with gRPC APIs, secure token handling, PostgreSQL persistence, metrics, health checks, Docker, migrations, and README.

**Architecture:** Clean/Hexagonal architecture separates domain models, usecases, ports, infrastructure adapters, and delivery. PostgreSQL uses shared tables with `tenant_id` for operational simplicity and can evolve to schema/database isolation for enterprise tenants.

**Tech Stack:** Go 1.23+, gRPC, PostgreSQL via pgx stdlib, Argon2id, Ed25519 JWT, slog JSON logging, Prometheus, goose-compatible SQL migrations, Docker.

---

### Task 1: Security Core

**Files:**
- Create: `internal/domain/errors.go`
- Create: `internal/domain/models.go`
- Create: `internal/ports/ports.go`
- Create: `internal/adapters/hasher/argon2id.go`
- Create: `internal/adapters/jwt/issuer.go`
- Create: `internal/usecase/token.go`
- Test: `internal/adapters/hasher/argon2id_test.go`
- Test: `internal/adapters/jwt/issuer_test.go`
- Test: `internal/usecase/auth_test.go`

- [x] Write failing tests for password hashing, JWT validation, and refresh token reuse detection.
- [x] Run tests and observe missing implementation failures.
- [x] Implement Argon2id, Ed25519 JWT, domain models, ports, and refresh token rotation.
- [x] Run tests and verify the security core passes.

### Task 2: Application Usecases

**Files:**
- Create: `internal/usecase/auth.go`

- [x] Implement register, login, refresh, forgot password, reset password, token validation, user info, and authorize flows.
- [x] Keep brute-forceable operations behind rate limiter ports.
- [x] Emit audit events for sensitive operations.

### Task 3: Infrastructure

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/adapters/postgres/repository.go`
- Create: `internal/adapters/ratelimit/memory.go`
- Create: `internal/adapters/audit/slog.go`
- Create: `internal/delivery/http/server.go`

- [x] Load configuration from environment variables.
- [x] Implement PostgreSQL repositories for users, refresh tokens, reset tokens, RBAC, and audit.
- [x] Add in-memory rate limiting and HTTP health/metrics server.

### Task 4: gRPC Contract and Server

**Files:**
- Create: `proto/auth/v1/auth.proto`
- Create: `internal/delivery/grpc/types.go`
- Create: `internal/delivery/grpc/server.go`

- [x] Define the full auth service proto contract.
- [x] Implement gRPC handlers that call usecases and map errors to gRPC status codes.
- [x] Register gRPC health checking.

### Task 5: Runtime, Migrations, Docs

**Files:**
- Create: `cmd/authsvc/main.go`
- Create: `migrations/00001_init.sql`
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.dockerignore`
- Create: `README.md`

- [x] Wire dependencies and graceful shutdown.
- [x] Add goose-compatible migrations.
- [x] Add Docker files and usage docs.
- [x] Run `gofmt`, `go test ./...`, and `go build ./cmd/authsvc`.
