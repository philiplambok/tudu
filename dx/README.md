# dx — Developer Experience Toolkit

Docker-based scripts for local development. All commands run inside containers — no local Go install needed.

## Quick Start

```bash
./dx/build            # Build the dev Docker image (first time only)
./dx/start            # Start app + postgres
./dx/db migrate       # Apply database migrations
./dx/dev              # Start HTTP server with live reload → http://localhost:8080
```

## Commands

### Container Lifecycle
| Command | Description |
|---|---|
| `./dx/build [--no-cache]` | Build the `tudu:dev` Docker image |
| `./dx/start` | Start app + postgres in the background |
| `./dx/stop [--remove]` | Stop containers; `--remove` also drops volumes |
| `./dx/status` | Show running container status |
| `./dx/logs [service]` | Tail container logs |
| `./dx/clean` | Remove containers, volumes, and image (**destructive**) |

### Development
| Command | Description |
|---|---|
| `./dx/dev [--no-reload]` | Run HTTP server (default: live reload via air) |
| `./dx/shell` | Open bash shell in the app container |
| `./dx/exec <cmd>` | Run any command in the app container |
| `./dx/test [args...]` | Run `go test` (default: `./...`) |
| `./dx/lint` | Run `go vet ./...` |
| `./dx/workspace show` | Show current workspace config and port assignments |
| `./dx/workspace init [name] [offset]` | Create `dx/workspace.env` for this worktree |

### Database
| Command | Description |
|---|---|
| `./dx/db migration <name>` | Create a new timestamped migration file |
| `./dx/db migrate` | Apply all pending migrations |
| `./dx/db migrate:rollback` | Roll back the most recent migration |
| `./dx/db status` | Show goose migration status |

### Code Generation
| Command | Description |
|---|---|
| `./dx/generate openapi` | Regenerate `pkg/openapi/v1/openapi.gen.go` from `api/openapi.yml` |

### Multi-Worktree (parallel workspaces)

Each worktree can run its own isolated Docker stack — separate containers, volumes, and ports — via a `dx/workspace.env` config file (gitignored).

| Command | Description |
|---|---|
| `./dx/workspace init [name] [port-offset]` | Create `dx/workspace.env` for this worktree |
| `./dx/workspace show` | Print active workspace name and port assignments |

**Setup for a second worktree (each active worktree needs a unique port offset):**

```bash
# In the second worktree directory:
./dx/workspace init feature-branch 1
./dx/workspace show    # confirms APP_PORT=8081, POSTGRES_PORT=5435

./dx/build
./dx/start
./dx/db migrate
./dx/dev               # → http://localhost:8081
```

Port offsets per worktree:

| Offset | APP_PORT | POSTGRES_PORT | DBGATE_PORT |
|---|---|---|---|
| 0 (default) | 8080 | 5434 | 3011 |
| 1 | 8081 | 5435 | 3012 |
| 2 | 8082 | 5436 | 3013 |

`dx/workspace.env` is gitignored — each developer sets it locally. See `dx/workspace.env.example` for all available settings.

### DBGate (optional web GUI)
| Command | Description |
|---|---|
| `./dx/dbgate start` | Start DBGate at http://localhost:3011 |
| `./dx/dbgate open` | Open in browser |
| `./dx/dbgate stop` | Stop the container |

## Ports

| Service | Host | Container |
|---|---|---|
| HTTP server | http://localhost:8080 | 8080 |
| PostgreSQL | localhost:5434 | 5432 |
| DBGate | http://localhost:3011 | 3000 |
| Swagger UI | http://localhost:8080/swagger/ | — |

Ports shift by `PORT_OFFSET` when running multiple worktrees simultaneously. Run `./dx/workspace show` to confirm active port assignments.
