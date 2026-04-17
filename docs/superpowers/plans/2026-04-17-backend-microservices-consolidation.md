# Backend Microservices Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate the existing backend microservices into `backend/` with shared orchestration, proto files, and migrations.

**Architecture:** Preserve each service as an independent Go module under `backend/<service>`. Add root-level Docker Compose orchestration, central proto sources, central migration copies, and PostgreSQL initialization for per-service databases.

**Tech Stack:** Go 1.25 modules, Docker Compose, PostgreSQL, Redpanda Kafka-compatible broker, migrate/migrate.

---

### Task 1: Create Backend Service Directories

**Files:**
- Create: `backend/auth-service`
- Create: `backend/animal-service`
- Create: `backend/billing-service`
- Create: `backend/chat-service`
- Create: `backend/feed-service`

- [ ] Copy `D:\Programming\Hackathon` into `backend/auth-service`.
- [ ] Copy each `D:\Programming\Back-architecture\<name>-service` into the matching `backend/<name>-service`.
- [ ] Exclude build and dependency caches: `.cache`, `.gocache`, `.gomodcache`, `.bufcache`, `bin`, and archive files.

### Task 2: Merge Shared Proto Files

**Files:**
- Create: `backend/proto`

- [ ] Copy `D:\Programming\Back-architecture\proto` into `backend/proto`.
- [ ] Copy `D:\Programming\Hackathon\proto` into `backend/proto`, preserving its `auth/v1` package path.
- [ ] Preserve service-local proto folders for compatibility.

### Task 3: Merge Migration Files

**Files:**
- Create: `backend/migrations/auth-service`
- Create: `backend/migrations/animal-service`
- Create: `backend/migrations/billing-service`
- Create: `backend/migrations/chat-service`
- Create: `backend/migrations/feed-service`

- [ ] Copy each service's migrations into its canonical central directory.
- [ ] Keep original service-local migrations in place.
- [ ] Do not run `feed-service` through `migrate/migrate` until its single SQL file is converted into up/down pairs.

### Task 4: Add Common Docker Runtime

**Files:**
- Create: `backend/docker/postgres/initdb.d/001-create-service-databases.sql`
- Create: `backend/docker-compose.yml`
- Create: `backend/.dockerignore`
- Create: `backend/README.md`

- [ ] Add PostgreSQL init SQL that creates separate service users and databases.
- [ ] Add one Redpanda broker shared by all services.
- [ ] Add migration containers for auth, animal, billing, and chat.
- [ ] Add service containers with non-conflicting host ports.
- [ ] Document startup with `docker compose up --build`.

### Task 5: Verify Structure

**Files:**
- Read: `backend/docker-compose.yml`
- Read: `backend/*-service/go.mod`

- [ ] Confirm all expected service directories exist.
- [ ] Confirm central proto and migrations directories exist.
- [ ] Run `git status --short` from `D:\Programming\capibarco`.
- [ ] Run `go test ./...` inside each service when local dependencies are available.
