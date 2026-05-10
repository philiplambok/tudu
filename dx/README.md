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

Ports are auto-assigned per worktree using a deterministic offset derived from the
workspace directory name. The defaults below apply when running a single worktree.

| Service | Default host port | Container port |
|---|---|---|
| HTTP server | 8080 + offset | 8080 |
| PostgreSQL | 5434 + offset | 5432 |
| DBGate | 3011 + offset | 3000 |
| Swagger UI | same as HTTP | — |

Run `./dx/status` to see the actual ports for the current workspace.

## Parallel Worktrees

Each git worktree runs its own fully isolated Docker stack — separate containers,
volumes, and ports. No configuration required.

**How it works:** `dx/_common` reads the workspace directory name (e.g. `khartoum`),
hashes it to a stable offset (0–49), and uses that to:

- Set the Docker Compose project name to `tudu-<workspace>` — this namespaces
  containers (`tudu-khartoum-app-1`), volumes, and networks automatically.
- Derive unique host ports for the app, Postgres, and DBGate services.

**Running two worktrees simultaneously:**

```bash
# Terminal 1 — worktree "khartoum"
cd ~/conductor/workspaces/tudu/khartoum
./dx/start      # prints: ✅ [khartoum] Up. App: http://localhost:808X

# Terminal 2 — worktree "london"
cd ~/conductor/workspaces/tudu/london
./dx/start      # prints: ✅ [london] Up. App: http://localhost:808Y
```

Each worktree has its own Postgres data volume — migrations run independently.
The `tudu:dev` Docker image is shared (built once, used by all worktrees).
