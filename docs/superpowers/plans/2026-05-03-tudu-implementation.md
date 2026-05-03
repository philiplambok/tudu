# Tudu Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a clean-architecture Go Todoist API (multi-user, JWT-authenticated, PostgreSQL-backed) that demonstrates the domain package pattern, service/repository interfaces, OpenAPI spec-first design, and the adapter/port pattern.

**Architecture:** Each domain (`user`, `task`) lives in `internal/<domain>/` with six focused files: domain logic, DTOs, service interface+impl, repository interface+impl, HTTP handler, and endpoint wiring. The `pkg/avatar/` package demonstrates the port/adapter pattern — a `Provider` interface with a Gravatar implementation (production) and a mock (local/sandbox), selected at startup by `cfg.Env`.

**Tech Stack:** Go 1.26, PostgreSQL 16, chi v5, GORM, goose (migrations), cobra/viper (CLI/config), golang-jwt/jwt/v5, bcrypt, oapi-codegen (OpenAPI spec-first), swaggo/http-swagger, Docker Compose, air (live reload)

---

## File Map

```
tudu/
├── api/openapi.yml
├── oapi_codegen.yml
├── cmd/
│   ├── cmd.go
│   ├── migrate.go
│   └── serve.go
├── config.example.yml
├── config.yml                        (gitignored)
├── db/migrations/
│   ├── 20260503000001_init.sql
│   ├── 20260503000002_create_users.sql
│   └── 20260503000003_create_tasks.sql
├── dx/
│   ├── _common
│   ├── Dockerfile
│   ├── README.md
│   ├── build, start, stop, dev, shell, exec
│   ├── test, lint, logs, status, clean
│   ├── generate, db, dbgate, help
├── docker-compose.dev.yml
├── .air.toml
├── .gitignore
├── .dockerignore
├── go.mod
├── main.go
├── internal/
│   ├── config.go                     Config struct (Env, Log, HTTPServer, Database, JWT)
│   ├── context.go                    WithUserID / UserIDFromContext helpers
│   ├── swagger/
│   │   └── swagger.go               Serves /swagger.json + /swagger/* UI
│   ├── transport/
│   │   └── rest.go                  chi server, JWT middleware, route mounting
│   ├── user/
│   │   ├── domain.go                sentinel errors, ValidateRegister, ValidateLogin
│   │   ├── dto.go                   DTOs + ValidationError
│   │   ├── repository.go            Repository interface + GORM impl
│   │   ├── service.go               Service interface + impl (register, login, me)
│   │   ├── service_test.go          unit tests (mock repo)
│   │   ├── handler.go               HTTP handlers
│   │   └── endpoint.go              route wiring; NewEndpoint(db, avatarProvider, jwtSecret)
│   └── task/
│       ├── domain.go                sentinel errors, ValidateCreate, ValidateUpdate, status constants
│       ├── dto.go                   DTOs + ValidationError
│       ├── repository.go            Repository interface + GORM impl
│       ├── service.go               Service interface + impl
│       ├── service_test.go          unit tests (mock repo)
│       ├── handler.go               HTTP handlers
│       └── endpoint.go              route wiring; NewEndpoint(db)
└── pkg/
    ├── avatar/
    │   ├── provider.go              Provider interface
    │   ├── gravatar.go              real impl: Gravatar URL
    │   └── mock.go                  mock impl: static DiceBear URL
    └── openapi/v1/
        └── openapi.gen.go           generated — do not edit
```

---

## Task 1: Initialize Go module and base project files

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `.gitignore`
- Create: `.dockerignore`

- [ ] **Step 1: Create the project directory and initialize the Go module**

```bash
mkdir -p /Users/mekari/Personal/tudu
cd /Users/mekari/Personal/tudu
git init
```

- [ ] **Step 2: Write `go.mod`**

```
module github.com/philiplambok/tudu

go 1.26

require (
	github.com/getkin/kin-openapi v0.137.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-chi/render v1.0.3
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.9.2
	github.com/pressly/goose/v3 v3.27.1
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	github.com/swaggo/files v1.0.1
	github.com/swaggo/http-swagger v1.3.4
	golang.org/x/crypto v0.38.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
)

tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
```

- [ ] **Step 3: Write `main.go`**

```go
package main

import "github.com/philiplambok/tudu/cmd"

func main() {
	cmd.Execute()
}
```

- [ ] **Step 4: Write `.gitignore`**

```
config.yml
tmp/
*.env
```

- [ ] **Step 5: Write `.dockerignore`**

```
tmp/
.git/
config.yml
```

- [ ] **Step 6: Commit**

```bash
git add go.mod main.go .gitignore .dockerignore
git commit -m "chore: initialize go module and base files"
```

---

## Task 2: dx toolkit and Docker environment

**Files:**
- Create: `dx/_common`, `dx/Dockerfile`, `dx/build`, `dx/start`, `dx/stop`, `dx/dev`
- Create: `dx/shell`, `dx/exec`, `dx/test`, `dx/lint`, `dx/logs`, `dx/status`
- Create: `dx/clean`, `dx/generate`, `dx/db`, `dx/dbgate`, `dx/help`, `dx/README.md`
- Create: `docker-compose.dev.yml`, `.air.toml`

- [ ] **Step 1: Write `dx/_common`**

```bash
#!/bin/bash
# Shared functions for dx/* scripts.

COMPOSE_FILE="docker-compose.dev.yml"
IMAGE_NAME="tudu:dev"

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
    if ! docker compose -f "$COMPOSE_FILE" ps --services --filter "status=running" | grep -q "^app$"; then
        echo "❌ Development container is not running"
        echo "💡 Start it first: ./dx/start"
        exit 1
    fi
}
```

- [ ] **Step 2: Write `dx/Dockerfile`**

```dockerfile
# Development Dockerfile for Tudu
FROM golang:1.26-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        git \
        postgresql-client \
        tzdata \
    && rm -rf /var/lib/apt/lists/*

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOPATH=/go \
    PATH="/go/bin:${PATH}"

RUN go install github.com/air-verse/air@latest \
    && go install github.com/pressly/goose/v3/cmd/goose@latest

WORKDIR /app

EXPOSE 8080

CMD ["sleep", "infinity"]
```

- [ ] **Step 3: Write `docker-compose.dev.yml`**

```yaml
services:
  app:
    build:
      context: .
      dockerfile: dx/Dockerfile
    image: tudu:dev
    container_name: tudu-app
    ports:
      - "8080:8080"
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
    container_name: tudu-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: tudu
      POSTGRES_INITDB_ARGS: "--encoding=UTF8 --lc-collate=C --lc-ctype=C"
    ports:
      - "5434:5432"
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
    container_name: tudu-dbgate
    profiles: ["dbgate"]
    restart: unless-stopped
    ports:
      - "3011:3000"
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

- [ ] **Step 4: Write `.air.toml`**

```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/tudu ./main.go"
  bin = "./tmp/tudu"
  full_bin = "./tmp/tudu serve"
  include_ext = ["go", "yml", "yaml"]
  exclude_dir = ["tmp", "vendor", "db/migrations", "dx", ".git"]
  delay = 500
  stop_on_error = true

[log]
  time = true

[color]
  main = "magenta"
  watcher = "cyan"
  build = "yellow"
  runner = "green"

[misc]
  clean_on_exit = true
```

- [ ] **Step 5: Write remaining dx scripts**

`dx/build`:
```bash
#!/bin/bash
set -e
source "$(dirname "$0")/_common"
check_docker

NO_CACHE=""
for arg in "$@"; do
    case $arg in
        --no-cache) NO_CACHE="--no-cache" ;;
        --help|-h)
            echo "Usage: ./dx/build [--no-cache]"
            exit 0 ;;
    esac
