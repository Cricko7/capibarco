# analytics-service

Production-oriented analytics microservice for ingesting user interactions via Kafka and serving analytics via gRPC.

## Transport
- **Kafka**: consumes raw events from `analytics.events.raw`, publishes ranking feedback to `analytics.ranking.feedback`.
- **gRPC**: serves query API (`GetMetrics`, `GetExtendedStats`, `GetRankingFeedback`) and gRPC health checks.

## Local run
```bash
make tidy
make test
docker compose up --build
```

## Contract
- Proto file: `api/analytics.proto`
- Generated gRPC stubs: `internal/delivery/grpc/pb`
