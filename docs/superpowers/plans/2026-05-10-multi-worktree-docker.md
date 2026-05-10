# Multi-Worktree Docker Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable multiple git worktrees of this project to run their own isolated Docker environments simultaneously with no port conflicts and no shared container/volume state.

**Architecture:** Each worktree carries a `dx/workspace.env` file (gitignored) that sets `WORKSPACE_NAME` and `PORT_OFFSET`. The `_common` script reads this file, derives `COMPOSE_PROJECT_NAME` and port variables, then exports them so every Docker Compose invocation is automatically scoped to the correct project. Docker Compose's built-in project namespacing handles container, volume, and network isolation; env-var substitution in `docker-compose.dev.yml` handles port isolation. When no `workspace.env` is present, defaults are derived from the current git branch name so the tool works out of the box.

**Tech Stack:** Bash, Docker Compose (COMPOSE_PROJECT_NAME, env-var substitution in compose files)

---

## How Isolation Works

| Concern | Mechanism |
|---|---|
| Container names | `COMPOSE_PROJECT_NAME` → Docker Compose prefixes all containers with project name |
| Volume names | `COMPOSE_PROJECT_NAME` → Docker Compose prefixes all volumes with project name |
| Network names | `COMPOSE_PROJECT_NAME` → Docker Compose prefixes the network with project name |
| Host ports | `${APP_PORT:-8080}`, `${POSTGRES_PORT:-5434}`, `${DBGATE_PORT:-3011}` in compose file |
| Per-worktree config | `dx/workspace.env` (gitignored) stores `WORKSPACE_NAME` + `PORT_OFFSET` |

Port offsets per worktree (each active worktree must use a unique offset):

| Offset | APP_PORT | POSTGRES_PORT | DBGATE_PORT |
|---|---|---|---|
| 0 (default) | 8080 | 5434 | 3011 |
| 1 | 8081 | 5435 | 3012 |
| 2 | 8082 | 5436 | 3013 |

---

## File Map

| File | Action | Purpose |
|---|---|---|
| `docker-compose.dev.yml` | Modify | Replace hardcoded ports with env vars; remove `container_name` directives |
| `dx/_common` | Modify | Load `workspace.env`, compute `COMPOSE_PROJECT_NAME` and port vars, export all |
| `dx/workspace` | Create | `init` and `show` subcommands for workspace config management |
| `dx/workspace.env.example` | Create | Committed example documenting available settings |
| `dx/dev` | Modify | Use `${APP_PORT}` in startup URL echo |
| `dx/dbgate` | Modify | Use `${DBGATE_PORT}` instead of hardcoded `3011` |
| `dx/start` | Modify | Show workspace name and ports in startup output |
| `dx/README.md` | Modify | Document workspace commands and multi-worktree workflow |

---

### Task 1: Parameterize docker-compose.dev.yml

**Files:**
- Modify: `docker-compose.dev.yml`

Removes explicit `container_name` directives (Docker Compose will auto-name containers as `{project}-{service}-1`, scoped per `COMPOSE_PROJECT_NAME`) and replaces hardcoded host ports with env vars.

- [ ] **Step 1: Replace docker-compose.dev.yml**

Overwrite `docker-compose.dev.yml` with:

```yaml
services:
  app:
    build:
      context: .
      dockerfile: dx/Dockerfile
    image: tudu:dev
    ports:
      - "${APP_PORT:-8080}:8080"
    volumes:
      - .:/app
      - go_mod_cache:/go/pkg/mod
      - go_build_cache:/root/.cache/go-build
    environment:
      - ENV_ENV=local
      - GO111MODULE=on
      - CGO_ENABLED=1
      - ENV_DATABASE_SOURCE=postgresql://postgres:postgres@postgres:5432/tudu?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - tudu-network
    command: ["sleep", "infinity"]

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: tudu
      POSTGRES_INITDB_ARGS: "--encoding=UTF8 --lc-collate=C --lc-ctype=C"
    ports:
      - "${POSTGRES_PORT:-5434}:5432"
    volumes:
      - postgres_dev_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d tudu"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 20s
    networks:
      - tudu-network

  dbgate:
    image: dbgate/dbgate:latest
    profiles: ["dbgate"]
    restart: unless-stopped
    ports:
      - "${DBGATE_PORT:-3011}:3000"
    environment:
      CONNECTIONS: postgres
      LABEL_postgres: PostgreSQL (tudu dev)
      SERVER_postgres: postgres
      PORT_postgres: 5432
      USER_postgres: postgres
      PASSWORD_postgres: postgres
      DATABASE_postgres: tudu
      ENGINE_postgres: postgres@dbgate-plugin-postgres
    volumes:
      - dbgate_dev_data:/root/.dbgate
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - tudu-network

volumes:
  postgres_dev_data:
  go_mod_cache:
  go_build_cache:
  dbgate_dev_data:

networks:
  tudu-network:
    driver: bridge
```