done

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "🐳 Building $IMAGE_NAME ..."
docker build $NO_CACHE -f dx/Dockerfile -t "$IMAGE_NAME" .
echo "✅ Image built: $IMAGE_NAME"
echo "🚀 Next: ./dx/start"
```

`dx/start`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker
check_image_exists

echo "🚀 Starting containers..."
docker compose -f "$COMPOSE_FILE" down --remove-orphans 2>/dev/null || true
docker compose -f "$COMPOSE_FILE" up -d

echo ""
docker compose -f "$COMPOSE_FILE" ps
echo ""
echo "✅ Up. Next:"
echo "   ./dx/db migrate    Apply migrations"
echo "   ./dx/dev           Start HTTP server with live reload"
```

`dx/stop`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker

REMOVE=false
for arg in "$@"; do
    case $arg in
        --remove) REMOVE=true ;;
    esac
done

if [ "$REMOVE" = true ]; then
    echo "🧹 Stopping and removing containers + volumes..."
    docker compose -f "$COMPOSE_FILE" down -v --remove-orphans
else
    echo "🛑 Stopping containers..."
    docker compose -f "$COMPOSE_FILE" stop
fi
echo "✅ Stopped."
```

`dx/dev`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker
check_container_running

if [ "$1" = "--no-reload" ]; then
    echo "🚀 Starting HTTP server (no live reload) on http://localhost:8080"
    docker compose -f "$COMPOSE_FILE" exec app go run main.go serve
else
    echo "🚀 Starting HTTP server with live reload on http://localhost:8080"
    docker compose -f "$COMPOSE_FILE" exec app air -c .air.toml
fi
```

`dx/shell`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker
check_container_running
docker compose -f "$COMPOSE_FILE" exec app bash
```

`dx/exec`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker
check_container_running

if [ $# -eq 0 ]; then
    echo "Usage: ./dx/exec <command> [args...]"
    exit 1
fi
docker compose -f "$COMPOSE_FILE" exec app "$@"
```

`dx/test`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker
check_container_running

if [ $# -eq 0 ]; then
    set -- "./..."
fi
docker compose -f "$COMPOSE_FILE" exec app go test "$@"
```

`dx/lint`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker
check_container_running

echo "🔍 go vet ./..."
docker compose -f "$COMPOSE_FILE" exec app go vet ./...
```

`dx/logs`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker
docker compose -f "$COMPOSE_FILE" logs -f --tail=200 "$@"
```

`dx/status`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker
docker compose -f "$COMPOSE_FILE" ps
```

`dx/clean`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"
check_docker

echo "⚠️  This will remove containers, volumes, and the '$IMAGE_NAME' image."
read -r -p "Continue? [y/N] " ans
case "$ans" in
    y|Y|yes|YES) ;;
    *) echo "Aborted."; exit 0 ;;
esac

docker compose -f "$COMPOSE_FILE" --profile dbgate down -v --remove-orphans 2>/dev/null || true
docker image rm "$IMAGE_NAME" 2>/dev/null || true
echo "✅ Cleaned. Run './dx/build' to rebuild."
```

`dx/generate`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"

SUBCOMMAND="${1:-}"
shift || true

if [ -z "$SUBCOMMAND" ] || [ "$SUBCOMMAND" = "-h" ] || [ "$SUBCOMMAND" = "--help" ]; then
    echo "Usage: ./dx/generate <command>"
    echo "Commands:"
    echo "  openapi     Generate Go types from api/openapi.yml"
    exit 0
fi

check_docker
check_container_running

case "$SUBCOMMAND" in
    openapi)
        echo "📝 Generating OpenAPI code from api/openapi.yml..."
        docker compose -f "$COMPOSE_FILE" exec app \
            go tool oapi-codegen -config ./oapi_codegen.yml ./api/openapi.yml
        echo "✅ Generated: pkg/openapi/v1/openapi.gen.go"
        ;;
    *)
        echo "❌ Unknown command: $SUBCOMMAND"
        exit 1 ;;
esac
```

`dx/db`:
```bash
#!/bin/bash
source "$(dirname "$0")/_common"

usage() {
    cat <<EOF
Usage: ./dx/db <command>

Commands:
  migration <name>     Create a new migration file
  migrate              Apply all pending migrations
  migrate:rollback     Roll back the most recent migration
  status               Show migration status
EOF
}

SUBCOMMAND="${1:-}"
shift || true

if [ -z "$SUBCOMMAND" ] || [ "$SUBCOMMAND" = "-h" ] || [ "$SUBCOMMAND" = "--help" ]; then
    usage; exit 0
fi

check_docker
check_container_running

DIR="./db/migrations"

case "$SUBCOMMAND" in
    migration)
        NAME="${1:-}"
        if [ -z "$NAME" ]; then echo "❌ migration name required"; exit 1; fi
        docker compose -f "$COMPOSE_FILE" exec app goose -dir="$DIR" create "$NAME" sql
        ;;
    migrate)
        echo "🔄 Applying migrations..."
        docker compose -f "$COMPOSE_FILE" exec app go run main.go migrate
        ;;
    migrate:rollback)
        echo "⏪ Rolling back last migration..."
        docker compose -f "$COMPOSE_FILE" exec app go run main.go migrate --rollback
        ;;
    status)
        docker compose -f "$COMPOSE_FILE" exec app \
            sh -c "goose -dir=$DIR postgres \"\$ENV_DATABASE_SOURCE\" status"
        ;;
    *)
        echo "❌ Unknown command: $SUBCOMMAND"; usage; exit 1 ;;
esac
```

`dx/dbgate`:
```bash
#!/bin/bash
set -euo pipefail
source "$(dirname "$0")/_common"

SERVICE="dbgate"
URL="http://localhost:3011"

check_docker

case "${1:-}" in
    start)
        echo "🚀 Starting DBGate..."
        docker compose -f "$COMPOSE_FILE" --profile dbgate up -d "$SERVICE"
        echo "✅ DBGate starting at: ${URL}"
        ;;
    stop)
        docker compose -f "$COMPOSE_FILE" --profile dbgate stop "$SERVICE"
        docker compose -f "$COMPOSE_FILE" --profile dbgate rm -f "$SERVICE"
        echo "✅ DBGate stopped"
        ;;
    status)
        docker compose -f "$COMPOSE_FILE" --profile dbgate ps "$SERVICE" ;;
    open)
        open "$URL" 2>/dev/null || xdg-open "$URL" 2>/dev/null || echo "Open: $URL" ;;
    logs)
        docker compose -f "$COMPOSE_FILE" --profile dbgate logs -f "$SERVICE" ;;
    *)
        echo "Usage: ./dx/dbgate start|stop|status|open|logs"; exit 1 ;;
esac
```

`dx/help`:
```bash
#!/bin/bash
cat <<'EOF'
Tudu — Developer Experience (dx) toolkit

Quick start
  ./dx/build            Build the dev Docker image
  ./dx/start            Start app + postgres
  ./dx/db migrate       Apply database migrations
  ./dx/dev              Start HTTP server with live reload (http://localhost:8080)

Containers
  ./dx/start            Start app + postgres in background
  ./dx/stop [--remove]  Stop containers (optionally drop volumes)
  ./dx/status           Show container status
  ./dx/logs [service]   Tail logs
  ./dx/clean            Remove containers, volumes, and image (destructive)

Developer workflow
  ./dx/dev [--no-reload]    Run HTTP server (default: live reload via air)
  ./dx/shell                Open bash shell in app container
  ./dx/exec <cmd>           Run an arbitrary command in app container
  ./dx/test [args...]       Run go tests
  ./dx/lint                 Run go vet

Database
  ./dx/db migration <name>  Create a new migration file
  ./dx/db migrate           Apply pending migrations
  ./dx/db migrate:rollback  Roll back the last migration
  ./dx/db status            Show migration status

DBGate (web GUI for Postgres)
  ./dx/dbgate start         Start at http://localhost:3011 (opt-in)
  ./dx/dbgate open          Open in browser
  ./dx/dbgate stop          Stop the container

Code generation
  ./dx/generate openapi     Regenerate pkg/openapi/v1/openapi.gen.go from api/openapi.yml

Ports
  HTTP server   → http://localhost:8080
  Postgres      → localhost:5434
  DBGate        → http://localhost:3011
  Swagger UI    → http://localhost:8080/swagger/
EOF
```

