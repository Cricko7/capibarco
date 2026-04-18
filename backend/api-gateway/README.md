# api-gateway

Mobile BFF и edge gateway для PetMatch. Сервис принимает внешний REST/JSON и WebSocket трафик от мобильного клиента, выполняет security/rate limit/observability и проксирует запросы во внутренние gRPC-сервисы.

## Responsibilities

- REST/JSON facade для мобильных клиентов.
- Регистрация, логин, JWT validation и RBAC delegation через auth-service.
- Guest sessions для анонимного просмотра ленты.
- Redis-backed rate limiting по IP, actor и role.
- gRPC orchestration для auth, animal, feed, matching, chat, billing, analytics и будущего notification-service.
- Multipart upload фото животных в MinIO/S3 с записью metadata в animal-service.
- WebSocket chat bridge в chat-service bidirectional streaming.
- Optional SSE bridge в notification-service server streaming.
- Kafka operational events для rejected requests и WebSocket lifecycle.
- Prometheus metrics, health checks, structured slog logging, request id propagation и graceful shutdown.

## Local Run

Из корня `backend`:

```powershell
docker compose up --build -d api-gateway
```

Gateway будет доступен на `http://localhost:18088`.

Локальный запуск без Docker из `backend/api-gateway`:

```powershell
go mod tidy
go test ./...
go run ./cmd/api-gateway
```

OpenAPI is served at `/openapi.yaml`.

## Общие правила

Base URL для Docker Compose:

```text
http://localhost:18088
```

JSON requests используют:

```http
Content-Type: application/json
```

Protected endpoints принимают JWT:

```http
Authorization: Bearer <access_token>
```

или guest session:

```http
X-Guest-Session-Token: <guest_session_token>
```

Для mutating requests передавайте idempotency key:

```http
Idempotency-Key: <unique-operation-key>
```

Gateway возвращает `X-Request-ID`. Можно передать свой:

```http
X-Request-ID: req-123
```

Ошибки возвращаются в RFC 7807-like формате:

```json
{
  "type": "https://api.petmatch.local/errors/bad-request",
  "title": "Bad Request",
  "status": 400,
  "detail": "invalid input: email is required",
  "instance": "/v1/auth/register"
}
```

Типовые статусы:

| Status | Meaning |
| --- | --- |
| `200` | Запрос выполнен |
| `201` | Resource/command создан |
| `204` | Preflight OPTIONS принят |
| `400` | Невалидный request |
| `401` | Нет или невалиден token |
| `403` | Недостаточно permissions |
| `404` | Resource не найден |
| `429` | Rate limit exceeded |
| `500` | Internal error |
| `502` | Downstream unavailable/deadline |

## Список методов

| Method | Path | Auth | Назначение |
| --- | --- | --- | --- |
| `GET` | `/healthz` | no | Liveness |
| `GET` | `/readyz` | no | Readiness, Redis ping |
| `GET` | `/metrics` | no | Prometheus metrics |
| `GET` | `/openapi.yaml` | no | OpenAPI spec |
| `OPTIONS` | `/*` | no | CORS preflight |
| `POST` | `/v1/auth/guest-sessions` | no | Guest session |
| `POST` | `/v1/auth/register` | no | Registration через auth-service |
| `POST` | `/v1/auth/login` | no | Login через auth-service |
| `GET` | `/v1/feed` | JWT/guest | Feed cards |
| `GET` | `/v1/animals/{animal_id}` | JWT/guest | Animal profile |
| `POST` | `/v1/animals` | JWT/guest | Create animal |
| `POST` | `/v1/animals/{animal_id}/photos` | JWT/guest | Upload animal photo |
| `POST` | `/v1/animals/{animal_id}/swipe` | JWT/guest | Swipe animal |
| `POST` | `/v1/animals/{animal_id}:swipe` | JWT/guest | Contract alias для swipe |
| `GET` | `/v1/animals/{animal_id}/stats` | JWT/guest | Animal analytics |
| `GET` | `/v1/chat/conversations` | JWT/guest | List conversations |
| `GET` | `/v1/chat/conversations/{conversation_id}/messages` | JWT/guest | List messages |
| `POST` | `/v1/chat/conversations/{conversation_id}/messages` | JWT/guest | Send message |
| `GET` | `/ws/chat` | JWT/guest | WebSocket chat bridge |
| `POST` | `/v1/billing/donation-intents` | JWT/guest | Create donation intent |
| `GET` | `/v1/notifications` | JWT/guest | List notifications |
| `GET` | `/v1/notifications/stream` | JWT/guest | SSE notification stream |
| `POST` | `/v1/notifications/devices` | JWT/guest | Register push token |
| `DELETE` | `/v1/notifications/devices/{device_token_id}` | JWT/guest | Unregister push token |
| `POST` | `/v1/notifications/{notification_id}/read` | JWT/guest | Mark notification read |
| `POST` | `/v1/notifications/{notification_id}:read` | JWT/guest | Contract alias для read |

