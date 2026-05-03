# tudu

Dockerized Go REST API for multi-user task management, built to experiment with clean architecture in Go. Users register, authenticate via JWT, and manage personal tasks with full activity history tracking.

## Tech Stack

- **Language**: Go 1.26
- **Router**: chi v5
- **ORM**: GORM + PostgreSQL 16
- **Auth**: JWT (HS256) + bcrypt
- **Migrations**: goose (SQL files)
- **API Spec**: OpenAPI 3.0 (`api/openapi.yml`) + Swagger UI
- **Dev tools**: Docker Compose, air (live reload), oapi-codegen

## Getting Started

Everything runs inside Docker — no local Go installation required.

```bash
# 1. Build the dev image (first time)
./dx/build

# 2. Copy and configure
cp config.example.yml config.yml
# Edit config.yml: set database.source and jwt.secret

# 3. Start app + PostgreSQL
./dx/start

# 4. Run migrations
./dx/db migrate

# 5. Start with live reload
./dx/dev
```

| Service | URL |
|---|---|
| HTTP API | http://localhost:8080 |
| Swagger UI | http://localhost:8080/swagger/ |
| PostgreSQL | localhost:5434 |
| DBGate (optional) | http://localhost:3011 |

## Configuration

Config is loaded from `config.yml`. All keys can be overridden via environment variables prefixed with `ENV_` (e.g. `ENV_DATABASE_SOURCE`).

```yaml
env: local                 # local | production
log:
  level: INFO
http_server:
  port: "8080"
database:
  source: ""               # PostgreSQL DSN
jwt:
  secret: ""               # HMAC signing secret
```

## API

All protected endpoints require `Authorization: Bearer <token>`.

**Auth**

| Method | Path | Description |
|---|---|---|
| POST | `/v1/auth/register` | Register; returns JWT + user |
| POST | `/v1/auth/login` | Login; returns JWT + user |

**Users**

| Method | Path | Description |
|---|---|---|
| GET | `/v1/users/me` | Authenticated user profile |

**Tasks**

| Method | Path | Description |
|---|---|---|
| POST | `/v1/tasks` | Create task |
| GET | `/v1/tasks` | List tasks (`?status=pending\|completed`) |
| GET | `/v1/tasks/{id}` | Get task |
| PATCH | `/v1/tasks/{id}` | Update task |
| POST | `/v1/tasks/{id}/complete` | Mark as completed |
| DELETE | `/v1/tasks/{id}` | Delete task |
| GET | `/v1/tasks/{id}/activities` | Activity history (audit log) |

Full API spec available at `/swagger.json` and `/swagger/` when the server is running.

## Development

Development and runtime are containerized, so Docker is the only local application required to develop and run this project.

```bash
./dx/test ./...            # run tests
./dx/lint                  # go vet
./dx/generate openapi      # regenerate types from api/openapi.yml
./dx/shell                 # bash shell in container
./dx/db migrate:rollback   # roll back last migration
./dx/db status             # migration status
./dx/dbgate start          # optional DB web GUI
./dx/stop --remove         # stop + drop volumes
```

## Project Structure

```
├── api/openapi.yml          # OpenAPI 3.0 spec (source of truth)
├── cmd/                     # CLI commands: serve, migrate
├── db/migrations/           # SQL migration files (goose)
├── internal/
│   ├── transport/           # chi router, REST wiring, and JWT middleware
│   ├── user/                # Register, login, profile domain
│   ├── task/                # Task CRUD + activity log domain
│   │   ├── endpoint.go      # route registration and dependency wiring
│   │   ├── handler.go       # HTTP/OpenAPI request and response mapping
│   │   ├── service.go       # application flow and DTO conversion
│   │   ├── domain.go        # task aggregate, rules, and activity generation
│   │   ├── repository.go    # PostgreSQL/GORM persistence
│   │   └── dto.go           # request, response, and record DTOs
│   ├── common/
│   │   ├── datamodel/       # database-representative structs for GORM scans
│   │   └── util/            # shared utilities such as pagination
│   ├── swagger/             # Swagger UI + spec endpoint
│   └── config.go
├── pkg/
│   └── openapi/v1/          # Generated types from OpenAPI spec
└── dx/                      # Docker-based developer scripts
```

## CLI

```bash
./tudu serve    # Start HTTP server
./tudu migrate  # Run database migrations
```

## Action Items

- Add a mock notification service before implementing the real notification service integration.
- Add worker scheduler to send reminders to the notification service.
- Add testcontainers to test infrastructure-layer integrations such as database access and API calls.
- Add database seeding for local development and repeatable manual testing.
- Add `dx/playbooks` with manual smoke-test guides: runnable curls, prerequisites, seeded data assumptions, and expected API/database results.