- [ ] **Step 6: Make all dx scripts executable**

```bash
chmod +x dx/_common dx/build dx/start dx/stop dx/dev dx/shell dx/exec \
         dx/test dx/lint dx/logs dx/status dx/clean dx/generate dx/db \
         dx/dbgate dx/help
```

- [ ] **Step 7: Commit**

```bash
git add dx/ docker-compose.dev.yml .air.toml
git commit -m "chore: add dx toolkit and Docker environment"
```

---

## Task 3: Build Docker image and start environment

- [ ] **Step 1: Build the Docker image**

```bash
./dx/build
```

Expected output ends with: `✅ Image built: tudu:dev`

- [ ] **Step 2: Start the containers**

```bash
./dx/start
```

Expected: postgres and app containers running.

- [ ] **Step 3: Resolve Go dependencies inside the container**

```bash
./dx/exec go mod tidy
```

Expected: `go.sum` is created, no errors. This downloads all dependencies listed in `go.mod`.

- [ ] **Step 4: Verify Go builds**

```bash
./dx/exec go build ./...
```

Expected: build fails because `cmd/` doesn't exist yet — that's fine. No output means success once those files are written. For now, the important check is that the container is running.

- [ ] **Step 5: Commit `go.sum`**

```bash
git add go.sum
git commit -m "chore: add go.sum after mod tidy"
```

---

## Task 4: Config struct and CLI commands

**Files:**
- Create: `internal/config.go`
- Create: `internal/context.go`
- Create: `cmd/cmd.go`
- Create: `cmd/migrate.go`
- Create: `cmd/serve.go` (stub — will be completed in Task 14)
- Create: `config.example.yml`

- [ ] **Step 1: Write `internal/config.go`**