## Auth flow

### POST /v1/auth/register

Регистрирует пользователя. Gateway сам подставляет `tenant_id` из config, локально это `default`.

```bash
curl -i -X POST http://localhost:18088/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "CorrectHorseBatteryStaple!",
    "locale": "ru-RU"
  }'
```

Response `201`:

```json
{
  "user": {
    "id": "4979adec-763c-43e9-bbdb-7186638898de",
    "tenant_id": "default",
    "email": "alice@example.com",
    "is_active": true,
    "created_at": "2026-04-18T06:29:52Z",
    "updated_at": "2026-04-18T06:29:52Z"
  },
  "access_token": "eyJhbGciOiJFZERTQSIs...",
  "refresh_token": "P_MQonrQwqyFt_ZwnyVTkSJQLvvkG4-ArC53wNn9nc8",
  "expires_at": "2026-04-18T06:44:52Z"
}
```

### POST /v1/auth/login

```bash
curl -i -X POST http://localhost:18088/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "CorrectHorseBatteryStaple!",
    "locale": "ru-RU"
  }'
```

Response `200`:

```json
{
  "user": {
    "id": "4979adec-763c-43e9-bbdb-7186638898de",
    "tenant_id": "default",
    "email": "alice@example.com",
    "is_active": true
  },
  "access_token": "eyJhbGciOiJFZERTQSIs...",
  "refresh_token": "aWQoJlUvoBsj06ZAauSpGPokdunxZ_mz9pG12DREvSY",
  "expires_at": "2026-04-18T06:44:52Z"
}
```

Wrong password возвращает `401`.

### POST /v1/auth/guest-sessions

Создает signed guest session для анонимного просмотра feed.

```bash
curl -i -X POST http://localhost:18088/v1/auth/guest-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "ios-device-123",
    "locale": "ru-RU"
  }'
```

Response `201`:

```json
{
  "guest_session_token": "eyJzaWQiOiJnc3QtLi4u",
  "expires_at": "2026-04-19T06:00:00Z",
  "allowed_scopes": ["feed:read", "animal:read"]
}
```

## Operational endpoints

### GET /healthz

```bash
curl -i http://localhost:18088/healthz
```

Response `200`:

```json
{"status":"ok"}
```

### GET /readyz

```bash
curl -i http://localhost:18088/readyz
```

Response `200`:

```json
{"status":"ready"}
```

Если Redis недоступен, вернется `503`.

### GET /metrics

```bash
curl -i http://localhost:18088/metrics
```

Response `200`: Prometheus text format.

### GET /openapi.yaml

```bash
curl -i http://localhost:18088/openapi.yaml
```

Response `200`: OpenAPI YAML.

### OPTIONS /*

CORS preflight.

```bash
curl -i -X OPTIONS http://localhost:18088/v1/feed \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET"
```

Response `204`.

## Feed

### GET /v1/feed

Возвращает окно карточек feed.

Query parameters:

| Name | Type | Description |
| --- | --- | --- |
| `surface` | integer | Feed surface enum |
| `page_size` | integer | Размер страницы, gateway применяет cap |
| `page_token` | string | Cursor следующей страницы |

JWT example:

```bash
curl -i "http://localhost:18088/v1/feed?page_size=10&surface=1" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Guest example:

```bash
curl -i "http://localhost:18088/v1/feed?page_size=10" \
  -H "X-Guest-Session-Token: $GUEST_SESSION_TOKEN"
```

Response `200`:

```json
{
  "cards": [
    {
      "card_id": "card-123",
      "animal": {
        "animal_id": "animal-123",
        "name": "Mila",
        "species": "SPECIES_CAT"
      }
    }
  ],
  "next_page_token": "cursor-2",
  "feed_session_id": "feed-session-abc"
}
```

## Animals

### GET /v1/animals/{animal_id}

```bash
curl -i http://localhost:18088/v1/animals/animal-123 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Response `200`:

```json
{
  "animal": {
    "animal_id": "animal-123",
    "owner_profile_id": "profile-123",
    "owner_type": "OWNER_TYPE_SHELTER",
    "name": "Mila",
    "species": "SPECIES_CAT",
    "breed": "Mixed",
    "sex": "ANIMAL_SEX_FEMALE",
    "size": "ANIMAL_SIZE_SMALL",
    "age_months": 8,
    "description": "Calm and friendly",
    "status": "ANIMAL_STATUS_AVAILABLE"
  }
}
```

### POST /v1/animals

Создает animal profile. Body использует JSON-представление `petmatch.animal.v1.AnimalProfile`.

