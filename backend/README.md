# Backend

This directory contains the consolidated backend microservices.

## Services

- `auth-service` from `D:\Programming\Hackathon`
- `animal-service` from `D:\Programming\Back-architecture\animal-service`
- `billing-service` from `D:\Programming\Back-architecture\billing-service`
- `chat-service` from `D:\Programming\Back-architecture\chat-service`
- `feed-service` from `D:\Programming\Back-architecture\feed-service`

Each service remains an independent Go module with its own `go.mod` and `Dockerfile`.

## Shared Files

- `proto/` contains the merged proto sources.
- `migrations/<service>/` contains canonical migration copies for the common Docker stack.
- Service-local proto and migration folders are preserved for compatibility with existing service builds.

## Run Everything

From this directory:

```powershell
docker compose up --build
```

Host ports:

- Auth HTTP: `18080`, gRPC: `15051`
- Animal HTTP: `18081`, gRPC: `19090`
- Billing HTTP: `18082`, gRPC: `19091`
- Chat HTTP: `18083`, gRPC: `19092`
- Feed HTTP: `18084`, gRPC: `18085`
- PostgreSQL: `15432`
- Redpanda Kafka API: `19093`

The compose stack creates separate PostgreSQL databases for each service and runs the central migrations before starting the service containers.
