# api-gateway

Mobile BFF and edge gateway for PetMatch.

## Responsibilities

- REST/JSON facade for mobile clients.
- JWT validation and RBAC delegation through auth-service.
- Guest sessions for anonymous feed browsing.
- Redis-backed rate limiting by IP, actor, and role.
- gRPC orchestration for auth, animal, feed, matching, chat, billing, analytics, and future notification services.
- WebSocket chat bridge to chat-service bidirectional streaming.
- Optional SSE bridge to notification-service server streaming.
- Kafka operational events for rejected requests and WebSocket lifecycle.
- Prometheus metrics, health checks, structured slog logging, and graceful shutdown.

## Local Run

```powershell
go mod tidy
go test ./...
go run ./cmd/api-gateway
```

OpenAPI is served at `/openapi.yaml`.