- [ ] **Step 2: Verify Docker Compose parses the file correctly**

Run from project root (containers do not need to be running):

```bash
docker compose -f docker-compose.dev.yml config
```

Expected: YAML dump with `8080`, `5434`, `3011` substituted as default port values. No errors.

- [ ] **Step 3: Commit**

```bash
git add docker-compose.dev.yml
git commit -m "parameterize docker-compose ports and remove container_name"
```

---

### Task 2: Enhance dx/_common with workspace detection

**Files:**
- Modify: `dx/_common`

After this task every dx script that sources `_common` automatically has `COMPOSE_PROJECT_NAME`, `APP_PORT`, `POSTGRES_PORT`, and `DBGATE_PORT` in its environment. Docker Compose reads `COMPOSE_PROJECT_NAME` automatically.

`${BASH_SOURCE[0]}` inside `_common` always resolves to `_common`'s own path (not the sourcing script's path), so `SCRIPT_DIR` reliably points to the `dx/` directory.

- [ ] **Step 1: Replace dx/_common**

Overwrite `dx/_common` with:

```bash
#!/bin/bash
# Shared environment for dx/* scripts.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="docker-compose.dev.yml"
IMAGE_NAME="tudu:dev"

# Load per-worktree config if present (gitignored via *.env pattern).
WORKSPACE_ENV="$SCRIPT_DIR/workspace.env"
if [ -f "$WORKSPACE_ENV" ]; then
    # shellcheck source=/dev/null
    source "$WORKSPACE_ENV"
fi

# Derive workspace name from git branch when workspace.env is absent.
# Sanitise: lowercase, replace non-alphanumeric chars (except dash/underscore) with dash.
if [ -z "${WORKSPACE_NAME:-}" ]; then
    _branch="$(git -C "$PROJECT_ROOT" branch --show-current 2>/dev/null || echo 'default')"
    WORKSPACE_NAME="$(echo "$_branch" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9_' '-' | sed 's/-*$//')"
    WORKSPACE_NAME="${WORKSPACE_NAME:-default}"
fi

PORT_OFFSET="${PORT_OFFSET:-0}"
APP_PORT=$((8080 + PORT_OFFSET))
POSTGRES_PORT=$((5434 + PORT_OFFSET))
DBGATE_PORT=$((3011 + PORT_OFFSET))

# COMPOSE_PROJECT_NAME scopes all containers, volumes, and networks to this workspace.
export COMPOSE_PROJECT_NAME="tudu-${WORKSPACE_NAME}"
export APP_PORT
export POSTGRES_PORT
export DBGATE_PORT

check_docker() {
    if ! command -v docker &> /dev/null; then
        echo "❌ Docker is not installed or not in PATH"
        exit 1
    fi
    if ! docker info &> /dev/null; then
        echo "❌ Docker daemon is not running"
        exit 1
    fi
    if ! docker compose version &> /dev/null; then
        echo "❌ docker compose plugin not available"
        exit 1
    fi
}

check_image_exists() {
    if ! docker image inspect "$IMAGE_NAME" &> /dev/null; then
        echo "❌ Docker image '$IMAGE_NAME' not found"
        echo "💡 Build it first: ./dx/build"
        exit 1
    fi
}

check_container_running() {
    if ! docker compose -f "$COMPOSE_FILE" ps --services --filter "status=running" 2>/dev/null | grep -q "^app$"; then
        echo "❌ Development container is not running (workspace: ${COMPOSE_PROJECT_NAME})"
        echo "💡 Start it first: ./dx/start"
        exit 1
    fi
}
```

