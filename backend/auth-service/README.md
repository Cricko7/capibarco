# Auth Service

Production-oriented multi-tenant Auth microservice on Go 1.25+.

## What Is Included

- gRPC API for `Register`, `Login`, `RefreshToken`, `ForgotPassword`, `ResetPassword`, `ValidateToken`, `GetUserInfo`, and `Authorize`.
- Clean/Hexagonal architecture: domain, ports, usecases, adapters, delivery.
- PostgreSQL persistence with shared tables and `tenant_id` scoping.
- Argon2id password hashing.
- Ed25519 JWT access tokens.
- Opaque refresh tokens stored as SHA-256 hashes with rotation and reuse detection.
- One-time password reset tokens stored as hashes.
- Tenant-scoped RBAC with permissions like `system:resource:action`.
- JSON structured logs through `log/slog`.
- Audit log persistence for sensitive operations.
- Prometheus metrics on `/metrics`.
- HTTP health and readiness checks on `/healthz` and `/readyz`.
- gRPC health checking.
- Kafka publishing for auth domain events.
- Graceful shutdown on `SIGINT` and `SIGTERM`.
- Dockerfile and docker-compose.

## Multi-Tenancy Choice

This service uses a shared PostgreSQL database with `tenant_id` in all tenant-scoped tables.

That is the recommended default because it keeps migrations, connection pooling, backup/restore, analytics, audit, and operational tooling simple. For tenants that need stronger isolation, the same domain/usecase boundaries can be extended with schema-per-tenant or database-per-tenant repository implementations.

## Project Layout

```text
cmd/authsvc                  application entrypoint
internal/domain              entities and domain errors
internal/ports               repository, token, audit, mailer, event interfaces
internal/usecase             auth workflows
internal/adapters/hasher     Argon2id password hashing
internal/adapters/jwt        Ed25519 JWT issuer
internal/adapters/kafka      Kafka publisher for auth events
internal/adapters/postgres   PostgreSQL repositories
internal/delivery/grpc       gRPC handlers
internal/delivery/http       health and metrics
proto/auth/v1/auth.proto     public gRPC contract
migrations                   goose-compatible SQL migrations
```

## Configuration

Configuration is loaded with Viper from `configs/config.yaml` + environment variables (env overrides file).

| Variable | Default | Description |
| --- | --- | --- |
| `GRPC_ADDR` | `:50051` | gRPC listen address |
| `HTTP_ADDR` | `:8080` | health/metrics listen address |
| `DATABASE_URL` | `postgres://auth:auth@localhost:5432/auth?sslmode=disable` | PostgreSQL DSN |
| `JWT_ISSUER` | `authsvc` | JWT issuer |
| `JWT_AUDIENCE` | `internal-services` | JWT audience |
| `JWT_KEY_ID` | `local-dev` | JWT `kid` header |
| `JWT_ED25519_PRIVATE_KEY_B64` | generated in memory | Base64 Ed25519 private key |
| `JWT_ED25519_PUBLIC_KEY_B64` | generated in memory | Base64 Ed25519 public key |
| `ACCESS_TTL` | `15m` | Access token lifetime |
| `REFRESH_TTL` | `720h` | Refresh token lifetime |
| `RESET_TTL` | `15m` | Password reset token lifetime |
| `ARGON2_MEMORY_KIB` | `131072` | Argon2id memory cost |
| `ARGON2_ITERATIONS` | `3` | Argon2id iterations |
| `ARGON2_PARALLELISM` | `4` | Argon2id parallelism |
| `KAFKA_ENABLED` | `false` | Enables publishing auth events to Kafka |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated Kafka bootstrap brokers |
| `KAFKA_CLIENT_ID` | `auth-service` | Kafka client id |
| `KAFKA_ALLOW_AUTO_TOPIC_CREATION` | `false` | Allows topic auto-creation; useful for local Docker only |

For production, provide stable Ed25519 keys. If they are omitted, the service generates ephemeral development keys on startup and all existing access tokens become invalid after restart.

## Generate Ed25519 Keys

```bash
go test ./internal/adapters/jwt -run TestEd25519IssuerCreatesAndValidatesAccessToken
```

