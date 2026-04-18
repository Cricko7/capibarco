# api-gateway

Mobile BFF and edge gateway for PetMatch. It exposes public REST/JSON, multipart upload, SSE, and WebSocket endpoints, then proxies calls to internal gRPC services.

## Base URL

Local Docker Compose: `http://localhost:18088`.

OpenAPI: `GET /openapi.yaml`.

## Common Rules

Public endpoints:

- `GET /healthz`
- `GET /readyz`
- `GET /metrics`
- `GET /openapi.yaml`
- `POST /v1/auth/guest-sessions`
- `POST /v1/auth/register`
- `POST /v1/auth/login`

All other endpoints require one of:

```http
Authorization: Bearer <access_token>
X-Guest-Session-Token: <guest_session_token>
```

JSON requests use `Content-Type: application/json`.

Mutating endpoints should send `Idempotency-Key: <unique-operation-key>`. `X-Idempotency-Key` is also accepted.

The gateway returns `X-Request-ID`. Clients may send `X-Request-ID` explicitly.

Errors are returned as problem JSON:

```json
{
  "type": "https://api.petmatch.local/errors/bad-request",
  "title": "Bad Request",
  "status": 400,
  "detail": "invalid input: email is required",
  "instance": "/v1/auth/register"
}
```

## Endpoint Summary

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/healthz` | no | Liveness check. |
| `GET` | `/readyz` | no | Readiness check; verifies Redis/rate limiter dependency. |
| `GET` | `/metrics` | no | Prometheus metrics. |
| `GET` | `/openapi.yaml` | no | OpenAPI document. |
| `POST` | `/v1/auth/guest-sessions` | no | Create a guest session token. |
| `POST` | `/v1/auth/register` | no | Register user and return auth tokens. |
| `POST` | `/v1/auth/login` | no | Authenticate user and return auth tokens. |
| `GET` | `/v1/feed` | yes | Get feed cards. |
| `GET` | `/v1/animals/{animal_id}` | yes | Get animal profile. |
| `POST` | `/v1/animals` | yes | Create animal profile. |
| `POST` | `/v1/animals/{animal_id}/photos` | yes | Upload animal photo. |
| `POST` | `/v1/animals/{animal_id}/swipe` | yes | Swipe animal. |
| `POST` | `/v1/animals/{animal_id}:swipe` | yes | Contract alias for swipe. |
| `GET` | `/v1/animals/{animal_id}/stats` | yes | Get animal analytics. |
| `GET` | `/v1/profiles` | yes | Search profiles. |
| `GET` | `/v1/profiles/{profile_id}` | yes | Get profile. |
| `PATCH` | `/v1/profiles/{profile_id}` | yes | Create or update profile fields. |
| `GET` | `/v1/profiles/{profile_id}/reviews` | yes | List profile reviews. |
| `POST` | `/v1/profiles/{profile_id}/reviews` | yes | Create review for profile. |
| `GET` | `/v1/profiles/{profile_id}/reputation` | yes | Get reputation summary. |
| `PATCH` | `/v1/reviews/{review_id}` | yes | Update review. |
| `GET` | `/v1/chat/conversations` | yes | List chat conversations. |
| `GET` | `/v1/chat/conversations/{conversation_id}/messages` | yes | List conversation messages. |
| `POST` | `/v1/chat/conversations/{conversation_id}/messages` | yes | Send chat message. |
| `GET` | `/ws/chat` | yes | WebSocket bridge to chat-service. |
| `POST` | `/v1/billing/donation-intents` | yes | Create donation payment intent. |
| `POST` | `/v1/notifications/devices` | yes | Register push device token. |
| `DELETE` | `/v1/notifications/devices/{device_token_id}` | yes | Unregister push device token. |
| `GET` | `/v1/notifications` | yes | List notifications. |
| `GET` | `/v1/notifications/stream` | yes | SSE notification stream. |
| `POST` | `/v1/notifications/{notification_id}/read` | yes | Mark notification as read. |
| `POST` | `/v1/notifications/{notification_id}:read` | yes | Contract alias for mark-read. |

## Operations

### GET /healthz

Returns `200` when the HTTP process is alive.

```json
{"status":"ok"}
```

### GET /readyz

Returns `200` when required runtime dependencies are ready.

```json
{"status":"ready"}
```

Returns `503` when Redis/rate limiter ping fails.

### GET /metrics

Returns Prometheus text metrics.

### GET /openapi.yaml

Returns the OpenAPI YAML bundled with the gateway.

## Auth

### POST /v1/auth/guest-sessions

Creates a guest token for anonymous read flows.

Request:

```json
{
  "device_id": "ios-device-123",
  "locale": "ru-RU"
}
```

Response `201`:

```json
{
  "guest_session_token": "<token>",
  "expires_at": "2026-04-19T06:00:00Z",
  "allowed_scopes": ["feed:read", "animal:read"]
}
```

### POST /v1/auth/register

Registers a user through auth-service. Gateway injects configured `tenant_id`.

Request:

```json
{
  "email": "alice@example.com",
  "password": "CorrectHorseBatteryStaple!",
  "locale": "ru-RU"
}
```

Response `201`:

```json
{
  "user": {
    "id": "user-id",
    "tenant_id": "default",
    "email": "alice@example.com",
    "is_active": true
  },
  "access_token": "<jwt>",
  "refresh_token": "<refresh-token>",
  "expires_at": "2026-04-18T06:44:52Z"
}
```

### POST /v1/auth/login

Authenticates a user through auth-service. Response `200` has the same token shape as register. Invalid credentials return `401`.

## Feed

### GET /v1/feed

Returns feed cards from feed-service.

Query: `surface` integer, `page_size` integer, `page_token` string.

Response `200`:

```json
{
  "cards": [],
  "next_page_token": "",
  "feed_session_id": "feed-session-id"
}
```

## Animals

### GET /v1/animals/{animal_id}

Returns one animal profile.

Response `200`:

```json
{
  "animal": {
    "animal_id": "animal-id",
    "owner_profile_id": "profile-id",
    "name": "Mila",
    "species": "SPECIES_CAT",
    "status": "ANIMAL_STATUS_AVAILABLE"
  }
}
```

### POST /v1/animals

Creates an animal profile. Body is JSON mapping of `petmatch.animal.v1.AnimalProfile`.

Request:

```json
{
  "owner_profile_id": "profile-id",
  "owner_type": "OWNER_TYPE_SHELTER",
  "name": "Mila",
  "species": "SPECIES_CAT",
  "breed": "Mixed",
  "sex": "ANIMAL_SEX_FEMALE",
  "size": "ANIMAL_SIZE_SMALL",
  "age_months": 8,
  "description": "Calm and friendly",
  "traits": ["calm", "friendly"],
  "vaccinated": true,
  "sterilized": true,
  "status": "ANIMAL_STATUS_DRAFT",
  "visibility": "VISIBILITY_PRIVATE"
}
```

Response `201` returns `{ "animal": ... }`.

### POST /v1/animals/{animal_id}/photos

Uploads a JPEG or PNG photo using multipart form data. Gateway uploads bytes to object storage, reads image dimensions, and sends photo metadata to animal-service.

Form fields: `photo` file is required; `sort_order` integer is optional.

Response `201`:

```json
{
  "animal": {
    "animal_id": "animal-id",
    "photos": [
      {
        "photo_id": "photo-id",
        "url": "http://storage/animals/animal-id/photo.jpg",
        "width": 1024,
        "height": 768,
        "content_type": "image/jpeg",
        "sort_order": 1
      }
    ]
  }
}
```

### POST /v1/animals/{animal_id}/swipe

Records a swipe through matching-service.

Request:

```json
{
  "owner_profile_id": "owner-profile-id",
  "direction": 1,
  "feed_card_id": "card-id",
  "feed_session_id": "feed-session-id"
}
```

`direction` accepts either an enum number or proto enum string, for example `SWIPE_DIRECTION_RIGHT`.

Response `200` returns swipe result, optional match, and optional `conversation_id`.

### POST /v1/animals/{animal_id}:swipe

Same behavior and body as `/v1/animals/{animal_id}/swipe`.

### GET /v1/animals/{animal_id}/stats

Returns animal analytics. Query: `bucket` integer time bucket enum.

## Profiles and Reviews

### GET /v1/profiles

Searches user profiles.

Query: `profile_type` repeatable integer, `city`, `min_average_rating`, `query`, `include_suspended`, `page_size`, `page_token`.

Response `200` returns `{ "profiles": [], "page": { "next_page_token": "" } }`.

### GET /v1/profiles/{profile_id}

Returns one profile.

### PATCH /v1/profiles/{profile_id}

Updates or creates a profile. Gateway accepts wrapper form:

```json
{
  "profile": {
    "display_name": "Alice",
    "bio": "Shelter volunteer",
    "address": {"city": "Moscow"},
    "visibility": 1
  },
  "update_mask": ["display_name", "bio", "address", "visibility"]
}
```

Gateway also accepts flat snake_case JSON:

```json
{
  "display_name": "Alice",
  "bio": "Shelter volunteer",
  "address": {"city": "Moscow"},
  "visibility": 1,
  "update_mask": ["display_name", "bio", "address", "visibility"]
}
```

Response `200` returns `{ "profile": ... }`.

### GET /v1/profiles/{profile_id}/reviews

Lists reviews for a profile. Query: `page_size`, `page_token`.

### POST /v1/profiles/{profile_id}/reviews

Creates a review for the target profile.

Request:

```json
{
  "rating": 5,
  "text": "Great adopter",
  "match_id": "match-id"
}
```

Response `201` returns `{ "review": ... }`.

### GET /v1/profiles/{profile_id}/reputation

Returns reputation summary for a profile.

### PATCH /v1/reviews/{review_id}

Updates a review.

Request:

```json
{
  "review": {
    "rating": 4,
    "text": "Updated text",
    "visibility": 1
  },
  "update_mask": ["rating", "text", "visibility"]
}
```

Response `200` returns `{ "review": ... }`.

## Chat

### GET /v1/chat/conversations

Lists conversations for the authenticated actor. Query: `page_size`, `page_token`.

### GET /v1/chat/conversations/{conversation_id}/messages

Lists messages in a conversation. Missing or invalid conversation IDs return `404`. Query: `page_size`, `page_token`.

### POST /v1/chat/conversations/{conversation_id}/messages

Sends a message.

Request:

```json
{
  "type": 1,
  "text": "Hello",
  "client_message_id": "client-message-id"
}
```

Response `201` returns `{ "message": ... }`.

## WebSocket

### GET /ws/chat

Upgrades to WebSocket and bridges JSON text frames to chat-service bidirectional gRPC stream.

Authentication is the same as protected REST endpoints:

```http
Authorization: Bearer <access_token>
```

The OpenAPI contract also documents `access_token` as a query parameter:

```text
ws://localhost:18088/ws/chat?access_token=<access_token>
```

Client frames are JSON mapping of `petmatch.chat.v1.ClientChatFrame`.

Example client frame:

```json
{
  "message": {
    "conversation_id": "conversation-id",
    "type": "MESSAGE_TYPE_TEXT",
    "text": "Hello"
  }
}
```

Server frames are JSON mapping of `petmatch.chat.v1.ServerChatFrame`.

Example server frame:

```json
{
  "message": {
    "message_id": "message-id",
    "conversation_id": "conversation-id",
    "text": "Hello"
  }
}
```

On connect and disconnect the gateway publishes operational Kafka events with connection id, actor id, request id, IP/user-agent, and timestamps.

## Billing

### POST /v1/billing/donation-intents

Creates a donation payment intent.

Request:

```json
{
  "target_type": 1,
  "target_id": "shelter-id",
  "amount": {
    "currency_code": "RUB",
    "units": 1000,
    "nanos": 0
  },
  "provider": "yookassa"
}
```

Response `201` returns donation details, `payment_url`, and `client_secret`.

## Notifications

### POST /v1/notifications/devices

Registers a push device token.

Request:

```json
{
  "token": "apns-or-fcm-token",
  "platform": "ios",
  "locale": "ru-RU"
}
```

Response `201` returns `{ "device_token_id": "device-token-id" }`.

### DELETE /v1/notifications/devices/{device_token_id}

Unregisters a push device token. Response `200` returns `{}`.

### GET /v1/notifications

Lists notifications for the authenticated actor. Query: `page_size`, `page_token`.

### GET /v1/notifications/stream

Opens an SSE stream from notification-service. Headers are flushed immediately after the downstream stream is opened.

Response headers:

```http
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

Event format:

```text
event: notification
data: {"notification_id":"notification-id","title":"New match","body":"You have a new match"}
```

### POST /v1/notifications/{notification_id}/read

Marks a notification as read. Response `200` returns `{ "notification": ... }`.

### POST /v1/notifications/{notification_id}:read

Same behavior as `/v1/notifications/{notification_id}/read`.

## Local Run

From `backend`:

```powershell
docker compose up --build -d api-gateway
```

From `backend/api-gateway` without Docker:

```powershell
go test ./...
go run ./cmd/api-gateway
```
