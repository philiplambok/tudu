# Architecture

This document is a bird's-eye map of the codebase: where requests enter, how data moves through the system, and which boundaries matter. It is intentionally not an exhaustive file listing.

## High-level diagram

```
main.go
  └── cmd/
        ├── serve.go ──► internal/transport/rest.go (chi router + JWT middleware)
        │                      ├── internal/swagger/       (/swagger.json, /swagger/*)
        │                      ├── internal/user/endpoint  (/v1/auth/*, /v1/users/me)
        │                      │     handler → service → repository
        │                      │                                ↓
        │                      │                    internal/common/datamodel
        │                      └── internal/task/endpoint  (/v1/tasks/*)
        │                            handler → service → repository
        │                                          ↓              ↓
        │                                     task/domain   internal/common/datamodel
        └── migrate.go ──► db/migrations/ (goose SQL files)

Shared helpers: internal/common/util (pagination), internal/config.go, internal/context.go
```

---

## What this project is

`tudu` is a multi-user task management REST API written in Go. Users register and log in, receive JWTs, and manage their own tasks. Task changes are recorded as activity history rows.

The codebase follows a small clean-architecture style: HTTP concerns stay in handlers, application flow stays in services, persistence stays in repositories, and domain rules stay in domain files.

## Runtime entry points

`main.go` delegates to Cobra commands in `cmd/`.

- `cmd/serve.go` loads config, opens a PostgreSQL connection through GORM, builds the HTTP server, and starts listening.
- `cmd/migrate.go` runs goose migrations from `db/migrations/`.

The HTTP server is assembled in `internal/transport/rest.go`. It creates a chi router, registers Swagger, mounts public auth routes under `/v1/auth`, then mounts JWT-protected user and task routes under `/v1/users` and `/v1/tasks`.

JWT middleware validates `Authorization: Bearer <token>`, extracts the subject as `userID`, and stores it in request context via `internal.WithUserID`. Protected handlers read it back with `internal.UserIDFromContext`.

## Package shape

Domain features live under `internal/<feature>/` and use the same broad shape:

- `endpoint.go` wires repository → service → handler and exposes chi routes.
- `handler.go` maps HTTP/OpenAPI inputs and outputs.
- `service.go` owns application flow and DTO conversion between handler and repository boundaries.
- `repository.go` owns SQL/GORM persistence.
- `domain.go` owns domain constants, errors, validation, and domain behaviour.
- `dto.go` defines boundary DTOs.

Current feature packages are `internal/user` and `internal/task`.

Shared code lives in `internal/common/`:

- `internal/common/datamodel` contains structs that represent database tables and scan targets.
- `internal/common/util` contains cross-feature utilities such as pagination.

## Request/data flow

A typical protected task request flows like this:

1. `internal/transport/rest.go` authenticates the JWT and writes `userID` into context.
2. `internal/task/handler.go` parses path/query/body values and maps OpenAPI models to service request DTOs.
3. `internal/task/service.go` validates input, coordinates reads/writes, and maps request/record/response DTOs.
4. `internal/task/domain.go` applies task-specific rules such as update activity generation.
5. `internal/task/repository.go` runs SQL/GORM queries, scans into `internal/common/datamodel`, and maps rows to record DTOs.
6. The handler maps service response DTOs back into generated OpenAPI response models.

Keep this direction of dependency: handlers do not talk directly to repositories, repositories do not know HTTP types, and database datamodels do not cross into handlers.

## DTO boundaries

The project deliberately distinguishes DTOs by boundary:

- `*RequestDTO`: handler → service input.
- `*ResponseDTO`: service → handler output.
- `*RecordDTO`: service ↔ repository records.
- `internal/common/datamodel`: database-table representation used inside repositories.

This duplication is intentional. It keeps HTTP response shape, application records, and database scan structs from becoming one shared type that is hard to change.

## Tasks and activity history

The task package contains the most important domain behaviour.

`internal/task/domain.go` defines the `Task` aggregate. A `Task` carries task fields plus pending `Activities`. Creating a task uses `NewTask`, which creates a pending `created` activity. Updating a task uses `TaskFromRecord(...).ApplyUpdate(req)`, which compares changed fields and appends `updated` activities for `title`, `description`, and `due_date`.

`internal/task/repository.go` persists task rows and activity rows atomically. For create and update, the repository receives the aggregate and writes both `tasks` and `task_activities` in one transaction. The repository should not reimplement activity diff logic; that belongs to the domain aggregate.

Activity history is exposed via `GET /v1/tasks/{id}/activities`. The repository first verifies the task belongs to the authenticated user, then lists matching activity rows ordered newest first.

## Pagination

Task listing uses offset pagination through `internal/common/util/pagination.go`.

- Query params: `page`, `limit`.
- Defaults: `page=1`, `limit=20`.
- Maximum limit: `100`.
- Invalid or non-positive values normalize silently to defaults.

Repositories return rows plus a total count. Services wrap results in `util.PagingResponse[T]`. Handlers return nested API responses with `data.tasks` and `data.page_info` using snake_case metadata fields.

## OpenAPI and generated models

`api/openapi.yml` is the API contract. Generated Go models live in `pkg/openapi/v1/openapi.gen.go` and are regenerated with:

```bash
./dx/generate openapi
```

Handlers should map between generated OpenAPI models and internal DTOs instead of passing generated models through services or repositories.

Swagger UI is registered by `internal/swagger` and served by the HTTP server.

## Persistence

PostgreSQL schema changes live in `db/migrations/` and are applied with goose via `./dx/db migrate` or the `migrate` command. The app uses GORM for database access, but repositories mostly keep SQL shape explicit and map scanned datamodel structs into record DTOs.

The dev PostgreSQL instance is provided by `docker-compose.dev.yml` and exposed on host port `5434`.

## Tests

Unit tests live beside the code they exercise. Service tests usually use small repository mocks. Domain tests exercise pure behaviour such as task activity generation. Utility tests cover shared helpers like pagination.

The usual commands are:

```bash
./dx/test ./...
./dx/test ./internal/task -run TestName -v
```
