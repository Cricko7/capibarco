# analytics-service

Production-oriented analytics microservice for ingesting user interactions and aggregating profile metrics.

## Features
- Idempotent raw event ingestion.
- Aggregation by hourly/daily buckets.
- Extended stats endpoint with entitlement check (`owner` / `shelter`).
- Ranking feedback endpoint for feed-service.
- Prometheus metrics, request-id, structured logs, rate limiting, graceful shutdown.

## Local run
```bash
make tidy
make test
docker compose up --build
```

## API
OpenAPI spec: `api/openapi.yaml`.
