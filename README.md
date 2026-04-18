# Capibarco

Решение команды Seg_Fault по кейсу от компании PVS-Studio.

## Структура проекта

```text
capibarco/
  backend/
    auth-service/
    animal-service/
    billing-service/
    chat-service/
    feed-service/
    matching-service/
    proto/
    migrations/
    docker/
    docker-compose.yml
  docs/
```

## Backend

В `backend/` собраны микросервисы проекта. Каждый сервис лежит в отдельной папке и остается самостоятельным Go-модулем со своим `go.mod`, `Dockerfile`, `cmd/` и `internal/`.

Сервисы:

- `backend/auth-service` - сервис аутентификации.
- `backend/animal-service` - сервис анкет животных.
- `backend/billing-service` - сервис платежей, донатов и бустов.
- `backend/chat-service` - сервис чатов.
- `backend/feed-service` - сервис ленты рекомендаций.
- `backend/matching-service` - сервис свайпов, матчей и создания чатов по right swipe.
- `backend/api-gateway` - мобильный REST/WebSocket фасад поверх внутренних gRPC-сервисов.

Общие ресурсы:

- `backend/proto/` - объединенные proto-файлы.
- `backend/migrations/<service>/` - объединенные миграции по сервисам.
- `backend/docker/postgres/initdb.d/` - SQL для создания баз и пользователей PostgreSQL.
- `backend/docker-compose.yml` - общий запуск инфраструктуры и всех сервисов.

## Запуск backend

Перейдите в папку backend:

```powershell
cd backend
```

Запустите все сервисы:

```powershell
docker compose up --build
```

Порты на хосте:

| Компонент | HTTP | gRPC / API |
| --- | ---: | ---: |
| Auth service | `18080` | `15051` |
| Animal service | `18081` | `19090` |
| Billing service | `18082` | `19091` |
| Chat service | `18083` | `19092` |
| Feed service | `18084` | `18085` |
| Matching service | `18086` | `19094` |
| API Gateway | `18088` | - |
| PostgreSQL | - | `15432` |
| Redis | - | `16379` |
| MinIO | `19098` | `19099` |
| Redpanda Kafka API | - | `19093` |

Compose поднимает один PostgreSQL-контейнер с отдельными базами для сервисов, один Redpanda Kafka-compatible брокер, миграции и приложения.

## Проверка сервисов

Тесты запускаются отдельно в папке каждого микросервиса:

```powershell
cd backend/auth-service
go test ./...
```

То же самое можно выполнить для `animal-service`, `billing-service`, `chat-service`, `feed-service` и `matching-service`.

## Документация

Дополнительные заметки по переносу backend находятся в:

- `docs/superpowers/specs/2026-04-17-backend-microservices-consolidation-design.md`
- `docs/superpowers/plans/2026-04-17-backend-microservices-consolidation.md`
