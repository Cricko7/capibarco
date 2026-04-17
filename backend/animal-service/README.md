# animal-service

Production-oriented Go microservice for animal profiles, photos, publishing state, search, and ownership-scoped mutations.

## Local Run

```bash
docker compose up -d postgres kafka zookeeper migrate
go run ./cmd/animal-service -config configs/config.yaml
```

Operational HTTP endpoints:

- `GET /healthz`
- `GET /readyz`
- `GET /metrics`
- `GET /openapi.yaml`

Business API is gRPC: `petmatch.animal.v1.AnimalService` on `0.0.0.0:9090`.

## Development

```bash
make proto
make test
make test-race
make build
```

Use `X-Actor-ID` gRPC metadata for owner-scoped mutations.
