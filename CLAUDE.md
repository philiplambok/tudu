# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Environment

All dev commands run through Docker Compose via the `dx/` wrapper scripts. The app itself (`./tudu`) has two subcommands: `serve` and `migrate`.

```bash
./dx/build              # build the Docker image
./dx/start              # start Postgres (and optional services)
./dx/dev                # start the app with hot-reload (air)
./dx/stop --remove      # stop and remove containers
./dx/shell              # open a shell inside the app container
./dx/workspace show     # show active workspace name and ports
```

### Multiple Worktrees

Each git worktree gets its own isolated Docker stack via `dx/workspace.env` (gitignored). Without this file, the workspace name is derived from the current git branch and all ports use offset 0.

```bash
./dx/workspace init <name> <port-offset>   # create workspace.env for this worktree
./dx/workspace show                         # confirm COMPOSE_PROJECT_NAME and ports
```

Port layout (offset 0 = default, each active worktree must use a unique offset):
- APP_PORT = 8080 + PORT_OFFSET
- POSTGRES_PORT = 5434 + PORT_OFFSET
- DBGATE_PORT = 3011 + PORT_OFFSET

`COMPOSE_PROJECT_NAME` is derived as `tudu-<WORKSPACE_NAME>` and scopes all Docker containers, volumes, and networks to the worktree.

## Common Commands

```bash
./dx/test ./...                                          # run all tests
./dx/test ./internal/task -run TestList_WithPagination  # run a single test
./dx/lint                                               # run golangci-lint
./dx/generate openapi                                   # regenerate pkg/openapi/v1/openapi.gen.go from api/openapi.yml
./dx/db migrate                                         # apply pending goose migrations
./dx/db migrate:rollback                                # roll back last migration
./dx/db status                                          # show migration status
```

Outside of Docker (unit tests only):

```bash
go test ./...
go test ./internal/task -run TestName -v
go build ./...
```

## OpenAPI Workflow

`api/openapi.yml` is the source of truth. Generated types live in `pkg/openapi/v1/openapi.gen.go`. After editing the spec, regenerate:

```bash
./dx/generate openapi
```

Config lives in `oapi_codegen.yml` (models only, no server stubs).

## Architecture

See `ARCHITECTURE.md` for a detailed walkthrough of code structure and data flow.

**DTO boundary naming conventions:**
- `*RequestDTO` — handler → service input
- `*ResponseDTO` — service → handler output
- `*RecordDTO` — service ↔ repository records
- `internal/common/datamodel` — DB-representative structs scanned by GORM; repositories map these to `*RecordDTO` before returning

**Domain aggregates:** `internal/task/domain.go` owns domain constants and aggregate behaviour. The `Task` aggregate carries pending `Activities`; `repository.go` persists both atomically in a single transaction. Business logic (e.g. activity diff generation) belongs in the domain layer, not the repository.

**Pagination:** shared utility lives in `internal/common/util/pagination.go`. Default page=1, limit=20, max limit=100. Invalid params normalise silently. Repository returns `([]RecordDTO, int64, error)`; service wraps in `util.PagingResponse[T]`.