- [ ] **Step 2: Verify _common exports correct values without workspace.env**

Run from project root (on branch `philiplambok/las-vegas` with no `dx/workspace.env`):

```bash
bash -c 'source dx/_common && echo "project=$COMPOSE_PROJECT_NAME app=$APP_PORT pg=$POSTGRES_PORT dbgate=$DBGATE_PORT"'
```

Expected:
```
project=tudu-philiplambok-las-vegas app=8080 pg=5434 dbgate=3011
```

- [ ] **Step 3: Verify _common exports correct values with workspace.env**

```bash
printf 'WORKSPACE_NAME=feature-x\nPORT_OFFSET=2\n' > dx/workspace.env
bash -c 'source dx/_common && echo "project=$COMPOSE_PROJECT_NAME app=$APP_PORT pg=$POSTGRES_PORT dbgate=$DBGATE_PORT"'
rm dx/workspace.env
```

Expected:
```
project=tudu-feature-x app=8082 pg=5436 dbgate=3013
```

- [ ] **Step 4: Commit**

```bash
git add dx/_common
git commit -m "add workspace detection to dx/_common for multi-worktree isolation"
```

---

### Task 3: Create dx/workspace management command

**Files:**
- Create: `dx/workspace`

- [ ] **Step 1: Create dx/workspace**

Write `dx/workspace`:

```bash
#!/bin/bash
source "$(dirname "$0")/_common"

usage() {
    cat <<EOF
Usage: ./dx/workspace <command>

Commands:
  init [name] [port-offset]   Create dx/workspace.env for this worktree
  show                        Print active workspace name and port assignments
EOF
}

case "${1:-}" in
    init)
        NAME="${2:-}"
        OFFSET="${3:-0}"

        if [ -z "$NAME" ]; then
            _branch="$(git -C "$PROJECT_ROOT" branch --show-current 2>/dev/null || echo 'default')"
            NAME="$(echo "$_branch" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9_' '-' | sed 's/-*$//')"
            NAME="${NAME:-default}"
        fi

        cat > "$SCRIPT_DIR/workspace.env" <<ENVEOF
# dx/workspace.env — per-worktree Docker isolation config. Gitignored.
# Edit manually or regenerate with: ./dx/workspace init [name] [offset]
WORKSPACE_NAME=${NAME}
PORT_OFFSET=${OFFSET}
ENVEOF
        echo "✅ Created dx/workspace.env"
        echo ""
        echo "   Workspace:     tudu-${NAME}"
        echo "   APP_PORT:      $((8080 + OFFSET))  → http://localhost:$((8080 + OFFSET))"
        echo "   POSTGRES_PORT: $((5434 + OFFSET))  → localhost:$((5434 + OFFSET))"
        echo "   DBGATE_PORT:   $((3011 + OFFSET))  → http://localhost:$((3011 + OFFSET))"
        ;;

    show)
        echo "Workspace: ${COMPOSE_PROJECT_NAME}"
        echo "  APP_PORT:      ${APP_PORT}  → http://localhost:${APP_PORT}"
        echo "  POSTGRES_PORT: ${POSTGRES_PORT}  → localhost:${POSTGRES_PORT}"
        echo "  DBGATE_PORT:   ${DBGATE_PORT}  → http://localhost:${DBGATE_PORT}"
        if [ -f "$WORKSPACE_ENV" ]; then
            echo "  Config:        dx/workspace.env (loaded)"
        else
            echo "  Config:        dx/workspace.env (absent — derived from git branch)"
        fi
        ;;

    ""|"-h"|"--help")
        usage ;;

    *)
        echo "❌ Unknown command: ${1}"; usage; exit 1 ;;
esac
```

- [ ] **Step 2: Make it executable**

```bash
chmod +x dx/workspace
```

- [ ] **Step 3: Verify init creates workspace.env with explicit args**

```bash
./dx/workspace init my-feature 1
cat dx/workspace.env
```

Expected output of `init`:
```
✅ Created dx/workspace.env

   Workspace:     tudu-my-feature
   APP_PORT:      8081  → http://localhost:8081
   POSTGRES_PORT: 5435  → localhost:5435
   DBGATE_PORT:   3012  → http://localhost:3012
```