```bash
curl -i -X POST http://localhost:18088/v1/animals \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Idempotency-Key: create-animal-001" \
  -H "Content-Type: application/json" \
  -d '{
    "owner_profile_id": "profile-123",
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
  }'
```

Response `201`:

```json
{
  "animal": {
    "animal_id": "animal-123",
    "owner_profile_id": "profile-123",
    "name": "Mila",
    "species": "SPECIES_CAT",
    "status": "ANIMAL_STATUS_DRAFT"
  }
}
```

### POST /v1/animals/{animal_id}/photos

Multipart upload фото животного.

```bash
curl -i -X POST http://localhost:18088/v1/animals/animal-123/photos \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Idempotency-Key: upload-photo-001" \
  -F "photo=@./cat.jpg;type=image/jpeg" \
  -F "sort_order=1"
```

Response `201`:

```json
{
  "animal": {
    "animal_id": "animal-123",
    "photos": [
      {
        "photo_id": "photo-123",
        "url": "http://localhost:19098/petmatch-photos/animals/animal-123/photo.jpg",
        "content_type": "image/jpeg",
        "sort_order": 1
      }
    ]
  }
}
```

### POST /v1/animals/{animal_id}/swipe

```bash
curl -i -X POST http://localhost:18088/v1/animals/animal-123/swipe \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Idempotency-Key: swipe-001" \
  -H "Content-Type: application/json" \
  -d '{
    "owner_profile_id": "owner-profile-123",
    "direction": 1,
    "feed_card_id": "card-123",
    "feed_session_id": "feed-session-abc"
  }'
```

Response `200`:

```json
{
  "swipe": {
    "swipe_id": "swipe-123",
    "actor_id": "profile-456",
    "animal_id": "animal-123",
    "direction": "SWIPE_DIRECTION_LIKE"
  },
  "match": {
    "match_id": "match-123",
    "animal_id": "animal-123"
  },
  "conversation_id": "conversation-123"
}
```

### POST /v1/animals/{animal_id}:swipe

Alias для contract path.

```bash
curl -i -X POST http://localhost:18088/v1/animals/animal-123:swipe \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Idempotency-Key: swipe-002" \
  -H "Content-Type: application/json" \
  -d '{
    "owner_profile_id": "owner-profile-123",
    "direction": 2
  }'
```

### GET /v1/animals/{animal_id}/stats

Возвращает animal analytics через analytics-service.

```bash
curl -i "http://localhost:18088/v1/animals/animal-123/stats?bucket=1" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Response `200`:

```json
{
  "stats": {
    "animal_id": "animal-123",
    "views": 120,
    "likes": 18,
    "swipes": 44,
    "matches": 6
  }
}
```

## Chat

### GET /v1/chat/conversations

```bash
curl -i "http://localhost:18088/v1/chat/conversations?page_size=20" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Response `200`:

```json
{
  "conversations": [
    {
      "conversation_id": "conversation-123",
      "match_id": "match-123",
      "animal_id": "animal-123",
      "adopter_profile_id": "profile-456",
      "owner_profile_id": "profile-123",
      "status": "CONVERSATION_STATUS_ACTIVE",
      "unread_count": 2
    }
  ],
  "page": {
    "next_page_token": "cursor-2"
  }
}
```

### GET /v1/chat/conversations/{conversation_id}/messages

```bash
curl -i "http://localhost:18088/v1/chat/conversations/conversation-123/messages?page_size=30" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Response `200`:

```json
{
  "messages": [
    {
      "message_id": "message-123",
      "conversation_id": "conversation-123",
      "sender_profile_id": "profile-456",
      "type": "MESSAGE_TYPE_TEXT",
      "text": "Здравствуйте! Можно узнать подробнее?",
      "sent_at": "2026-04-18T06:30:00Z"
    }
  ],
  "page": {
    "next_page_token": "cursor-2"
  }
}
```

### POST /v1/chat/conversations/{conversation_id}/messages

```bash
curl -i -X POST http://localhost:18088/v1/chat/conversations/conversation-123/messages \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Idempotency-Key: send-message-001" \
  -H "Content-Type: application/json" \
  -d '{
    "type": 1,
    "text": "Здравствуйте! Можно узнать подробнее?",
    "client_message_id": "client-msg-001"
  }'
```

Response `201`:

```json
{
  "message": {
    "message_id": "message-123",
    "conversation_id": "conversation-123",
    "sender_profile_id": "profile-456",
    "type": "MESSAGE_TYPE_TEXT",
    "text": "Здравствуйте! Можно узнать подробнее?",
    "sent_at": "2026-04-18T06:30:00Z"
  }
}
```

## WebSocket chat

### GET /ws/chat

WebSocket bridge в chat-service.

```bash
wscat -c "ws://localhost:18088/ws/chat" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Или через query token:

