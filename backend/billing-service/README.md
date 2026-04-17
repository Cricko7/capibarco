# billing-service

Production-shaped Go 1.25 billing microservice for PetMatch donations, boosts, entitlements, and immutable billing ledger.

## Interfaces

- gRPC: `petmatch.billing.v1.BillingService` from `proto/petmatch/billing/v1/billing.proto`
- HTTP operations: `/healthz`, `/readyz`, `/metrics`, `/openapi.yaml`

## Local Run

```powershell
docker compose up -d postgres
docker compose up -d kafka kafka-init
goose -dir migrations postgres "postgres://billing:billing@localhost:5432/billing?sslmode=disable" up
go run ./cmd/billing-service -config configs/config.yaml
```

## Security Notes

- Payment integration is intentionally mocked.
- Billing domain events are published to Kafka as protobuf messages using the contracts in `billing.proto`.
- Card, bank, and token data are not accepted or persisted.
- Idempotency keys are SHA-256 hashed before storage.
- Logs omit `client_secret` and provider secrets.
- PostgreSQL access uses parameterized `pgx` queries.

## Development

```powershell
make generate
make test
make test-race
make vet
make build
```

For a local run without Kafka, set `BILLING_EVENTS_PUBLISHER=log`.

Kafka topics created by `kafka-init`:

- `billing.donation_succeeded`
- `billing.donation_failed`
- `billing.boost_activated`
- `billing.boost_expired`
- `billing.entitlement_granted`
- `billing.payment_failed`