For real deployments, generate keys through your secret-management process and store them as base64 values in your secret store.

## Database Migrations

Install goose:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Run migrations:

```bash
goose -dir migrations postgres "postgres://auth:auth@localhost:5432/auth?sslmode=disable" up
```

The first migration creates a `default` tenant.

## Run Locally

Start PostgreSQL:

```bash
docker compose up -d postgres
```

Run migrations:

```bash
goose -dir migrations postgres "postgres://auth:auth@localhost:5432/auth?sslmode=disable" up
```

Run the service:

```bash
go run ./cmd/authsvc
```

Health check:

```bash
curl http://localhost:8080/healthz
```

Metrics:

```bash
curl http://localhost:8080/metrics
```

## Docker

```bash
docker compose up --build
```

Run migrations before starting traffic. In production, run migrations as a separate deployment job instead of inside the application container.

## Kafka Events

`auth-service` publishes the events described in `D:\Programming\Back-architecture\docs\contracts\services\auth-service.md`.

Published topics:

| Topic | Partition key | When published |
| --- | --- | --- |
| `auth.user_registered` | `user.id` | After successful registration |
| `auth.user_logged_in` | `user.id` | After successful login |
| `auth.token_refreshed` | `user_id` | After successful refresh token rotation |
| `auth.password_reset_requested` | `email` | After a password reset token is created |
| `auth.password_reset_completed` | `user_id` | After password reset token consumption and password update |
| `auth.permission_denied` | `subject` | When `Authorize` denies a permission |

Auth does not consume Kafka events on the critical authentication path.

Every event uses this envelope:

```json
{
  "event_id": "c99fbe8f-5c8c-46dd-b8b8-d481dd7f031a",
  "event_type": "auth.user_registered",
  "schema_version": "1",
  "occurred_at": "2026-04-17T10:00:00Z",
  "producer": "auth-service",
  "trace_id": "trace-123",
  "correlation_id": "corr-123",
  "idempotency_key": "register-default-alice@example.com",
  "payload": {}
}
```

The gRPC layer propagates event metadata from incoming headers:

```text
x-trace-id
x-correlation-id
x-idempotency-key
```

For local Docker, Kafka is provided by Redpanda and topic auto-creation is enabled. For production, pre-create topics with adequate partition counts, replication factor, retention, ACLs, and monitoring.

### Kafka Payload Examples

`auth.user_registered`:

```json
{
  "user": {
    "id": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
    "tenant_id": "default",
    "email": "alice@example.com",
    "is_active": true,
    "created_at": "2026-04-17T10:00:00Z",
    "updated_at": "2026-04-17T10:00:00Z"
  },
  "tenant_id": "default",
  "roles": ["User"],
  "registration_ip": "203.0.113.10"
}
```

`auth.user_logged_in`:

```json
{
  "user_id": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
  "tenant_id": "default",
  "email": "alice@example.com",
  "token_id": "1776420000000000000",
  "roles": ["User"],
  "ip": "203.0.113.10",
  "user_agent": "petmatch-api/1.0"
}
```

`auth.token_refreshed`:

```json
{
  "user_id": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
  "tenant_id": "default",
  "old_token_id": "refresh-token-id-old",
  "new_token_id": "refresh-token-id-new",
  "expires_at": "2026-04-17T10:15:00Z"
}
```

`auth.password_reset_requested`:

```json
{
  "tenant_id": "default",
  "email": "alice@example.com",
  "reset_token_id": "reset-token-id",
  "expires_at": "2026-04-17T10:15:00Z",
  "ip": "203.0.113.10"
}
```

`auth.password_reset_completed`:

```json
{
  "user_id": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
  "tenant_id": "default",
  "email": "alice@example.com",
  "reset_token_id": "reset-token-id",
  "ip": "203.0.113.10"
}
```

`auth.permission_denied`:

```json
{
  "subject": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
  "tenant_id": "default",
  "permission": "billing:invoice:read",
  "roles": ["User"],
  "token_id": "1776420000000000000"
}
```

## Proto

The contract lives in:

