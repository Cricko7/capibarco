# Capibarco Mobile

Production-ready Flutter client for the PetMatch backend microservices.

## Architecture

- Clean Architecture with feature-first modules.
- Riverpod for state management and dependency graph.
- `go_router` with guarded auth-aware navigation.
- Dio networking with auth, retry, logging, timeout, and centralized error mapping.
- Local cache and secure session persistence.
- Material 3, adaptive light/dark theme, localization-ready app shell.

## Project Structure

```text
lib/
  app/                application shell, router, theme, localization
  bootstrap/          startup composition and provider overrides
  core/               config, networking, storage, cache, analytics, push hooks
  features/
    auth/
    discovery/
    feed/
    notifications/
    profile/
  shared/             reusable presentation and utility building blocks
```

## Backend Integration

The mobile client is built around `backend/api-gateway/README.md` and talks to the gateway as the public BFF. Internally, the app still separates transport from business logic:

- `API clients` know HTTP contracts.
- `data sources` orchestrate remote/local persistence.
- `repositories` expose domain-friendly interfaces.
- `domain entities` stay decoupled from DTOs.

Current service adapters:

- Auth
- Feed / matching
- Profiles
- Notifications

The environment layer is intentionally prepared for future REST, gRPC, or GraphQL transports without rewriting the UI layer.

## Environment

The app supports `--dart-define` configuration:

```bash
flutter run ^
  --dart-define=APP_ENV=staging ^
  --dart-define=API_GATEWAY_URL=http://10.0.2.2:18088 ^
  --dart-define=API_VERSION=v1
```

Available keys:

- `APP_ENV`: `local`, `staging`, `production`
- `API_GATEWAY_URL`: public gateway base URL
- `API_VERSION`: current REST version prefix
- `ENABLE_HTTP_LOGS`: `true` or `false`

## Run

```bash
flutter pub get
flutter run --dart-define=API_GATEWAY_URL=http://10.0.2.2:18088
```

## Test

```bash
flutter test
flutter analyze
```
