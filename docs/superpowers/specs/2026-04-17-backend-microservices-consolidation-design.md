# Backend Microservices Consolidation Design

## Goal

Move the existing Go microservices from `D:\Programming\Back-architecture` and `D:\Programming\Hackathon` into `D:\Programming\capibarco\backend` while preserving each service as an independent service directory.

## Target Structure

```text
backend/
  auth-service/
  animal-service/
  billing-service/
  chat-service/
  feed-service/
  proto/
  migrations/
    auth-service/
    animal-service/
    billing-service/
    chat-service/
    feed-service/
  docker/
    postgres/
      initdb.d/
  docker-compose.yml
  .dockerignore
  README.md
```

## Architecture

Each microservice remains a standalone Go module with its own `go.mod`, `Dockerfile`, `cmd`, and `internal` packages. The backend root owns local orchestration: a single `docker-compose.yml`, shared infrastructure services, shared proto sources, and a central migrations index.

The compose stack uses one PostgreSQL container with separate databases and users per service, plus one Redpanda Kafka-compatible broker. Per-service migration containers run before the matching application container when the migration format is compatible with `migrate/migrate`.

## Source Mapping

- `D:\Programming\Hackathon` becomes `backend/auth-service`.
- `D:\Programming\Back-architecture\animal-service` becomes `backend/animal-service`.
- `D:\Programming\Back-architecture\billing-service` becomes `backend/billing-service`.
- `D:\Programming\Back-architecture\chat-service` becomes `backend/chat-service`.
- `D:\Programming\Back-architecture\feed-service` becomes `backend/feed-service`.
- `D:\Programming\Back-architecture\proto` becomes `backend/proto`, with `Hackathon\proto` merged under `backend/proto`.

## Migration Strategy

Canonical migration copies live under `backend/migrations/<service>`. Service-local migration directories are preserved during the initial move so existing Dockerfiles and code do not break. The shared compose file mounts the canonical migration directories for migration containers.

`feed-service` currently uses a single `001_init.sql` file rather than paired `.up.sql` and `.down.sql` files, so its migration is copied into the central migration tree but not run by `migrate/migrate` until it is converted.

## Docker Strategy

The root compose file builds each service from its own directory and assigns unique host ports:

- auth: HTTP `18080`, gRPC `15051`
- animal: HTTP `18081`, gRPC `19090`
- billing: HTTP `18082`, gRPC `19091`
- chat: HTTP `18083`, gRPC `19092`
- feed: service ports `18084`, `18085`

Internal container ports stay close to each service's existing defaults to minimize code/config changes.

## Risks

- Some service Dockerfiles may assume local proto or migration paths. The first migration keeps local copies to reduce this risk.
- `feed-service` migration format needs a follow-up conversion before automated migration in compose.
- Go module paths are inconsistent across services. Keeping modules independent avoids a risky import rewrite during consolidation.