```go
package internal

import "log/slog"

type Config struct {
	Env        string           `mapstructure:"env"`
	Log        LogConfig        `mapstructure:"log"`
	HTTPServer HTTPServerConfig `mapstructure:"http_server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	JWT        JWTConfig        `mapstructure:"jwt"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func (c LogConfig) ParseSlogLevel() slog.Level {
	switch c.Level {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type HTTPServerConfig struct {
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Source string `mapstructure:"source"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}
```

- [ ] **Step 2: Write `internal/context.go`**

```go
package internal

import "context"

type ctxKey struct{}

func WithUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func UserIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(ctxKey{}).(int64)
	return id
}
```

- [ ] **Step 3: Write `cmd/cmd.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/philiplambok/tudu/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(serveCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadConfig(path string) (internal.Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("env")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return internal.Config{}, err
	}

	var cfg internal.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
```

- [ ] **Step 4: Write `cmd/migrate.go`**

```go
package cmd

import (
	"context"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var (
	migrateCmd = &cobra.Command{
		RunE:  runMigration,
		Use:   "migrate",
		Short: "run db migration files under db/migrations",
	}
	migrateRollback bool
	migrateDir      string
)

func init() {
	migrateCmd.Flags().BoolVarP(&migrateRollback, "rollback", "r", false, "rollback the latest migration")
	migrateCmd.PersistentFlags().StringVarP(&migrateDir, "dir", "d", "db/migrations", "migrations directory")
}

func runMigration(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := loadConfig(".")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := goose.OpenDBWithDriver("pgx", cfg.Database.Source)
	if err != nil {
		log.Fatalf("goose: failed to open DB: %v", err)
	}
	defer db.Close()

	goose.SetTableName("schema_migrations")

	if migrateRollback {
		return goose.RunContext(ctx, "down", db, migrateDir)
	}
	return goose.RunContext(ctx, "up", db, migrateDir)
}
```

- [ ] **Step 5: Write `cmd/serve.go` (stub — updated in Task 14)**

```go
package cmd

import (
	"log"
	"log/slog"

	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var serveCmd = &cobra.Command{
	RunE:  runServer,
	Use:   "serve",
	Short: "start the HTTP server",
}

func runServer(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig(".")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	slog.SetLogLoggerLevel(cfg.Log.ParseSlogLevel())

	db, err := gorm.Open(postgres.Open(cfg.Database.Source), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	// transport.NewServer wired in Task 14
	_ = db
	_ = cfg
	return nil
}
```

- [ ] **Step 6: Write `config.example.yml`**

```yaml
# Config values can be overridden with environment variables using the ENV_ prefix.
# Examples:
#   http_server.port  → ENV_HTTP_SERVER_PORT
#   database.source   → ENV_DATABASE_SOURCE
#   jwt.secret        → ENV_JWT_SECRET

env: local
log:
  level: INFO
http_server:
  port: "8080"
database:
  source: ""
jwt:
  secret: ""
```

- [ ] **Step 7: Create `config.yml` for local dev (not committed)**

```yaml
env: local
log:
  level: INFO
http_server:
  port: "8080"
database:
  source: "postgresql://postgres:postgres@postgres:5432/tudu?sslmode=disable"
jwt:
  secret: "local-dev-secret-change-in-production"
```

- [ ] **Step 8: Verify it compiles**

```bash
./dx/exec go build ./...
```

Expected: no errors.

- [ ] **Step 9: Commit**

```bash
git add internal/config.go internal/context.go cmd/ config.example.yml
git commit -m "feat: add config struct and CLI commands"
```

---

## Task 5: OpenAPI spec and code generation

**Files:**
- Create: `api/openapi.yml`
- Create: `oapi_codegen.yml`
- Create: `pkg/openapi/v1/` (directory)
- Generate: `pkg/openapi/v1/openapi.gen.go`

- [ ] **Step 1: Write `api/openapi.yml`**

```yaml
openapi: 3.0.0
info:
  version: 1.0.0
  title: Tudu API
  description: Clean-architecture Todoist application — multi-user, JWT-authenticated.
servers:
  - description: Local Development
    url: http://localhost:8080

tags:
  - name: Auth
  - name: Users
  - name: Tasks

paths:

  /v1/auth/register:
    post:
      tags: [Auth]
      summary: Register a new user
      operationId: register
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/RegisterRequest"
      responses:
        "201":
          description: Created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/AuthResponse"
        "409":
          $ref: "#/components/responses/Conflict"
        "422":
          $ref: "#/components/responses/UnprocessableEntity"

  /v1/auth/login:
    post:
      tags: [Auth]
      summary: Login
      operationId: login
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/LoginRequest"
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/AuthResponse"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "422":
          $ref: "#/components/responses/UnprocessableEntity"

  /v1/users/me:
    get:
      tags: [Users]
      summary: Get current user
      operationId: getMe
      security:
        - bearerAuth: []
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserResponse"
        "401":
          $ref: "#/components/responses/Unauthorized"

  /v1/tasks:
    post:
      tags: [Tasks]
      summary: Create task
      operationId: createTask
      security:
        - bearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/CreateTaskRequest"
      responses:
        "201":
          description: Created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/TaskResponse"
        "422":
          $ref: "#/components/responses/UnprocessableEntity"
    get:
      tags: [Tasks]
      summary: List tasks
      operationId: listTasks
      security:
        - bearerAuth: []
      parameters:
        - name: status
          in: query
          schema:
            type: string
            enum: [pending, completed]
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/TaskListResponse"

  /v1/tasks/{id}:
    parameters:
      - $ref: "#/components/parameters/IDParam"
    get:
      tags: [Tasks]
      summary: Get task
      operationId: getTask
      security:
        - bearerAuth: []
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/TaskResponse"
        "404":
          $ref: "#/components/responses/NotFound"
    patch:
      tags: [Tasks]
      summary: Update task
      operationId: updateTask
      security:
        - bearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UpdateTaskRequest"
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/TaskResponse"
        "404":
          $ref: "#/components/responses/NotFound"
        "422":
          $ref: "#/components/responses/UnprocessableEntity"
    delete:
      tags: [Tasks]
      summary: Delete task
      operationId: deleteTask
      security:
        - bearerAuth: []
      responses:
        "204":
          description: No Content
        "404":
          $ref: "#/components/responses/NotFound"

  /v1/tasks/{id}/complete:
    parameters:
      - $ref: "#/components/parameters/IDParam"
    post:
      tags: [Tasks]
      summary: Mark task as completed
      operationId: completeTask
      security:
        - bearerAuth: []
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/TaskResponse"
        "404":
          $ref: "#/components/responses/NotFound"

components:

  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

  parameters:
    IDParam:
      name: id
      in: path
      required: true
      schema:
        type: integer
        format: int64

  responses:
    Unauthorized:
      description: Unauthorized
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/ErrorResponse"
    NotFound:
      description: Not found
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/ErrorResponse"
    Conflict:
      description: Conflict
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/ErrorResponse"
    UnprocessableEntity:
      description: Validation error
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/ErrorResponse"

  schemas:

    ErrorResponse:
      type: object
      required: [error]
      properties:
        error:
          type: string

    RegisterRequest:
      type: object
      required: [email, password]
      properties:
        email:
          type: string
          format: email
        password:
          type: string
          minLength: 8

    LoginRequest:
      type: object
      required: [email, password]
      properties:
        email:
          type: string
          format: email
        password:
          type: string

    User:
      type: object
      required: [id, email, avatar_url, created_at, updated_at]
      properties:
        id:
          type: integer
          format: int64
        email:
          type: string
        avatar_url:
          type: string
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time

    UserResponse:
      type: object
      required: [data]
      properties:
        data:
          $ref: "#/components/schemas/User"

    AuthResponse:
      type: object
      required: [token, data]
      properties:
        token:
          type: string
        data:
          $ref: "#/components/schemas/User"

    TaskStatus:
      type: string
      enum: [pending, completed]

    Task:
      type: object
      required: [id, user_id, title, status, created_at, updated_at]
      properties:
        id:
          type: integer
          format: int64
        user_id:
          type: integer
          format: int64
        title:
          type: string
        description:
          type: string
          nullable: true
        status:
          $ref: "#/components/schemas/TaskStatus"
        due_date:
          type: string
          format: date-time
          nullable: true
        completed_at:
          type: string
          format: date-time
          nullable: true
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time

    TaskResponse:
      type: object
      required: [data]
      properties:
        data:
          $ref: "#/components/schemas/Task"

    TaskListResponse:
      type: object
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: "#/components/schemas/Task"

    CreateTaskRequest:
      type: object
      required: [title]
      properties:
        title:
          type: string
        description:
          type: string
        due_date:
          type: string
          format: date-time
          nullable: true

    UpdateTaskRequest:
      type: object
      properties:
        title:
          type: string
        description:
          type: string
        due_date:
          type: string
          format: date-time
          nullable: true
```

- [ ] **Step 2: Write `oapi_codegen.yml`**

```yaml
package: v1
generate:
  models: true
  embedded-spec: true
output: pkg/openapi/v1/openapi.gen.go
output-options:
  skip-prune: true
```

- [ ] **Step 3: Create the output directory**

```bash
mkdir -p pkg/openapi/v1
```

- [ ] **Step 4: Generate the Go types**

```bash
./dx/generate openapi
```

Expected output: `✅ Generated: pkg/openapi/v1/openapi.gen.go`

- [ ] **Step 5: Verify it compiles**

```bash
./dx/exec go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add api/openapi.yml oapi_codegen.yml pkg/openapi/v1/openapi.gen.go
git commit -m "feat: add OpenAPI spec and generate Go types"
```

---

## Task 6: Database migrations

**Files:**
- Create: `db/migrations/20260503000001_init.sql`
- Create: `db/migrations/20260503000002_create_users.sql`
- Create: `db/migrations/20260503000003_create_tasks.sql`

- [ ] **Step 1: Write `db/migrations/20260503000001_init.sql`**

```sql
-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

-- +goose Down
DROP EXTENSION IF EXISTS citext;
```

- [ ] **Step 2: Write `db/migrations/20260503000002_create_users.sql`**

```sql
-- +goose Up
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         CITEXT    UNIQUE NOT NULL,
    password_hash TEXT             NOT NULL,
    avatar_url    TEXT             NOT NULL,
    created_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS users;
```

- [ ] **Step 3: Write `db/migrations/20260503000003_create_tasks.sql`**

```sql
-- +goose Up
CREATE TABLE tasks (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id),
    title        TEXT         NOT NULL,
    description  TEXT,
    status       TEXT         NOT NULL DEFAULT 'pending',
    due_date     TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX tasks_user_id_idx ON tasks(user_id);
CREATE INDEX tasks_status_idx  ON tasks(status);

-- +goose Down
DROP TABLE IF EXISTS tasks;
```

- [ ] **Step 4: Run migrations**

```bash
./dx/db migrate
```

Expected output includes: `OK    20260503000001_init.sql`, `OK    20260503000002_create_users.sql`, `OK    20260503000003_create_tasks.sql`

- [ ] **Step 5: Verify tables exist**

```bash
./dx/exec psql "$ENV_DATABASE_SOURCE" -c "\dt"
```

Expected: `schema_migrations`, `users`, `tasks` listed.

- [ ] **Step 6: Commit**

```bash
git add db/migrations/
git commit -m "feat: add database migrations (citext, users, tasks)"
```

---

## Task 7: Avatar adapter

**Files:**
- Create: `pkg/avatar/provider.go`
- Create: `pkg/avatar/gravatar.go`
- Create: `pkg/avatar/mock.go`
- Create: `pkg/avatar/provider_test.go`

- [ ] **Step 1: Write `pkg/avatar/provider.go`**

```go
package avatar

// Provider generates an avatar URL for a given email address.
// Selected at startup: NewGravatar for production, NewMock for local/sandbox.
type Provider interface {
	GetAvatarURL(email string) string
}
```

- [ ] **Step 2: Write `pkg/avatar/gravatar.go`**

```go
package avatar

import (
	"crypto/md5"
	"fmt"
	"strings"
)

type gravatar struct{}

func NewGravatar() Provider {
	return &gravatar{}
}

func (g *gravatar) GetAvatarURL(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	hash := md5.Sum([]byte(normalized))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%x?d=identicon", hash)
}
```

- [ ] **Step 3: Write `pkg/avatar/mock.go`**

```go
package avatar

type mock struct{}

func NewMock() Provider {
	return &mock{}
}

func (m *mock) GetAvatarURL(_ string) string {
	return "https://api.dicebear.com/7.x/identicon/svg?seed=tudu"
}
```

- [ ] **Step 4: Write `pkg/avatar/provider_test.go`**

```go
package avatar_test

import (
	"strings"
	"testing"

	"github.com/philiplambok/tudu/pkg/avatar"
)

func TestGravatar_GetAvatarURL(t *testing.T) {
	p := avatar.NewGravatar()
	url := p.GetAvatarURL("test@example.com")
	if !strings.HasPrefix(url, "https://www.gravatar.com/avatar/") {
		t.Errorf("expected Gravatar URL, got %s", url)
	}
}

func TestGravatar_CaseInsensitive(t *testing.T) {
	p := avatar.NewGravatar()
	lower := p.GetAvatarURL("test@example.com")
	upper := p.GetAvatarURL("TEST@EXAMPLE.COM")
	if lower != upper {
		t.Errorf("expected same URL for same email regardless of case: %s != %s", lower, upper)
	}
}

func TestMock_GetAvatarURL(t *testing.T) {
	p := avatar.NewMock()
	url := p.GetAvatarURL("any@email.com")
	if url == "" {
		t.Error("expected non-empty URL from mock provider")
	}
	// Mock always returns the same URL regardless of email
	url2 := p.GetAvatarURL("other@email.com")
	if url != url2 {
		t.Errorf("mock should return same URL for all emails: %s != %s", url, url2)
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
./dx/test ./pkg/avatar/...
```

Expected: `PASS`

- [ ] **Step 6: Commit**

```bash
git add pkg/avatar/
git commit -m "feat: add avatar adapter (Gravatar + mock implementations)"
```

---

## Task 8: User domain layer

**Files:**
- Create: `internal/user/domain.go`
- Create: `internal/user/dto.go`
- Create: `internal/user/repository.go`

- [ ] **Step 1: Write `internal/user/dto.go`**

```go
package user

import "time"

type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }

type RegisterRequestDTO struct {
	Email    string
	Password string
}

type LoginRequestDTO struct {
	Email    string
	Password string
}

type UserResponseDTO struct {
	ID        int64
	Email     string
	AvatarURL string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AuthResponseDTO struct {
	Token string
	User  UserResponseDTO
}

type CreateUserRecordDTO struct {
	Email        string
	PasswordHash string
	AvatarURL    string
}

// AuthRecord carries the password hash alongside user fields for login verification.
type AuthRecord struct {
	ID           int64
	Email        string
	PasswordHash string
	AvatarURL    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
```

- [ ] **Step 2: Write `internal/user/domain.go`**

```go
package user

import "errors"

var (
	ErrNotFound      = errors.New("user not found")
	ErrEmailConflict = errors.New("email already registered")
	ErrInvalidCreds  = errors.New("invalid email or password")
)

func ValidateRegister(req RegisterRequestDTO) error {
	if req.Email == "" {
		return &ValidationError{"email is required"}
	}
	if len(req.Password) < 8 {
		return &ValidationError{"password must be at least 8 characters"}
	}
	return nil
}

func ValidateLogin(req LoginRequestDTO) error {
	if req.Email == "" {
		return &ValidationError{"email is required"}
	}
	if req.Password == "" {
		return &ValidationError{"password is required"}
	}
	return nil
}
```

- [ ] **Step 3: Write `internal/user/repository.go`**

```go
package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, rec CreateUserRecordDTO) (*UserResponseDTO, error)
	FindByEmailForAuth(ctx context.Context, email string) (*AuthRecord, error)
	FindByID(ctx context.Context, id int64) (*UserResponseDTO, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, rec CreateUserRecordDTO) (*UserResponseDTO, error) {
	var row UserResponseDTO
	res := r.db.WithContext(ctx).Raw(`
		INSERT INTO users (email, password_hash, avatar_url, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
		RETURNING id, email, avatar_url, created_at, updated_at`,
		rec.Email, rec.PasswordHash, rec.AvatarURL,
	).Scan(&row)
	if res.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(res.Error, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailConflict
		}
		return nil, res.Error
	}
	return &row, nil
}

func (r *repository) FindByEmailForAuth(ctx context.Context, email string) (*AuthRecord, error) {
	var row AuthRecord
	res := r.db.WithContext(ctx).Raw(`
		SELECT id, email, password_hash, avatar_url, created_at, updated_at
		FROM users WHERE email = ?`, email,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &row, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*UserResponseDTO, error) {
	var row UserResponseDTO
	res := r.db.WithContext(ctx).Raw(`
		SELECT id, email, avatar_url, created_at, updated_at
		FROM users WHERE id = ?`, id,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &row, nil
}
```

- [ ] **Step 4: Verify it compiles**

```bash
./dx/exec go build ./internal/user/...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/user/domain.go internal/user/dto.go internal/user/repository.go
git commit -m "feat: add user domain, DTOs, and repository"
```

---

## Task 9: User service (TDD)

**Files:**
- Create: `internal/user/service_test.go`
- Create: `internal/user/service.go`

- [ ] **Step 1: Write the failing tests in `internal/user/service_test.go`**

```go
package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/philiplambok/tudu/internal/user"
	"github.com/philiplambok/tudu/pkg/avatar"
)

// mockRepo satisfies user.Repository without a database.
type mockRepo struct {
	createFn             func(ctx context.Context, rec user.CreateUserRecordDTO) (*user.UserResponseDTO, error)
	findByEmailForAuthFn func(ctx context.Context, email string) (*user.AuthRecord, error)
	findByIDFn           func(ctx context.Context, id int64) (*user.UserResponseDTO, error)
}

func (m *mockRepo) Create(ctx context.Context, rec user.CreateUserRecordDTO) (*user.UserResponseDTO, error) {
	return m.createFn(ctx, rec)
}
func (m *mockRepo) FindByEmailForAuth(ctx context.Context, email string) (*user.AuthRecord, error) {
	return m.findByEmailForAuthFn(ctx, email)
}
func (m *mockRepo) FindByID(ctx context.Context, id int64) (*user.UserResponseDTO, error) {
	return m.findByIDFn(ctx, id)
}

func newTestService(repo user.Repository) user.Service {
	return user.NewService(repo, avatar.NewMock(), "test-secret")
}

func TestRegister_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, rec user.CreateUserRecordDTO) (*user.UserResponseDTO, error) {
			return &user.UserResponseDTO{
				ID:        1,
				Email:     rec.Email,
				AvatarURL: rec.AvatarURL,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}
	svc := newTestService(repo)
	resp, err := svc.Register(context.Background(), user.RegisterRequestDTO{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty JWT token")
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", resp.User.Email)
	}
}

func TestRegister_ValidationError_ShortPassword(t *testing.T) {
	svc := newTestService(&mockRepo{})
	_, err := svc.Register(context.Background(), user.RegisterRequestDTO{
		Email:    "alice@example.com",
		Password: "short",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *user.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected *user.ValidationError, got %T", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ user.CreateUserRecordDTO) (*user.UserResponseDTO, error) {
			return nil, user.ErrEmailConflict
		},
	}
	svc := newTestService(repo)
	_, err := svc.Register(context.Background(), user.RegisterRequestDTO{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if !errors.Is(err, user.ErrEmailConflict) {
		t.Errorf("expected ErrEmailConflict, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	// Pre-hash the password as bcrypt would store it.
	// We call Register first to get a valid hash, then verify Login works.
	repo := &mockRepo{
		createFn: func(_ context.Context, rec user.CreateUserRecordDTO) (*user.UserResponseDTO, error) {
			return &user.UserResponseDTO{ID: 1, Email: rec.Email, AvatarURL: rec.AvatarURL}, nil
		},
		findByEmailForAuthFn: func(_ context.Context, email string) (*user.AuthRecord, error) {
			// Return a record with a known bcrypt hash for "password123".
			// Generated via: bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
			return &user.AuthRecord{
				ID:           1,
				Email:        email,
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
				AvatarURL:    "https://example.com/avatar.png",
			}, nil
		},
	}
	svc := newTestService(repo)
	resp, err := svc.Login(context.Background(), user.LoginRequestDTO{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty JWT token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &mockRepo{
		findByEmailForAuthFn: func(_ context.Context, email string) (*user.AuthRecord, error) {
			return &user.AuthRecord{
				ID:           1,
				Email:        email,
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			}, nil
		},
	}
	svc := newTestService(repo)
	_, err := svc.Login(context.Background(), user.LoginRequestDTO{
		Email:    "alice@example.com",
		Password: "wrongpassword",
	})
	if !errors.Is(err, user.ErrInvalidCreds) {
		t.Errorf("expected ErrInvalidCreds, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockRepo{
		findByEmailForAuthFn: func(_ context.Context, _ string) (*user.AuthRecord, error) {
			return nil, user.ErrNotFound
		},
	}
	svc := newTestService(repo)
	_, err := svc.Login(context.Background(), user.LoginRequestDTO{
		Email:    "nobody@example.com",
		Password: "password123",
	})
	if !errors.Is(err, user.ErrInvalidCreds) {
		t.Errorf("expected ErrInvalidCreds (not ErrNotFound) to avoid leaking user existence, got %v", err)
	}
}
```

- [ ] **Step 2: Run the tests to confirm they fail**

```bash
./dx/test ./internal/user/... -run TestRegister
```

Expected: compile error — `user.NewService`, `user.Service` not defined yet.

- [ ] **Step 3: Write `internal/user/service.go`**

```go
package user

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/philiplambok/tudu/pkg/avatar"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, req RegisterRequestDTO) (*AuthResponseDTO, error)
	Login(ctx context.Context, req LoginRequestDTO) (*AuthResponseDTO, error)
	Me(ctx context.Context, userID int64) (*UserResponseDTO, error)
}

type service struct {
	repo           Repository
	avatarProvider avatar.Provider
	jwtSecret      string
}

func NewService(repo Repository, avatarProvider avatar.Provider, jwtSecret string) Service {
	return &service{repo: repo, avatarProvider: avatarProvider, jwtSecret: jwtSecret}
}

func (s *service) Register(ctx context.Context, req RegisterRequestDTO) (*AuthResponseDTO, error) {
	if err := ValidateRegister(req); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	avatarURL := s.avatarProvider.GetAvatarURL(req.Email)

	u, err := s.repo.Create(ctx, CreateUserRecordDTO{
		Email:        req.Email,
		PasswordHash: string(hash),
		AvatarURL:    avatarURL,
	})
	if err != nil {
		return nil, err
	}

	token, err := s.generateToken(u.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponseDTO{Token: token, User: *u}, nil
}

func (s *service) Login(ctx context.Context, req LoginRequestDTO) (*AuthResponseDTO, error) {
	if err := ValidateLogin(req); err != nil {
		return nil, err
	}

	rec, err := s.repo.FindByEmailForAuth(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidCreds
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(rec.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCreds
	}

	token, err := s.generateToken(rec.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponseDTO{
		Token: token,
		User: UserResponseDTO{
			ID:        rec.ID,
			Email:     rec.Email,
			AvatarURL: rec.AvatarURL,
			CreatedAt: rec.CreatedAt,
			UpdatedAt: rec.UpdatedAt,
		},
	}, nil
}

func (s *service) Me(ctx context.Context, userID int64) (*UserResponseDTO, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *service) generateToken(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
```

- [ ] **Step 4: Add the missing `fmt` import to `service.go`**

The `generateToken` function uses `fmt.Sprintf`. Make sure the import block is:

```go
import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/philiplambok/tudu/pkg/avatar"
	"golang.org/x/crypto/bcrypt"
)
```

- [ ] **Step 5: Run the tests to confirm they pass**

```bash
./dx/test ./internal/user/...
```

Expected: `ok  	github.com/philiplambok/tudu/internal/user`

- [ ] **Step 6: Commit**

```bash
git add internal/user/service.go internal/user/service_test.go
git commit -m "feat: add user service with register/login/me and unit tests"
```

---

## Task 10: User handler and endpoint

**Files:**
- Create: `internal/user/handler.go`
- Create: `internal/user/endpoint.go`

- [ ] **Step 1: Write `internal/user/handler.go`**

```go
package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"github.com/philiplambok/tudu/internal"
	v1 "github.com/philiplambok/tudu/pkg/openapi/v1"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body v1.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Register(r.Context(), RegisterRequestDTO{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeError(w, r, http.StatusUnprocessableEntity, ve.Error())
			return
		}
		if errors.Is(err, ErrEmailConflict) {
			writeError(w, r, http.StatusConflict, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to register")
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, v1.AuthResponse{Token: resp.Token, Data: toV1User(&resp.User)})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body v1.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Login(r.Context(), LoginRequestDTO{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeError(w, r, http.StatusUnprocessableEntity, ve.Error())
			return
		}
		if errors.Is(err, ErrInvalidCreds) {
			writeError(w, r, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to login")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.AuthResponse{Token: resp.Token, Data: toV1User(&resp.User)})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	u, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to get user")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.UserResponse{Data: toV1User(u)})
}

func toV1User(u *UserResponseDTO) v1.User {
	return v1.User{
		Id:        u.ID,
		Email:     u.Email,
		AvatarUrl: u.AvatarURL,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	render.Status(r, status)
	render.JSON(w, r, v1.ErrorResponse{Error: msg})
}
```

- [ ] **Step 2: Write `internal/user/endpoint.go`**

```go
package user

import (
	"github.com/go-chi/chi/v5"
	"github.com/philiplambok/tudu/pkg/avatar"
	"gorm.io/gorm"
)

type Endpoint struct {
	handler *Handler
}

func NewEndpoint(db *gorm.DB, avatarProvider avatar.Provider, jwtSecret string) *Endpoint {
	repo := NewRepository(db)
	svc := NewService(repo, avatarProvider, jwtSecret)
	return &Endpoint{handler: NewHandler(svc)}
}

// AuthRoutes returns public auth routes (no JWT required).
func (e *Endpoint) AuthRoutes() *chi.Mux {
	r := chi.NewMux()
	r.Post("/register", e.handler.Register)
	r.Post("/login", e.handler.Login)
	return r
}

// MeRoutes returns the protected /me route (JWT required).
func (e *Endpoint) MeRoutes() *chi.Mux {
	r := chi.NewMux()
	r.Get("/me", e.handler.Me)
	return r
}
```

- [ ] **Step 3: Verify it compiles**

```bash
./dx/exec go build ./internal/user/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/user/handler.go internal/user/endpoint.go
git commit -m "feat: add user HTTP handler and endpoint wiring"
```

---

## Task 11: Task domain layer

**Files:**
- Create: `internal/task/domain.go`
- Create: `internal/task/dto.go`
- Create: `internal/task/repository.go`

- [ ] **Step 1: Write `internal/task/dto.go`**

```go
package task

import "time"

type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }

type CreateRequestDTO struct {
	Title       string
	Description string
	DueDate     *time.Time
}

type UpdateRequestDTO struct {
	Title       *string
	Description *string
	DueDate     *time.Time
}

type TaskResponseDTO struct {
	ID          int64
	UserID      int64
	Title       string
	Description string
	Status      string
	DueDate     *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

- [ ] **Step 2: Write `internal/task/domain.go`**

```go
package task

import "errors"

const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
)

var ErrNotFound = errors.New("task not found")

func ValidateCreate(req CreateRequestDTO) error {
	if req.Title == "" {
		return &ValidationError{"title is required"}
	}
	return nil
}

func ValidateUpdate(req UpdateRequestDTO) error {
	if req.Title == nil && req.Description == nil && req.DueDate == nil {
		return &ValidationError{"at least one field is required"}
	}
	if req.Title != nil && *req.Title == "" {
		return &ValidationError{"title cannot be empty"}
	}
	return nil
}
```

- [ ] **Step 3: Write `internal/task/repository.go`**

```go
package task

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, userID int64, req CreateRequestDTO) (*TaskResponseDTO, error)
	List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error)
	Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error)
	Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Delete(ctx context.Context, userID int64, id int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, userID int64, req CreateRequestDTO) (*TaskResponseDTO, error) {
	var row TaskResponseDTO
	res := r.db.WithContext(ctx).Raw(`
		INSERT INTO tasks (user_id, title, description, status, due_date, created_at, updated_at)
		VALUES (?, ?, ?, 'pending', ?, NOW(), NOW())
		RETURNING id, user_id, title, description, status, due_date, completed_at, created_at, updated_at`,
		userID, req.Title, req.Description, req.DueDate,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	return &row, nil
}

func (r *repository) List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error) {
	q := r.db.WithContext(ctx).
		Select("id, user_id, title, description, status, due_date, completed_at, created_at, updated_at").
		Table("tasks").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if status != "" {
		q = q.Where("status = ?", status)
	}

	var rows []TaskResponseDTO
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []TaskResponseDTO{}
	}
	return rows, nil
}