```text
proto/auth/v1/auth.proto
```

Generate standard Go stubs:

```bash
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/auth/v1/auth.proto
```

The current server registers a manual gRPC service with a JSON codec so the skeleton compiles without checked-in generated protobuf files. For a production client SDK workflow, generate and publish protobuf stubs from `proto/auth/v1/auth.proto` in CI.

## Request And Response Examples

Examples below use JSON field names from `proto/auth/v1/auth.proto`. Tokens are shortened for readability.

If you use generated clients, call the same RPC methods with the corresponding protobuf messages. If you want to use `grpcurl`, generate/register standard protobuf stubs and enable server reflection first, or pass the proto file explicitly from a standard protobuf-based server build.

### Register

Request:

```json
{
  "tenant_id": "default",
  "email": "alice@example.com",
  "password": "CorrectHorseBatteryStaple!",
  "ip": "203.0.113.10"
}
```

Successful response:

```json
{
  "user": {
    "id": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
    "tenant_id": "default",
    "email": "alice@example.com",
    "is_active": true,
    "created_at": "2026-04-16T18:30:00Z",
    "updated_at": "2026-04-16T18:30:00Z"
  },
  "access_token": "eyJhbGciOiJFZERTQSIsImtpZCI6ImxvY2FsLWRldiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "oIzn9AcQzA1tMsuRKn5tO4KfXucflEr1rwT1STnqNL8",
  "expires_at": "2026-04-16T18:45:00Z"
}
```

Possible errors:

```json
{
  "code": "AlreadyExists",
  "message": "already exists"
}
```

```json
{
  "code": "InvalidArgument",
  "message": "weak password"
}
```

### Login

Request:

```json
{
  "tenant_id": "default",
  "email": "alice@example.com",
  "password": "CorrectHorseBatteryStaple!",
  "ip": "203.0.113.10"
}
```

Successful response:

```json
{
  "user": {
    "id": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
    "tenant_id": "default",
    "email": "alice@example.com",
    "is_active": true,
    "created_at": "2026-04-16T18:30:00Z",
    "updated_at": "2026-04-16T18:30:00Z"
  },
  "access_token": "eyJhbGciOiJFZERTQSIsImtpZCI6ImxvY2FsLWRldiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "fGyfJvJwZPUXhcn8H7RKNMr_a1LHhzaU--mXiwCsIVQ",
  "expires_at": "2026-04-16T18:47:00Z"
}
```

Invalid credentials response:

```json
{
  "code": "Unauthenticated",
  "message": "invalid credentials"
}
```

### RefreshToken

Request:

```json
{
  "refresh_token": "fGyfJvJwZPUXhcn8H7RKNMr_a1LHhzaU--mXiwCsIVQ"
}
```

Successful response:

```json
{
  "user": {
    "id": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
    "tenant_id": "default",
    "email": "alice@example.com",
    "is_active": true,
    "created_at": "2026-04-16T18:30:00Z",
    "updated_at": "2026-04-16T18:30:00Z"
  },
  "access_token": "eyJhbGciOiJFZERTQSIsImtpZCI6ImxvY2FsLWRldiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "WkyzQrVpXtFyUJ6EVvtHRDzTtmUNOdR7eDI7RiQWrAI",
  "expires_at": "2026-04-16T19:00:00Z"
}
```

Replay detection response when the old refresh token is used again:

```json
{
  "code": "Unauthenticated",
  "message": "refresh token reused"
}
```

When reuse is detected, the service revokes the whole refresh-token family.

### ForgotPassword

Request:

```json
{
  "tenant_id": "default",
  "email": "alice@example.com",
  "ip": "203.0.113.10"
}
```

Successful response:

```json
{}
```

For account enumeration resistance, this endpoint also returns success when the email does not exist. In the development adapter, the reset token is written to structured logs. Replace `LogMailer` with an SMTP or provider adapter in production.

### ResetPassword

Request:

```json
{
  "tenant_id": "default",
  "reset_token": "s9omkMSEpOnZf_h18uxOUd1mBk0vG_9fyLgAduUEC6k",
  "new_password": "AnotherCorrectHorseBatteryStaple!",
  "ip": "203.0.113.10"
}
```

