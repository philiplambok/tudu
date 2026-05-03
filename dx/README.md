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