func (r *repository) Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	var row TaskResponseDTO
	res := r.db.WithContext(ctx).Raw(`
		SELECT id, user_id, title, description, status, due_date, completed_at, created_at, updated_at
		FROM tasks WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &row, nil
}

func (r *repository) Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error) {
	updates := map[string]any{"updated_at": gorm.Expr("NOW()")}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DueDate != nil {
		updates["due_date"] = *req.DueDate
	}

	res := r.db.WithContext(ctx).
		Table("tasks").
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, userID, id)
}

func (r *repository) Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	res := r.db.WithContext(ctx).
		Table("tasks").
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]any{
			"status":       StatusCompleted,
			"completed_at": gorm.Expr("NOW()"),
			"updated_at":   gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, userID, id)
}

func (r *repository) Delete(ctx context.Context, userID int64, id int64) error {
	res := r.db.WithContext(ctx).
		Table("tasks").
		Where("id = ? AND user_id = ?", id, userID).
		Delete(nil)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
```

- [ ] **Step 4: Verify it compiles**

```bash
./dx/exec go build ./internal/task/...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/task/domain.go internal/task/dto.go internal/task/repository.go
git commit -m "feat: add task domain, DTOs, and repository"
```

---

## Task 12: Task service (TDD)

**Files:**
- Create: `internal/task/service_test.go`
- Create: `internal/task/service.go`

- [ ] **Step 1: Write the failing tests in `internal/task/service_test.go`**

```go
package task_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/philiplambok/tudu/internal/task"
)

type mockTaskRepo struct {
	createFn   func(ctx context.Context, userID int64, req task.CreateRequestDTO) (*task.TaskResponseDTO, error)
	listFn     func(ctx context.Context, userID int64, status string) ([]task.TaskResponseDTO, error)
	getFn      func(ctx context.Context, userID int64, id int64) (*task.TaskResponseDTO, error)
	updateFn   func(ctx context.Context, userID int64, id int64, req task.UpdateRequestDTO) (*task.TaskResponseDTO, error)
	completeFn func(ctx context.Context, userID int64, id int64) (*task.TaskResponseDTO, error)
	deleteFn   func(ctx context.Context, userID int64, id int64) error
}

func (m *mockTaskRepo) Create(ctx context.Context, userID int64, req task.CreateRequestDTO) (*task.TaskResponseDTO, error) {
	return m.createFn(ctx, userID, req)
}
func (m *mockTaskRepo) List(ctx context.Context, userID int64, status string) ([]task.TaskResponseDTO, error) {
	return m.listFn(ctx, userID, status)
}
func (m *mockTaskRepo) Get(ctx context.Context, userID int64, id int64) (*task.TaskResponseDTO, error) {
	return m.getFn(ctx, userID, id)
}
func (m *mockTaskRepo) Update(ctx context.Context, userID int64, id int64, req task.UpdateRequestDTO) (*task.TaskResponseDTO, error) {
	return m.updateFn(ctx, userID, id, req)
}
func (m *mockTaskRepo) Complete(ctx context.Context, userID int64, id int64) (*task.TaskResponseDTO, error) {
	return m.completeFn(ctx, userID, id)
}
func (m *mockTaskRepo) Delete(ctx context.Context, userID int64, id int64) error {
	return m.deleteFn(ctx, userID, id)
}

func TestCreate_Success(t *testing.T) {
	repo := &mockTaskRepo{
		createFn: func(_ context.Context, userID int64, req task.CreateRequestDTO) (*task.TaskResponseDTO, error) {
			return &task.TaskResponseDTO{
				ID:     1,
				UserID: userID,
				Title:  req.Title,
				Status: task.StatusPending,
			}, nil
		},
	}
	svc := task.NewService(repo)
	got, err := svc.Create(context.Background(), 1, task.CreateRequestDTO{Title: "Buy groceries"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Title != "Buy groceries" {
		t.Errorf("expected title 'Buy groceries', got %s", got.Title)
	}
	if got.Status != task.StatusPending {
		t.Errorf("expected status pending, got %s", got.Status)
	}
}

func TestCreate_EmptyTitle(t *testing.T) {
	svc := task.NewService(&mockTaskRepo{})
	_, err := svc.Create(context.Background(), 1, task.CreateRequestDTO{Title: ""})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *task.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected *task.ValidationError, got %T", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	repo := &mockTaskRepo{
		getFn: func(_ context.Context, _ int64, _ int64) (*task.TaskResponseDTO, error) {
			return nil, task.ErrNotFound
		},
	}
	svc := task.NewService(repo)
	_, err := svc.Get(context.Background(), 1, 999)
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestComplete_Success(t *testing.T) {
	now := time.Now()
	repo := &mockTaskRepo{
		completeFn: func(_ context.Context, userID int64, id int64) (*task.TaskResponseDTO, error) {
			return &task.TaskResponseDTO{
				ID:          id,
				UserID:      userID,
				Title:       "Exercise",
				Status:      task.StatusCompleted,
				CompletedAt: &now,
			}, nil
		},
	}
	svc := task.NewService(repo)
	got, err := svc.Complete(context.Background(), 1, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Status != task.StatusCompleted {
		t.Errorf("expected status completed, got %s", got.Status)
	}
	if got.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestUpdate_NoFields(t *testing.T) {
	svc := task.NewService(&mockTaskRepo{})
	_, err := svc.Update(context.Background(), 1, 1, task.UpdateRequestDTO{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *task.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected *task.ValidationError, got %T", err)
	}
}
```

- [ ] **Step 2: Run the tests to confirm they fail**

```bash
./dx/test ./internal/task/... -run TestCreate
```

Expected: compile error — `task.NewService` not defined yet.

- [ ] **Step 3: Write `internal/task/service.go`**

```go
package task

import "context"

type Service interface {
	Create(ctx context.Context, userID int64, req CreateRequestDTO) (*TaskResponseDTO, error)
	List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error)
	Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error)
	Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Delete(ctx context.Context, userID int64, id int64) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, userID int64, req CreateRequestDTO) (*TaskResponseDTO, error) {
	if err := ValidateCreate(req); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, userID, req)
}

func (s *service) List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error) {
	return s.repo.List(ctx, userID, status)
}

func (s *service) Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *service) Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error) {
	if err := ValidateUpdate(req); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, userID, id, req)
}

func (s *service) Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	return s.repo.Complete(ctx, userID, id)
}

func (s *service) Delete(ctx context.Context, userID int64, id int64) error {
	return s.repo.Delete(ctx, userID, id)
}
```

- [ ] **Step 4: Run the tests to confirm they pass**

```bash
./dx/test ./internal/task/...
```

Expected: `ok  	github.com/philiplambok/tudu/internal/task`

- [ ] **Step 5: Commit**

```bash
git add internal/task/service.go internal/task/service_test.go
git commit -m "feat: add task service and unit tests"
```

---

## Task 13: Task handler and endpoint

**Files:**
- Create: `internal/task/handler.go`
- Create: `internal/task/endpoint.go`

- [ ] **Step 1: Write `internal/task/handler.go`**

```go
package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/philiplambok/tudu/internal"
	v1 "github.com/philiplambok/tudu/pkg/openapi/v1"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	var body v1.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	req := CreateRequestDTO{Title: body.Title}
	if body.Description != nil {
		req.Description = *body.Description
	}
	if body.DueDate != nil {
		req.DueDate = body.DueDate
	}

	task, err := h.svc.Create(r.Context(), userID, req)
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeError(w, r, http.StatusUnprocessableEntity, ve.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to create task")
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, v1.TaskResponse{Data: toV1Task(task)})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())
	status := r.URL.Query().Get("status")

	tasks, err := h.svc.List(r.Context(), userID, status)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	data := make([]v1.Task, len(tasks))
	for i := range tasks {
		data[i] = toV1Task(&tasks[i])
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskListResponse{Data: data})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.svc.Get(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to get task")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskResponse{Data: toV1Task(task)})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	var body v1.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	req := UpdateRequestDTO{
		Title:       body.Title,
		Description: body.Description,
		DueDate:     body.DueDate,
	}

	task, err := h.svc.Update(r.Context(), userID, id, req)
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeError(w, r, http.StatusUnprocessableEntity, ve.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to update task")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskResponse{Data: toV1Task(task)})
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.svc.Complete(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to complete task")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskResponse{Data: toV1Task(task)})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.svc.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toV1Task(t *TaskResponseDTO) v1.Task {
	task := v1.Task{
		Id:          t.ID,
		UserId:      t.UserID,
		Title:       t.Title,
		Status:      v1.TaskStatus(t.Status),
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
		DueDate:     t.DueDate,
	}
	if t.Description != "" {
		task.Description = &t.Description
	}
	return task
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	render.Status(r, status)
	render.JSON(w, r, v1.ErrorResponse{Error: msg})
}
```

- [ ] **Step 2: Write `internal/task/endpoint.go`**

```go
package task

import (
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Endpoint struct {
	handler *Handler
}

func NewEndpoint(db *gorm.DB) *Endpoint {
	repo := NewRepository(db)
	svc := NewService(repo)
	return &Endpoint{handler: NewHandler(svc)}
}

func (e *Endpoint) Routes() *chi.Mux {
	r := chi.NewMux()
	r.Post("/", e.handler.Create)
	r.Get("/", e.handler.List)
	r.Get("/{id}", e.handler.Get)
	r.Patch("/{id}", e.handler.Update)
	r.Post("/{id}/complete", e.handler.Complete)
	r.Delete("/{id}", e.handler.Delete)
	return r
}
```

- [ ] **Step 3: Verify it compiles**

```bash
./dx/exec go build ./internal/task/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/task/handler.go internal/task/endpoint.go
git commit -m "feat: add task HTTP handler and endpoint wiring"
```

---

## Task 14: Transport server, swagger, and final wiring

**Files:**
- Create: `internal/swagger/swagger.go`
- Create: `internal/transport/rest.go`
- Modify: `cmd/serve.go` (replace stub with full wiring)

- [ ] **Step 1: Write `internal/swagger/swagger.go`**

```go
package swagger

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	v1 "github.com/philiplambok/tudu/pkg/openapi/v1"
	httpSwagger "github.com/swaggo/http-swagger"
)

func Register(r chi.Router) {
	r.Get("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		s, err := v1.GetSpec()
		if err != nil {
			http.Error(w, "failed to load spec", http.StatusInternalServerError)
			return
		}
		b, err := s.MarshalJSON()
		if err != nil {
			http.Error(w, "failed to marshal spec", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})

	r.Handle("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger.json"),
	))
}
```

- [ ] **Step 2: Write `internal/transport/rest.go`**

```go
package transport

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/philiplambok/tudu/internal"
	"github.com/philiplambok/tudu/internal/swagger"
	"github.com/philiplambok/tudu/internal/task"
	"github.com/philiplambok/tudu/internal/user"
	"github.com/philiplambok/tudu/pkg/avatar"
	"gorm.io/gorm"
)

type Server struct {
	mux *chi.Mux
	cfg internal.Config
}

func NewServer(cfg internal.Config, db *gorm.DB, avatarProvider avatar.Provider) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	swagger.Register(r)

	userEndpoint := user.NewEndpoint(db, avatarProvider, cfg.JWT.Secret)
	r.Mount("/v1/auth", userEndpoint.AuthRoutes())

	r.Group(func(r chi.Router) {
		r.Use(jwtMiddleware(cfg.JWT.Secret))
		r.Mount("/v1/users", userEndpoint.MeRoutes())
		r.Mount("/v1/tasks", task.NewEndpoint(db).Routes())
	})

	return &Server{mux: r, cfg: cfg}
}

func (s *Server) Start() error {
	addr := ":" + s.cfg.HTTPServer.Port
	slog.Info("tudu listening", "addr", addr)
	return http.ListenAndServe(addr, s.mux)
}

func jwtMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			sub, err := claims.GetSubject()
			if err != nil {
				http.Error(w, `{"error":"invalid token subject"}`, http.StatusUnauthorized)
				return
			}

			var userID int64
			if _, err := fmt.Sscanf(sub, "%d", &userID); err != nil {
				http.Error(w, `{"error":"invalid token subject"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(internal.WithUserID(r.Context(), userID)))
		})
	}
}
```

- [ ] **Step 3: Add missing `fmt` import to `rest.go`**

The `jwtMiddleware` function uses `fmt.Sscanf`. Add `"fmt"` to the import block:

```go
import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	// ... rest of imports
)
```

- [ ] **Step 4: Replace the stub `cmd/serve.go` with the full wiring**

```go
package cmd