Successful response:

```json
{}
```

Expired token response:

```json
{
  "code": "Unauthenticated",
  "message": "token expired"
}
```

### ValidateToken

Request:

```json
{
  "access_token": "eyJhbGciOiJFZERTQSIsImtpZCI6ImxvY2FsLWRldiIsInR5cCI6IkpXVCJ9..."
}
```

Successful response:

```json
{
  "valid": true,
  "subject": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
  "tenant_id": "default",
  "email": "alice@example.com",
  "roles": ["admin"],
  "permissions": ["billing:invoice:read", "billing:invoice:write"],
  "expires_at": "2026-04-16T18:45:00Z",
  "token_id": "1776364200000000000"
}
```

Invalid token response:

```json
{
  "code": "Unauthenticated",
  "message": "invalid token"
}
```

### GetUserInfo

Request:

```json
{
  "access_token": "eyJhbGciOiJFZERTQSIsImtpZCI6ImxvY2FsLWRldiIsInR5cCI6IkpXVCJ9..."
}
```

Successful response:

```json
{
  "user": {
    "id": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
    "tenant_id": "default",
    "email": "alice@example.com",
    "is_active": true,
    "created_at": "2026-04-16T18:30:00Z",
    "updated_at": "2026-04-16T18:30:00Z"
  }
}
```

### Authorize

Before this call can return `allowed: true`, create tenant-scoped roles, permissions, role-permission links, and user-role links in PostgreSQL.

Request:

```json
{
  "access_token": "eyJhbGciOiJFZERTQSIsImtpZCI6ImxvY2FsLWRldiIsInR5cCI6IkpXVCJ9...",
  "permission": "billing:invoice:read"
}
```

Successful response:

```json
{
  "allowed": true,
  "claims": {
    "valid": true,
    "subject": "4f6fd4df-24ef-4e41-9dd2-b3c7c216f931",
    "tenant_id": "default",
    "email": "alice@example.com",
    "roles": ["admin"],
    "permissions": ["billing:invoice:read", "billing:invoice:write"],
    "expires_at": "2026-04-16T18:45:00Z",
    "token_id": "1776364200000000000"
  }
}
```

Permission denied response:

```json
{
  "code": "PermissionDenied",
  "message": "permission denied"
}
```

### RBAC Seed Example

Use this SQL after migrations to create a simple admin role for the `default` tenant. Replace `USER_ID_FROM_REGISTER_RESPONSE` with the registered user's id.

```sql
insert into roles (id, tenant_id, name)
values ('role-admin', 'default', 'admin')
on conflict do nothing;

insert into permissions (id, tenant_id, value)
values
  ('perm-invoice-read', 'default', 'billing:invoice:read'),
  ('perm-invoice-write', 'default', 'billing:invoice:write')
on conflict do nothing;

insert into role_permissions (tenant_id, role_id, permission_id)
values
  ('default', 'role-admin', 'perm-invoice-read'),
  ('default', 'role-admin', 'perm-invoice-write')
on conflict do nothing;

insert into user_roles (tenant_id, user_id, role_id)
values ('default', 'USER_ID_FROM_REGISTER_RESPONSE', 'role-admin')
on conflict do nothing;
```

## Security Notes

- Access tokens are short-lived JWTs signed with Ed25519.
- Refresh tokens are opaque random values. Only SHA-256 hashes are stored in PostgreSQL.
- Refresh token rotation revokes the used token. A second use of the same token revokes the whole token family.
- Password reset tokens are opaque, one-time, short-lived, and stored hashed.
- Request rate limiting is enforced at the gateway layer.
- Sensitive operations are written to `audit_logs`.
- RBAC is tenant-scoped. Permissions use `system:resource:action`, for example `billing:invoice:read`.
- ABAC can be added behind the `RBACRepository`/authorization boundary by introducing a policy evaluator without changing delivery contracts.

## Test And Build

```bash
go test ./...
go build ./cmd/authsvc
```

On constrained environments, point Go cache directories into the workspace:

```bash
GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache GOTELEMETRY=off go test ./...
```
