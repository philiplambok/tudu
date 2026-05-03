# Tudu — Clean Architecture Go Template (Design Spec)

**Date:** 2026-05-03
**Stack:** Go 1.26, PostgreSQL, chi, GORM, goose, cobra/viper, oapi-codegen, golang-jwt/jwt/v5, golang.org/x/crypto/bcrypt

## Purpose

A portfolio-grade Todoist application that demonstrates clean architecture in Go. Multi-user, JWT-authenticated. Intended to show other programmers how to structure a Go project: domain packages, service/repository interfaces, OpenAPI spec-first design, and the adapter/port pattern for external services.

---

## Project Structure

```
tudu/
├── api/openapi.yml                  # OpenAPI 3.0 spec (source of truth)
├── oapi_codegen.yml                 # oapi-codegen config
├── cmd/
│   ├── cmd.go                       # cobra root + loadConfig
│   ├── serve.go                     # wire app, start server
│   └── migrate.go                   # goose up/down
├── config.example.yml
├── config.yml                       # gitignored
├── db/migrations/                   # .sql goose migration files
├── dx/                              # developer scripts (Docker-based)
│   ├── _common                      # shared IMAGE_NAME, COMPOSE_FILE, helper fns
│   ├── Dockerfile                   # Go 1.26-bookworm dev image
│   ├── build, start, stop, dev      # container lifecycle
│   ├── shell, exec, logs, status    # container inspection
│   ├── test, lint                   # quality checks
│   ├── generate                     # openapi code generation
│   ├── db                           # migration subcommands
│   ├── dbgate                       # opt-in web DB GUI
│   ├── clean                        # destructive reset
│   └── README.md
├── docker-compose.dev.yml
├── .air.toml                        # live reload config
├── internal/
│   ├── config.go                    # Config struct (env, log, http_server, database, jwt)
│   ├── transport/
│   │   └── rest.go                  # chi server, JWT middleware, route mounting
│   ├── user/
│   │   ├── domain.go                # sentinel errors, ValidateRegister, ValidateLogin
│   │   ├── dto.go                   # RegisterRequestDTO, LoginRequestDTO, UserResponseDTO, ValidationError
│   │   ├── service.go               # Service interface + impl (register, login, me)
│   │   ├── repository.go            # Repository interface + impl
│   │   ├── handler.go               # HTTP handlers
│   │   └── endpoint.go              # route wiring; NewEndpoint(db, avatarProvider)
│   └── task/
│       ├── domain.go                # sentinel errors, ValidateCreate, ValidateUpdate, status constants (pending/completed)
│       ├── dto.go                   # CreateRequestDTO, UpdateRequestDTO, TaskResponseDTO, ValidationError
│       ├── service.go               # Service interface + impl
│       ├── repository.go            # Repository interface + impl
│       ├── handler.go               # HTTP handlers
│       └── endpoint.go              # route wiring
├── pkg/
│   ├── avatar/
│   │   ├── provider.go              # Provider interface: GetAvatarURL(email string) string
│   │   ├── gravatar.go              # real impl: MD5-hashed Gravatar URL
│   │   └── mock.go                  # mock impl: returns a static placeholder URL
│   └── openapi/v1/openapi.gen.go    # generated — do not edit by hand
└── main.go
```

---

## Architecture

Each domain (`user`, `task`) follows the same 6-file pattern from `fine`:

| File | Responsibility |
|---|---|
| `domain.go` | Sentinel errors, validation functions, domain aggregates |
| `dto.go` | All data transfer objects; `ValidationError` type |
| `service.go` | `Service` interface + struct impl; orchestrates domain logic + repo |
| `repository.go` | `Repository` interface + struct impl; all DB access via GORM |
| `handler.go` | HTTP handlers; parse request → call service → write response |
| `endpoint.go` | Wires repo → service → handler; returns `*chi.Mux` |

`internal/transport/rest.go` mounts all endpoints and holds JWT middleware.

`pkg/avatar/` demonstrates the adapter/port pattern: the `Provider` interface is consumed by `user.Service`; the concrete implementation is injected at startup.

---

## Database Schema

### `users`

Requires `CREATE EXTENSION IF NOT EXISTS citext;` — added in the first migration.