import (
	"log"
	"log/slog"

	"github.com/philiplambok/tudu/internal/transport"
	"github.com/philiplambok/tudu/pkg/avatar"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var serveCmd = &cobra.Command{
	RunE:  runServer,
	Use:   "serve",
	Short: "start the HTTP server",
}

func runServer(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig(".")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	slog.SetLogLoggerLevel(cfg.Log.ParseSlogLevel())

	db, err := gorm.Open(postgres.Open(cfg.Database.Source), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	var avatarProvider avatar.Provider
	if cfg.Env == "production" {
		avatarProvider = avatar.NewGravatar()
	} else {
		avatarProvider = avatar.NewMock()
	}

	return transport.NewServer(cfg, db, avatarProvider).Start()
}
```

- [ ] **Step 5: Verify the full build**

```bash
./dx/exec go build ./...
```

Expected: no errors.

- [ ] **Step 6: Run all tests**

```bash
./dx/test ./...
```

Expected: all packages pass.

- [ ] **Step 7: Commit**

```bash
git add internal/swagger/ internal/transport/ cmd/serve.go
git commit -m "feat: add transport server with JWT middleware and full route wiring"
```

---

## Task 15: End-to-end smoke test

- [ ] **Step 1: Start the HTTP server**

```bash
./dx/dev
```

Expected: `tudu listening  addr=:8080`

In a separate terminal, run the following curl commands.

- [ ] **Step 2: Register a user**

```bash
curl -s -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"password123"}' | jq .
```

Expected:
```json
{
  "token": "<jwt>",
  "data": {
    "id": 1,
    "email": "alice@example.com",
    "avatar_url": "https://api.dicebear.com/7.x/identicon/svg?seed=tudu",
    ...
  }
}
```

- [ ] **Step 3: Save the token and call /v1/users/me**

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"password123"}' | jq -r .token)

curl -s http://localhost:8080/v1/users/me \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Expected: user object with `id`, `email`, `avatar_url`.

- [ ] **Step 4: Create a task**

```bash
curl -s -X POST http://localhost:8080/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy groceries","description":"Milk, eggs, bread"}' | jq .
```

Expected: task object with `status: "pending"`.

- [ ] **Step 5: Complete the task**

```bash
curl -s -X POST http://localhost:8080/v1/tasks/1/complete \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Expected: task object with `status: "completed"` and `completed_at` set.

- [ ] **Step 6: Verify task list filters work**

```bash
curl -s "http://localhost:8080/v1/tasks?status=completed" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Expected: array with the completed task.

- [ ] **Step 7: Check Swagger UI**

Open `http://localhost:8080/swagger/` in a browser.

Expected: Swagger UI loads with all tudu endpoints listed.

- [ ] **Step 8: Final commit**

```bash
git add .
git commit -m "feat: complete tudu — clean architecture Go template"
```

---

## Self-Review Notes

**Spec coverage check:**
- ✅ Multi-user with JWT auth → Tasks 9, 14
- ✅ Register / Login → Task 9, 10
- ✅ GET /v1/users/me → Task 10
- ✅ Add task → Tasks 11, 12, 13
- ✅ Update task → Tasks 11, 12, 13
- ✅ Mark task as completed → Tasks 11, 12, 13
- ✅ Avatar adapter (production/sandbox) → Tasks 7, 14
- ✅ citext for email → Task 6
- ✅ dx toolkit → Task 2
- ✅ OpenAPI spec-first → Task 5
- ✅ Smoke test → Task 15

**Type consistency:**
- `internal.UserIDFromContext` used in `user/handler.go` and `task/handler.go` — defined in Task 4
- `avatar.Provider` interface defined in Task 7, consumed in Task 9 (user service) and Task 14 (serve.go)
- `v1.Task`, `v1.User`, `v1.AuthResponse` generated in Task 5, used in Task 10 and 13
- `task.StatusPending` / `task.StatusCompleted` defined in Task 11, used in Task 12 (tests) and Task 11 (repository)
- `user.AuthRecord` defined in Task 8 (dto.go), used in Task 9 (service) — consistent