Expected content of `dx/workspace.env`:
```
# dx/workspace.env — per-worktree Docker isolation config. Gitignored.
# Edit manually or regenerate with: ./dx/workspace init [name] [offset]
WORKSPACE_NAME=my-feature
PORT_OFFSET=1
```

- [ ] **Step 4: Verify show reads workspace.env**

```bash
./dx/workspace show
```

Expected:
```
Workspace: tudu-my-feature
  APP_PORT:      8081  → http://localhost:8081
  POSTGRES_PORT: 5435  → localhost:5435
  DBGATE_PORT:   3012  → http://localhost:3012
  Config:        dx/workspace.env (loaded)
```

- [ ] **Step 5: Verify show falls back to branch name without workspace.env**

```bash
rm dx/workspace.env
./dx/workspace show
```

Expected (on branch `philiplambok/las-vegas`):
```
Workspace: tudu-philiplambok-las-vegas
  APP_PORT:      8080  → http://localhost:8080
  POSTGRES_PORT: 5434  → localhost:5434
  DBGATE_PORT:   3011  → http://localhost:3011
  Config:        dx/workspace.env (absent — derived from git branch)
```

- [ ] **Step 6: Verify init without args uses branch name**

```bash
./dx/workspace init
cat dx/workspace.env
rm dx/workspace.env
```

Expected: `WORKSPACE_NAME=philiplambok-las-vegas` (sanitised branch name).

- [ ] **Step 7: Commit**

```bash
git add dx/workspace
git commit -m "add dx/workspace command for multi-worktree config management"
```

---

### Task 4: Update dx/dev and dx/dbgate to show correct URLs

**Files:**
- Modify: `dx/dev`
- Modify: `dx/dbgate`

- [ ] **Step 1: Update dx/dev**

In `dx/dev`, replace both hardcoded `8080` URL strings.

Change:
```bash
echo "🚀 Starting HTTP server (no live reload) on http://localhost:8080"
```
To:
```bash
echo "🚀 Starting HTTP server (no live reload) on http://localhost:${APP_PORT}"
```

Change:
```bash
echo "🚀 Starting HTTP server with live reload on http://localhost:8080"
```
To:
```bash
echo "🚀 Starting HTTP server with live reload on http://localhost:${APP_PORT}"
```

- [ ] **Step 2: Update dx/dbgate**

In `dx/dbgate`, change:
```bash
URL="http://localhost:3011"
```
To:
```bash
URL="http://localhost:${DBGATE_PORT}"
```

- [ ] **Step 3: Verify both files reference the dynamic vars**

```bash
grep -n 'APP_PORT\|DBGATE_PORT' dx/dev dx/dbgate
```

Expected:
```
dx/dev:7:    echo "🚀 Starting HTTP server (no live reload) on http://localhost:${APP_PORT}"
dx/dev:10:    echo "🚀 Starting HTTP server with live reload on http://localhost:${APP_PORT}"
dx/dbgate:6:URL="http://localhost:${DBGATE_PORT}"
```
(line numbers may differ)

- [ ] **Step 4: Verify the URL expands correctly**

```bash
printf 'WORKSPACE_NAME=test\nPORT_OFFSET=1\n' > dx/workspace.env
bash -c 'source dx/_common && echo "app=http://localhost:${APP_PORT} dbgate=http://localhost:${DBGATE_PORT}"'
rm dx/workspace.env
```

Expected:
```
app=http://localhost:8081 dbgate=http://localhost:3012
```

- [ ] **Step 5: Commit**

```bash
git add dx/dev dx/dbgate
git commit -m "use dynamic port vars in dx/dev and dx/dbgate URLs"
```

---

### Task 5: Update dx/start to show workspace and port info

**Files:**
- Modify: `dx/start`

- [ ] **Step 1: Replace dx/start**

Overwrite `dx/start` with:

```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker
check_image_exists

echo "🚀 Starting containers (workspace: ${COMPOSE_PROJECT_NAME})..."
docker compose -f "$COMPOSE_FILE" down --remove-orphans 2>/dev/null || true
docker compose -f "$COMPOSE_FILE" up -d

echo ""
docker compose -f "$COMPOSE_FILE" ps
echo ""
echo "✅ Up (workspace: ${COMPOSE_PROJECT_NAME})"
echo "   HTTP API:      http://localhost:${APP_PORT}"
echo "   PostgreSQL:    localhost:${POSTGRES_PORT}"
echo ""
echo "   ./dx/db migrate    Apply migrations"
echo "   ./dx/dev           Start HTTP server with live reload"
```