```sql
id            BIGSERIAL PRIMARY KEY
email         CITEXT UNIQUE NOT NULL
password_hash TEXT NOT NULL
avatar_url    TEXT NOT NULL
created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### `tasks`
```sql
id              BIGSERIAL PRIMARY KEY
user_id         BIGINT NOT NULL REFERENCES users(id)
title           TEXT NOT NULL
description     TEXT
status          TEXT NOT NULL DEFAULT 'pending'   -- 'pending' | 'completed'
due_date        DATE                              -- optional display field
completed_at    TIMESTAMPTZ
created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
```


---

## API Endpoints

### Auth (no JWT required)
```
POST /v1/auth/register    Register user; returns JWT + user profile
POST /v1/auth/login       Login; returns JWT + user profile
```

### Users (JWT required)
```
GET /v1/users/me          Current user profile (id, email, avatar_url)
```

### Tasks (JWT required — all scoped to the authenticated user)
```
POST   /v1/tasks                    Create task (title, description, due_date)
GET    /v1/tasks                    List tasks (filter: ?status=pending|completed)
GET    /v1/tasks/{id}               Get single task
PATCH  /v1/tasks/{id}               Update title, description, due_date
POST   /v1/tasks/{id}/complete      Mark task as completed
DELETE /v1/tasks/{id}               Delete task
```

JWT middleware extracts `user_id` from the token and injects it into `context.Context`. Task handlers read it from context — never from the request body.

---

## Authentication

- **Registration:** `bcrypt` hashes the password; `avatar.Provider.GetAvatarURL(email)` is called to populate `avatar_url`; a signed JWT (`HS256`) is returned.
- **Login:** compare `bcrypt` hash; return JWT on success.
- **JWT:** standard claims — `sub` = user ID, `exp` = 24h. Secret loaded from config (`jwt.secret`).
- **Middleware:** `JWTMiddleware` in `internal/transport/rest.go` validates the token and writes `userID` into context. Returns `401` on missing/invalid token.

---

## Business Logic

### Complete task (`task.Service.Complete`)

1. Load task; verify `user_id` matches the caller (return `404` if not found or not owned).
2. Set `status = completed`, `completed_at = NOW()`.
3. Return the completed `TaskResponseDTO`.

### Avatar adapter (`pkg/avatar`)

Selected at startup in `cmd/serve.go`:

```go
var avatarProvider avatar.Provider
if cfg.Env == "production" {
    avatarProvider = avatar.NewGravatar()
} else {
    avatarProvider = avatar.NewMock()
}
```

`avatar.NewGravatar()` returns the MD5-hashed Gravatar URL for the email.  
`avatar.NewMock()` returns a static `https://api.dicebear.com/...` URL regardless of email.

---

## Error Handling

Mirrors `fine`:

- `ValidationError` (pointer type) — `422 Unprocessable Entity`
- `ErrNotFound` sentinel — `404 Not Found` (also returned when a task exists but belongs to another user — avoids leaking task existence)
- Unexpected errors — `500 Internal Server Error` with a generic message (no internals leaked)

Response shape: `{"error": "<message>"}` — matches `fine`'s `ErrorResponse` schema.

---

## Config

```yaml
# config.example.yml
env: local                     # local | production — controls avatar provider
log:
  level: INFO
http_server:
  port: "8080"
database:
  source: ""
jwt:
  secret: ""
```

Environment variable override prefix: `ENV_` (e.g. `ENV_JWT_SECRET`).

---

## dx Toolkit

Full parity with `fine`. Adaptations:

| Item | Value |
|---|---|
| `IMAGE_NAME` | `tudu:dev` |
| Container names | `tudu-app`, `tudu-postgres` |
| DB name | `tudu` |
| App port | `8080` (host) → `8080` (container) |
| Postgres port | `5434` (host) → `5432` (container) |
| DBGate port | `3011` (host) → `3000` (container) |
| `air` build cmd | `go build -o ./tmp/tudu ./main.go` |
| `air` full_bin | `./tmp/tudu serve` |

Scripts: `build`, `start`, `stop`, `dev`, `shell`, `exec`, `test`, `lint`, `logs`, `status`, `clean`, `generate`, `db`, `dbgate`, `help`.

---

## OpenAPI / Code Generation

- Spec at `api/openapi.yml` is the single source of truth for request/response shapes.
- `oapi_codegen.yml` generates models-only into `pkg/openapi/v1/openapi.gen.go`.
- Handlers decode from `*http.Request` and map to internal DTOs manually (same pattern as `fine`).
- Regenerate with `./dx/generate openapi` after editing the spec.
