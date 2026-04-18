# analytics-service

Production-oriented analytics microservice for ingesting user interactions and aggregating profile metrics.

## Features
- Idempotent raw event ingestion (`event_id` unique key).
- gRPC API for ingestion and analytics queries used by `api-gateway` (`GetAnimalStats`).
- Kafka publishing for ranking feedback (`analytics.ranking_feedback.v1`) consumed by feed-service.
- Aggregation by hourly/daily buckets.
- Extended stats with entitlement logic (`owner` / `shelter`).
- Prometheus metrics, request-id, structured logs, rate limiting, graceful shutdown.

## Local run
```bash
make tidy
make test
docker compose up --build
```

## API
- gRPC: `proto/petmatch/analytics/v1/analytics.proto`
- HTTP (ops only): health/readiness/metrics endpoints.
- OpenAPI spec (ops and compatibility): `api/openapi.yaml`.