- [ ] **Step 2: Verify start references the dynamic vars**

```bash
grep -n 'COMPOSE_PROJECT_NAME\|APP_PORT\|POSTGRES_PORT' dx/start
```

Expected: at least 3 matching lines showing each var.

- [ ] **Step 3: Commit**

```bash
git add dx/start
git commit -m "show workspace name and ports in dx/start output"
```

---

### Task 6: Create dx/workspace.env.example

**Files:**
- Create: `dx/workspace.env.example`

The `.gitignore` pattern `*.env` does NOT match `*.env.example`, so this file is committable.

- [ ] **Step 1: Create dx/workspace.env.example**

Write `dx/workspace.env.example`:

```bash
# dx/workspace.env — copy to dx/workspace.env and edit per worktree.
# Or generate automatically with: ./dx/workspace init [name] [port-offset]
#
# WORKSPACE_NAME pins the Docker Compose project name to "tudu-<name>".
# When absent, the current git branch name is used automatically.
#
# PORT_OFFSET shifts all host ports by this integer:
#   APP_PORT      = 8080 + PORT_OFFSET
#   POSTGRES_PORT = 5434 + PORT_OFFSET
#   DBGATE_PORT   = 3011 + PORT_OFFSET
#
# Offset examples (each active worktree must use a unique offset):
#   0 → 8080, 5434, 3011  (default)
#   1 → 8081, 5435, 3012
#   2 → 8082, 5436, 3013
#
# Run ./dx/workspace show to confirm active settings.

WORKSPACE_NAME=your-name-here
PORT_OFFSET=0
```

- [ ] **Step 2: Confirm it is NOT gitignored**

```bash
git check-ignore -v dx/workspace.env.example
```

Expected: no output (the file is not ignored).

- [ ] **Step 3: Confirm dx/workspace.env IS gitignored**

```bash
touch dx/workspace.env
git check-ignore -v dx/workspace.env
rm dx/workspace.env
```

Expected: output shows `*.env` rule matches `dx/workspace.env`.

- [ ] **Step 4: Commit**

```bash
git add dx/workspace.env.example
git commit -m "add dx/workspace.env.example documenting multi-worktree config"
```

---

### Task 7: Update dx/README.md

**Files:**
- Modify: `dx/README.md`

- [ ] **Step 1: Add workspace commands to the Development table**

In the Development table, add a new row after `./dx/lint`:

```markdown
| `./dx/workspace show` | Show current workspace config and port assignments |
| `./dx/workspace init [name] [offset]` | Create `dx/workspace.env` for this worktree |
```

- [ ] **Step 2: Add Multi-Worktree section before DBGate**

Add a new section between "Code Generation" and "DBGate (optional web GUI)":

```markdown
### Multi-Worktree (parallel workspaces)

Each worktree runs its own isolated Docker stack — separate containers, volumes, and ports — via a `dx/workspace.env` config file (gitignored).

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

See `dx/workspace.env.example` for all available settings.
```

- [ ] **Step 3: Verify the section appears in the file**

```bash
grep -A 5 "Multi-Worktree" dx/README.md
```

Expected: the heading and first few lines of the new section.

- [ ] **Step 4: Commit**

```bash
git add dx/README.md
git commit -m "document multi-worktree Docker workflow in dx/README.md"
```

---

## Migration Note for Existing Containers

The old `docker-compose.dev.yml` hardcoded `container_name: tudu-app` etc. Those containers will not be found or stopped by the new scripts (which now use `COMPOSE_PROJECT_NAME`-scoped names). After upgrading, clean up the old containers once:

```bash
# Stop old hardcoded containers (safe to run even if they don't exist)
docker rm -f tudu-app tudu-postgres tudu-dbgate 2>/dev/null || true
docker volume rm tudu_postgres_dev_data tudu_go_mod_cache tudu_go_build_cache tudu_dbgate_dev_data 2>/dev/null || true
```

Then start fresh with `./dx/start`.
