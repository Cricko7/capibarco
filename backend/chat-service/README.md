# chat-service

Production-ready PetMatch chat microservice for adoption conversations, messages, read state, and realtime streams.

## API

Business API is gRPC and follows `proto/petmatch/chat/v1/chat.proto`.

Operational HTTP endpoints:

- `GET /healthz`
- `GET /readyz`
- `GET /metrics`
- `GET /openapi.yaml`

## Local Run

```powershell
docker compose up -d postgres
migrate -path migrations -database "postgres://chat:chat@localhost:5432/chat?sslmode=disable" up
go run ./cmd/chat-service
```

## Configuration

Configuration is loaded from `configs/config.yaml` and `CHAT_*` environment variables. Example:

```powershell
$env:CHAT_POSTGRES_DSN = "postgres://chat:chat@localhost:5432/chat?sslmode=disable"
```

## Architecture

- `internal/domain/chat`: entities, invariants, errors, ports, events.
- `internal/application/chat`: use cases and transaction-independent orchestration.
- `internal/infrastructure/postgres`: PostgreSQL persistence.
- `internal/infrastructure/realtime`: in-memory event fanout for streams.
- `internal/delivery/grpc`: gRPC handlers and middleware.
- `internal/delivery/http`: health, readiness, metrics, and OpenAPI serving.

## Versioning

Current service version is `0.1.0`. Release tags should use semantic versioning, for example `chat-service/v0.1.0`.
