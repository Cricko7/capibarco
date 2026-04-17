# feed-service

`feed-service` builds and serves the animal swipe feed. It owns feed filtering, boost-aware ranking, served-card lookup, card-open telemetry, and Kafka-driven projection updates from animal, matching, billing, and analytics services.

## What It Does

- Serves `petmatch.feed.v1.FeedService` over gRPC.
- Filters unavailable, hidden, blocked, excluded, and already-swiped animals.
- Applies paid advanced filters only when entitlement state allows it.
- Interleaves boosted profiles with organic results instead of replacing organic feed content.
- Publishes feed telemetry events to Kafka.
- Consumes upstream domain events and updates the in-memory feed projection.
- Can persist the feed projection in PostgreSQL instead of process memory.
- Exposes HTTP health endpoints for runtime checks.

## Layout

```text
feed-service/
  cmd/feed-service/          process entrypoint and wiring
  internal/feed/             core feed service, ports, ranking, contracts
  internal/adapters/kafka/   Kafka producer, consumer, payload decoding
  internal/adapters/memory/  in-memory projection, stores, test adapters
  internal/config/           environment config
  internal/server/           gRPC/HTTP server lifecycle and interceptors
  proto/                     source proto subset used by this service
  gen/go/                    generated protobuf and gRPC Go code
```

The core package depends on interfaces. Runtime dependencies live under `internal/adapters`, so replacing the in-memory projection with PostgreSQL/Redis/etc. should not require rewriting `internal/feed`.

## Runtime Config

| Variable | Default | Description |
| --- | --- | --- |
| `FEED_GRPC_ADDR` | `:8081` | gRPC listen address. |
| `FEED_HTTP_ADDR` | `:8080` | HTTP operational endpoint listen address. |
| `FEED_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout. |
| `FEED_STORAGE` | `memory` | Storage backend: `memory` or `postgres`. |
| `FEED_POSTGRES_DSN` | empty | PostgreSQL connection string. Required when `FEED_STORAGE=postgres`. |
| `FEED_POSTGRES_MAX_OPEN_CONNS` | `10` | Maximum open PostgreSQL connections. |
| `FEED_POSTGRES_MAX_IDLE_CONNS` | `5` | Maximum idle PostgreSQL connections. |
| `FEED_KAFKA_ENABLED` | `false` | Enables Kafka producer and consumer. |
| `FEED_KAFKA_BROKERS` | empty | Comma-separated Kafka brokers. Required when Kafka is enabled. |
| `FEED_KAFKA_CONSUMER_GROUP` | `feed-service` | Consumer group for inbound projection updates. |

HTTP endpoints:

- `GET /healthz`
- `GET /readyz`

## gRPC API

Implemented from `proto/petmatch/feed/v1/feed.proto`:

- `GetFeed`
- `StreamFeed`
- `GetFeedCard`
- `RecordCardOpen`
- `ExplainRanking`

## Kafka

Published events are protobuf messages:

| Topic | Partition key | Payload |
| --- | --- | --- |
| `feed.card_served` | `feed_session_id` | `petmatch.feed.v1.FeedCardServedEvent` |
| `feed.card_opened` | `animal_id` | `petmatch.feed.v1.FeedCardOpenedEvent` |
| `feed.filters_applied` | `actor_id` | `petmatch.feed.v1.FeedFiltersAppliedEvent` |

Consumed events update the local projection:

| Topic | Payload | Effect |
| --- | --- | --- |
| `animal.profile_published` | `petmatch.animal.v1.AnimalPublishedEvent` | Upserts animal profile into feed candidates. |
| `animal.profile_archived` | JSON archive payload | Removes animal from candidates and invalidates cached cards. |
| `animal.status_changed` | `petmatch.animal.v1.AnimalStatusChangedEvent` | Removes non-available animals or updates available status. |
| `matching.swipe_recorded` | `petmatch.matching.v1.SwipeRecordedEvent` | Suppresses swiped animal for the actor. |
| `billing.boost_activated` | `petmatch.billing.v1.BoostActivatedEvent` | Updates boost flag, expiration, and score component. |
| `billing.entitlement_granted` | `petmatch.billing.v1.EntitlementGrantedEvent` | Updates advanced-filter entitlement cache. |
| `analytics.animal_stats_aggregated` | `petmatch.analytics.v1.AnimalStatsAggregatedEvent` | Updates ranking signal components. |

`animal.profile_archived` uses this documented JSON shape:

```json
{
  "animal_id": "animal-123",
  "owner_profile_id": "owner-123",
  "previous_status": "ANIMAL_STATUS_AVAILABLE",
  "reason": "owner archived"
}
```

## Development

Generate protobuf code:

```powershell
buf generate
```

Run checks:

```powershell
go test ./...
go vet ./...
go build ./cmd/feed-service
```

Run locally without Kafka:

```powershell
go run ./cmd/feed-service
```

Run locally with Kafka:

```powershell
$env:FEED_KAFKA_ENABLED="true"
$env:FEED_KAFKA_BROKERS="localhost:9092"
go run ./cmd/feed-service
```

Run locally with PostgreSQL storage:

```powershell
$env:FEED_STORAGE="postgres"
$env:FEED_POSTGRES_DSN="postgres://feed:feed@localhost:5432/feed?sslmode=disable"
go run ./cmd/feed-service
```

## Current Storage

The default runtime uses an in-memory projection from `internal/adapters/memory`. For durable runs, set `FEED_STORAGE=postgres` to use `internal/adapters/postgres`; the service applies embedded SQL migrations from `internal/adapters/postgres/migrations` on startup.

Storage adapters implement these core interfaces from `internal/feed`:

- `Store`
- `SwipeStore`
- `EntitlementChecker`
- `EventApplier`
- `Publisher`
