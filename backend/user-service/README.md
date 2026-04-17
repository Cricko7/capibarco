# user-service

Production-ready Go microservice for user/shelter profiles, ratings and reviews.

## Features
- Clean/Hex architecture with domain, app, infra and delivery layers
- gRPC API by `proto/petmatch/user/v1/user.proto`
- Kafka events for profile/review updates
- PostgreSQL storage and SQL migrations
- HTTP operational endpoints (`/healthz`, `/readyz`, `/metrics`)
- Structured slog logs with request IDs
- Rate limiting for gRPC, retry + circuit breaker for Kafka publisher

## Run locally
```bash
make tidy
make build
make test
make run
```

## gRPC contract
See `proto/petmatch/user/v1/user.proto`.
