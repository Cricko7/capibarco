# notification-service

Go microservice for notification orchestration across in-app, push, email, and Kafka-driven domain events.

## Scope

- gRPC API from `proto/petmatch/notification/v1/notification.proto`
- PostgreSQL persistence for inbox items, device tokens, and notification preferences
- Kafka publish/subscribe integration for notification lifecycle and upstream business events
- HTTP operational endpoints: `/healthz`, `/readyz`, `/metrics`
- Graceful shutdown, request ID propagation, structured logging, rate limiting, retries, and circuit breaker protected publishing

## Assumptions

- `chat.message_sent` does not currently include recipients in the protobuf contract, so the consumer only creates notifications when `message.metadata` contains either `recipient_profile_id` or comma-separated `recipient_profile_ids`.
- `billing.donation_succeeded` notifies the donor unconditionally and also notifies a target owner only when the event envelope partition key contains a distinct owner profile id.
- Delivery is currently synchronous and local to this service: accepted channels are marked delivered immediately after persistence and event publication.

## Run locally

```bash
make tidy
make proto
make build
make test
make run
```

## Configuration

- File: `configs/config.yaml`
- Environment overrides: `NOTIFICATION_*`
- Important variables: `NOTIFICATION_DB_DSN`, `NOTIFICATION_KAFKA_ENABLED`, `NOTIFICATION_KAFKA_BROKERS`, `NOTIFICATION_HTTP_ADDR`, `NOTIFICATION_GRPC_ADDR`
