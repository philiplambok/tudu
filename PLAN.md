# Plan: Parallel Worktree Support for dx/ Scripts

## Problem

The `dx/` scripts assume a single running environment. Four hard conflicts prevent
multiple git worktrees from running simultaneously:

| Conflict | Current value | Impact |
|---|---|---|
| Hardcoded container names | `tudu-app`, `tudu-postgres` | Docker rejects duplicate names |
| Hardcoded host ports | `8080`, `5434`, `3011` | Port bind fails on second worktree |
| Hardcoded volume names | `postgres_dev_data`, `go_mod_cache`, ... | Branches share Postgres data |
| `container_name:` directives | override Compose project isolation | `-p` flag alone cannot help |

## Design Approach: Auto-Namespacing via Workspace Directory Name

**Zero config.** Every `dx/` command auto-detects its workspace identity from the
git worktree directory name (e.g., `khartoum`, `london`). No env vars to set, no
init step, no files to generate.

### How it works

1. `dx/_common` reads the workspace name: `basename $(git rev-parse --show-toplevel)`
2. That name becomes the Docker Compose **project name**: `tudu-<workspace>`  
   Docker Compose already uses the project name to namespace containers, networks,
   and volumes automatically — we just need to unlock it.
3. A deterministic **port offset** (0–49) is derived from a hash of the workspace
   name. Each worktree gets its own stable port set.
4. `docker-compose.dev.yml` is updated to read ports from env vars, and all
   hardcoded `container_name:` directives are removed.

### Port derivation

```
offset = cksum(workspace_name) mod 50
app_port     = 8080 + offset    # e.g. khartoum → 8080+N
postgres_port = 5434 + offset
dbgate_port  = 3011 + offset
```

`cksum` is available on both macOS and Linux, so no platform issues.  
With a range of 50, up to 50 parallel worktrees have zero overlap.

### What Docker Compose project namespacing gives us for free

When `COMPOSE_PROJECT_NAME=tudu-khartoum`:

- Containers become `tudu-khartoum-app-1`, `tudu-khartoum-postgres-1`
- Volumes become `tudu-khartoum_postgres_dev_data`, `tudu-khartoum_go_mod_cache`
- Network becomes `tudu-khartoum_tudu-network`

No extra logic needed — removing `container_name:` is the only required compose change
(plus parametrizing the ports).

### Image sharing

`tudu:dev` image is **shared** across all worktrees — it is read-only and branch-
agnostic (just the Go toolchain + air). This is desirable: one build serves all branches.

---

## Files Changed

### 1. `dx/_common`

Add workspace detection + port derivation + a shared `COMPOSE_ARGS` variable.
All scripts swap `docker compose -f "$COMPOSE_FILE"` for `docker compose $COMPOSE_ARGS`.

```bash
# --- new additions ---
WORKSPACE=$(basename "$(git rev-parse --show-toplevel 2>/dev/null || pwd)")
_HASH=$(echo -n "$WORKSPACE" | cksum | awk '{print $1}')
WORKSPACE_OFFSET=$((_HASH % 50))

COMPOSE_PROJECT="tudu-${WORKSPACE}"
COMPOSE_ARGS="-f ${COMPOSE_FILE} -p ${COMPOSE_PROJECT}"

export APP_PORT=$((8080 + WORKSPACE_OFFSET))
export PG_PORT=$((5434 + WORKSPACE_OFFSET))
export DBGATE_PORT=$((3011 + WORKSPACE_OFFSET))
```

`check_container_running` updates its docker compose call to use `$COMPOSE_ARGS`.

### 2. `docker-compose.dev.yml`

Two changes:
- Remove all `container_name:` lines (4 total)
- Parameterize host ports using the exported env vars

```yaml
# Before
ports:
  - "8080:8080"
container_name: tudu-app

# After
ports:
  - "${APP_PORT:-8080}:8080"
# container_name removed
```

Same pattern for `postgres` (`${PG_PORT:-5434}:5432`) and `dbgate`
(`${DBGATE_PORT:-3011}:3000`).

The `ENV_DATABASE_SOURCE` in the app service continues to reference `postgres` by
service name — this still works because both containers are on the same Compose
project network.

### 3. All `dx/*` scripts

Every script that calls `docker compose -f "$COMPOSE_FILE"` is updated to use
`docker compose $COMPOSE_ARGS` instead. Affected scripts:

- `dx/start`
- `dx/stop`
- `dx/dev`
- `dx/test`
- `dx/lint`
- `dx/shell`
- `dx/exec`
- `dx/db`
- `dx/generate`
- `dx/dbgate`
- `dx/status`
- `dx/logs`
- `dx/clean`

### 4. `dx/start` — show workspace info on startup

```
🚀 Starting [khartoum] → app: http://localhost:8093 · postgres: localhost:5451
```

### 5. `dx/status` — show workspace context

Prepend workspace name and port info so it's clear which stack is which.

### 6. `dx/README.md`

Add a "Parallel Worktrees" section explaining how namespacing works and what ports
each workspace gets.

---

## Implementation Steps

- [ ] 1. Update `dx/_common`: add workspace detection, port derivation, `COMPOSE_ARGS`
- [ ] 2. Update `docker-compose.dev.yml`: remove `container_name:` directives, parameterize ports
- [ ] 3. Update `dx/start`: use `$COMPOSE_ARGS`, print workspace name + resolved ports
- [ ] 4. Update `dx/stop` and `dx/clean`: use `$COMPOSE_ARGS`
- [ ] 5. Update `dx/status` and `dx/logs`: use `$COMPOSE_ARGS`, show workspace context
- [ ] 6. Update `dx/dev`, `dx/shell`, `dx/exec`: use `$COMPOSE_ARGS`
- [ ] 7. Update `dx/test`, `dx/lint`, `dx/generate`, `dx/db`: use `$COMPOSE_ARGS`
- [ ] 8. Update `dx/dbgate`: use `$COMPOSE_ARGS`
- [ ] 9. Update `dx/README.md` with parallel worktree documentation
- [ ] 10. Smoke test: `./dx/start` works in two different worktrees simultaneously

---

## What Does NOT Change

- The Go source code and application — no changes needed
- The `tudu:dev` image build process — shared across worktrees
- Database connection string inside the app — still `postgres:5432` within the network
- Migration files — each worktree has its own volume, migrations run independently
- `dx/build` — builds the shared image, works as before

## Trade-offs and Risks

| Risk | Mitigation |
|---|---|
| Hash collision (two workspaces → same offset) | Range is 0–49; collisions detectable from port conflict on `start` |
| `cksum` output varies if workspace name changes | Workspace names (dir names) are stable in Conductor |
| Developers accustomed to `tudu-app` container name | `dx/shell`, `dx/exec` still work — they go through Compose, not container name |
| Old running containers with hardcoded names | `./dx/stop --remove` in old worktrees cleans them before upgrade |