```bash
wscat -c "ws://localhost:18088/ws/chat?access_token=$ACCESS_TOKEN"
```

Client frame example:

```json
{
  "type": "CHAT_FRAME_TYPE_MESSAGE",
  "message": {
    "conversation_id": "conversation-123",
    "type": "MESSAGE_TYPE_TEXT",
    "text": "Привет!"
  }
}
```

Server frame example:

```json
{
  "type": "CHAT_FRAME_TYPE_ACK",
  "message": {
    "message_id": "message-123",
    "conversation_id": "conversation-123"
  }
}
```

Обычный HTTP GET без WebSocket upgrade вернет `400`.

## Billing

### POST /v1/billing/donation-intents

Создает donation payment intent.

```bash
curl -i -X POST http://localhost:18088/v1/billing/donation-intents \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Idempotency-Key: donation-001" \
  -H "Content-Type: application/json" \
  -d '{
    "target_type": 1,
    "target_id": "shelter-123",
    "amount": {
      "currency_code": "RUB",
      "units": 1000,
      "nanos": 0
    },
    "provider": "yookassa"
  }'
```

Response `201`:

```json
{
  "donation": {
    "donation_id": "donation-123",
    "payer_profile_id": "profile-456",
    "target_type": "DONATION_TARGET_TYPE_SHELTER",
    "target_id": "shelter-123",
    "amount": {
      "currency_code": "RUB",
      "units": 1000
    },
    "status": "PAYMENT_STATUS_PENDING",
    "provider": "yookassa"
  },
  "payment_url": "https://pay.example/checkout/donation-123",
  "client_secret": "secret-123"
}
```

## Notifications

Notification-service пока optional/future service. В локальном compose он может быть выключен через `API_GATEWAY_GRPC_NOTIFICATION_ENABLED=false`.

### GET /v1/notifications

```bash
curl -i "http://localhost:18088/v1/notifications?page_size=30" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Response `200`:

```json
{
  "notifications": [
    {
      "notification_id": "notification-123",
      "recipient_profile_id": "profile-456",
      "title": "Новый матч",
      "body": "У вас новый матч по анкете Mila"
    }
  ],
  "page": {
    "next_page_token": "cursor-2"
  }
}
```

### GET /v1/notifications/stream

SSE stream уведомлений.

```bash
curl -N -i http://localhost:18088/v1/notifications/stream \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Event example:

```text
event: notification
data: {"notification_id":"notification-123","title":"Новый матч","body":"У вас новый матч"}
```

### POST /v1/notifications/devices

Регистрирует push token.

```bash
curl -i -X POST http://localhost:18088/v1/notifications/devices \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "token": "apns-or-fcm-token",
    "platform": "ios",
    "locale": "ru-RU"
  }'
```

Response `201`:

```json
{
  "device_token_id": "device-token-123"
}
```

### DELETE /v1/notifications/devices/{device_token_id}

```bash
curl -i -X DELETE http://localhost:18088/v1/notifications/devices/device-token-123 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Response `200`:

```json
{}
```

### POST /v1/notifications/{notification_id}/read

```bash
curl -i -X POST http://localhost:18088/v1/notifications/notification-123/read \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Response `200`:

```json
{
  "notification": {
    "notification_id": "notification-123",
    "read_at": "2026-04-18T06:30:00Z"
  }
}
```

### POST /v1/notifications/{notification_id}:read

Alias для contract path.

```bash
curl -i -X POST http://localhost:18088/v1/notifications/notification-123:read \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## Полный smoke пример

```powershell
$base = "http://localhost:18088"
$email = "gateway-smoke-$([guid]::NewGuid().ToString('N'))@example.com"
$password = "CorrectHorseBatteryStaple!"

$body = @{
  email = $email
  password = $password
  locale = "ru-RU"
} | ConvertTo-Json -Compress

$register = Invoke-RestMethod -Method Post -Uri "$base/v1/auth/register" -ContentType "application/json" -Body $body
$token = $register.access_token

Invoke-RestMethod -Uri "$base/healthz"
Invoke-RestMethod -Uri "$base/readyz"
Invoke-RestMethod -Uri "$base/v1/feed?page_size=3" -Headers @{ Authorization = "Bearer $token" }
Invoke-RestMethod -Uri "$base/v1/chat/conversations?page_size=2" -Headers @{ Authorization = "Bearer $token" }
```

## Known local limitations

- `notification-service` может быть выключен в локальном compose, поэтому notification endpoints зависят от `API_GATEWAY_GRPC_NOTIFICATION_ENABLED`.
- `analytics-service` и `billing-service` должны быть доступны для соответствующих downstream endpoints.
- Redpanda init может логировать `topic_already_exists` при повторном запуске. Это не ошибка gateway, если topics уже созданы.
