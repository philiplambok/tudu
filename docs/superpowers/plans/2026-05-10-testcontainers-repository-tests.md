# Testcontainers Repository Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add integration tests for both repository layers (`internal/task/` and `internal/user/`) backed by real PostgreSQL via testcontainers, runnable inside the Docker dev environment with `./dx/test`.

**Architecture:** Spin up an isolated PostgreSQL container per test suite (`BeforeSuite`), run goose migrations to build the schema, load base fixture data (users), and snapshot. Each test restores to the clean snapshot state. Test files live alongside `repository.go` in each domain package as `package task_test` / `package user_test`.

**Tech Stack:** testcontainers-go (postgres module), Ginkgo v2, Gomega, go-testfixtures v3, goose (already in use)

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `docker-compose.dev.yml` | Modify | Mount Docker socket so testcontainers can spawn containers from within the app container |
| `db/migrations.go` | Create | Embed `db/migrations/` directory so testinfra can run goose without needing a filesystem path |
| `internal/common/testinfra/testinfra.go` | Create | `SetupTestDB` (container + migrations + fixtures + snapshot) and `RestoreDB` (restore + reconnect) |
| `internal/common/testinfra/fixtures/users.yml` | Create | Base user fixture, pre-seeded into snapshot; tasks FK to users |
| `internal/user/suite_test.go` | Create | Ginkgo suite bootstrap for user repository tests |
| `internal/user/repository_test.go` | Create | Tests: Create (success + duplicate email), FindByEmailForAuth (found/not found), FindByID (found/not found) |
| `internal/task/suite_test.go` | Create | Ginkgo suite bootstrap for task repository tests |
| `internal/task/repository_test.go` | Create | Tests: Create, List (filter/pagination), Get, Update, Complete, Delete, ListActivities |

---

## Task 1: Add dependencies and Docker socket access

**Files:**
- Modify: `docker-compose.dev.yml`
- Run: `go get` commands (from host)

- [ ] **Step 1: Add testcontainers and testing framework dependencies**

Run from the project root (on the host, not inside Docker):
```bash
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
go get github.com/onsi/ginkgo/v2@latest
go get github.com/onsi/gomega@latest
go get github.com/go-testfixtures/testfixtures/v3@latest
go mod tidy
```

- [ ] **Step 2: Mount the Docker socket in docker-compose.dev.yml**

The app container needs access to the host Docker daemon so testcontainers can spawn PostgreSQL containers. Add to the `app` service's `volumes`:

In `docker-compose.dev.yml`, change the `app.volumes` section from:
```yaml
    volumes:
      - .:/app
      - go_mod_cache:/go/pkg/mod
      - go_build_cache:/root/.cache/go-build
```
to:
```yaml
    volumes:
      - .:/app
      - go_mod_cache:/go/pkg/mod
      - go_build_cache:/root/.cache/go-build
      - /var/run/docker.sock:/var/run/docker.sock
```

- [ ] **Step 3: Rebuild the dev container**

```bash
./dx/build
./dx/stop --remove
./dx/start
```

- [ ] **Step 4: Verify Docker is accessible from inside the container**

```bash
./dx/shell
# Inside container:
docker ps
# Expected: list of running containers (should show tudu-app and tudu-postgres at minimum)
```

---

## Task 2: Embed migrations and create testinfra package

**Files:**
- Create: `db/migrations.go`
- Create: `internal/common/testinfra/testinfra.go`
- Create: `internal/common/testinfra/fixtures/users.yml`

- [ ] **Step 1: Create the migrations embed file**

Create `db/migrations.go`:
```go
package db

import "embed"

//go:embed migrations
var Migrations embed.FS
```

This allows testinfra to run goose migrations without needing an absolute filesystem path. The embed path `migrations` is relative to the `db/` directory, so it captures all SQL files in `db/migrations/`.

- [ ] **Step 2: Create the users fixture YAML**

Create `internal/common/testinfra/fixtures/users.yml`:
```yaml
- id: 1
  email: testuser@example.com
  password_hash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
  created_at: "2026-01-01 00:00:00+00"
  updated_at: "2026-01-01 00:00:00+00"
```

User ID 1 is the FK parent used in all task tests. The password hash is a real bcrypt hash of `"password123"` (only used in user auth tests).

- [ ] **Step 3: Create the testinfra package**

Create `internal/common/testinfra/testinfra.go`:
```go
package testinfra

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"time"

	"context"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/go-testfixtures/testfixtures/v3"

	tududb "github.com/philiplambok/tudu/db"
)

//go:embed fixtures
var fixturesFS embed.FS

// SetupTestDB starts a PostgreSQL container, runs all goose migrations,
// seeds base fixture data (users), and takes a snapshot. Returns a ready GORM DB.
// Call Terminate on the container in AfterSuite.
func SetupTestDB(ctx context.Context) (*gorm.DB, *postgrescontainer.PostgresContainer, error) {
	container, err := postgrescontainer.Run(ctx, "postgres:16-alpine",
		postgrescontainer.WithDatabase("tudu_test"),
		postgrescontainer.WithUsername("postgres"),
		postgrescontainer.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("connection string: %w", err)
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open sql db: %w", err)
	}

	goose.SetBaseFS(tududb.Migrations)
	goose.SetTableName("schema_migrations")
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	subFS, err := fs.Sub(fixturesFS, "fixtures")
	if err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("sub fixtures fs: %w", err)
	}
	fixtures, err := testfixtures.New(
		testfixtures.Database(sqlDB),
		testfixtures.Dialect("postgresql"),
		testfixtures.FS(subFS),
	)
	if err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("setup fixtures: %w", err)
	}
	if err := fixtures.Load(); err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("load fixtures: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return nil, nil, fmt.Errorf("close migration db: %w", err)
	}

	if err := container.Snapshot(ctx); err != nil {
		return nil, nil, fmt.Errorf("snapshot: %w", err)
	}

	gormDB, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("gorm open: %w", err)
	}

	return gormDB, container, nil
}

// RestoreDB restores the database to the clean snapshot state and reconnects GORM.
// Call this in BeforeEach for each test to get a clean starting state.
func RestoreDB(ctx context.Context, container *postgrescontainer.PostgresContainer, db **gorm.DB) error {
	if err := container.Restore(ctx); err != nil {
		return fmt.Errorf("restore snapshot: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("connection string: %w", err)
	}

	newDB, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("gorm reconnect: %w", err)
	}
	*db = newDB
	return nil
}
```

**Import alias note:** The `postgres` name is used by both testcontainers and gorm driver, so use these aliases:
- `postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"`
- `gormpostgres "gorm.io/driver/postgres"`

Update the import block above accordingly.

- [ ] **Step 4: Verify the package compiles**

```bash
./dx/shell
# Inside container:
go build ./internal/common/testinfra/...
# Expected: no output (successful compile)
```

---

## Task 3: User repository tests

**Files:**
- Create: `internal/user/suite_test.go`
- Create: `internal/user/repository_test.go`

- [ ] **Step 1: Create the Ginkgo suite file**

Create `internal/user/suite_test.go`:
```go
package user_test

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/gorm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/philiplambok/tudu/internal/common/testinfra"
)

var (
	db        *gorm.DB
	container *postgres.PostgresContainer
	ctx       context.Context
)

func TestUserRepository(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User Repository Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	var err error
	db, container, err = testinfra.SetupTestDB(ctx)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	if container != nil {
		err := container.Terminate(ctx)
		Expect(err).ToNot(HaveOccurred())
	}
})
```

- [ ] **Step 2: Write the failing tests**

Create `internal/user/repository_test.go`:
```go
package user_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/philiplambok/tudu/internal/common/testinfra"
	"github.com/philiplambok/tudu/internal/user"
)

var _ = Describe("Repository", func() {
	var repo user.Repository

	BeforeEach(func() {
		err := testinfra.RestoreDB(ctx, container, &db)
		Expect(err).ToNot(HaveOccurred())
		repo = user.NewRepository(db)
	})

	Describe("Create", func() {
		It("creates a user and returns the DTO without exposing the password hash", func() {
			result, err := repo.Create(ctx, user.CreateUserRecordDTO{
				Email:        "new@example.com",
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).ToNot(BeZero())
			Expect(result.Email).To(Equal("new@example.com"))
			Expect(result.CreatedAt).ToNot(BeZero())
		})

		It("returns ErrEmailConflict when the email already exists", func() {
			_, err := repo.Create(ctx, user.CreateUserRecordDTO{
				Email:        "testuser@example.com", // seeded via fixture
				PasswordHash: "hash",
			})
			Expect(err).To(Equal(user.ErrEmailConflict))
		})
	})

	Describe("FindByEmailForAuth", func() {
		It("returns the auth record for an existing email", func() {
			result, err := repo.FindByEmailForAuth(ctx, "testuser@example.com")
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(int64(1)))
			Expect(result.Email).To(Equal("testuser@example.com"))
			Expect(result.PasswordHash).ToNot(BeEmpty())
		})

		It("returns ErrNotFound for an unknown email", func() {
			_, err := repo.FindByEmailForAuth(ctx, "nobody@example.com")
			Expect(err).To(Equal(user.ErrNotFound))
		})
	})

	Describe("FindByID", func() {
		It("returns the user DTO for an existing ID", func() {
			result, err := repo.FindByID(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(int64(1)))
			Expect(result.Email).To(Equal("testuser@example.com"))
		})

		It("returns ErrNotFound for an unknown ID", func() {
			_, err := repo.FindByID(ctx, 9999)
			Expect(err).To(Equal(user.ErrNotFound))
		})
	})
})
```

- [ ] **Step 3: Run the tests and verify they pass**

```bash
./dx/test ./internal/user/... -v
# Expected: All 6 specs pass under "User Repository Suite"
# Container boot + migrations takes ~10s on first run
```

- [ ] **Step 4: Commit**

```bash
git add internal/user/suite_test.go internal/user/repository_test.go
git add internal/common/testinfra/ db/migrations.go docker-compose.dev.yml
git commit -m "test: add testcontainers integration tests for user repository"
```

---

## Task 4: Task repository tests

**Files:**
- Create: `internal/task/suite_test.go`
- Create: `internal/task/repository_test.go`

- [ ] **Step 1: Create the Ginkgo suite file**

Create `internal/task/suite_test.go`:
```go
package task_test

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/gorm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/philiplambok/tudu/internal/common/testinfra"
)

var (
	db        *gorm.DB
	container *postgres.PostgresContainer
	ctx       context.Context
)

func TestTaskRepository(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Task Repository Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	var err error
	db, container, err = testinfra.SetupTestDB(ctx)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	if container != nil {
		err := container.Terminate(ctx)
		Expect(err).ToNot(HaveOccurred())
	}
})
```

- [ ] **Step 2: Write the failing tests**

Create `internal/task/repository_test.go`:
```go
package task_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/philiplambok/tudu/internal/common/testinfra"
	"github.com/philiplambok/tudu/internal/common/util"
	"github.com/philiplambok/tudu/internal/task"
)

const fixtureUserID = int64(1) // pre-seeded in users.yml

func newTask(title string) task.Task {
	return task.NewTask(task.CreateTaskRecordDTO{
		UserID:      fixtureUserID,
		Title:       title,
		Description: "test description",
	})
}

var _ = Describe("Repository", func() {
	var repo task.Repository

	BeforeEach(func() {
		err := testinfra.RestoreDB(ctx, container, &db)
		Expect(err).ToNot(HaveOccurred())
		repo = task.NewRepository(db)
	})

	Describe("Create", func() {
		It("persists the task with pending status and records a created activity", func() {
			result, err := repo.Create(ctx, newTask("Buy groceries"))
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).ToNot(BeZero())
			Expect(result.UserID).To(Equal(fixtureUserID))
			Expect(result.Title).To(Equal("Buy groceries"))
			Expect(result.Status).To(Equal(task.StatusPending))
			Expect(result.CreatedAt).ToNot(BeZero())

			activities, err := repo.ListActivities(ctx, fixtureUserID, result.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(activities).To(HaveLen(1))
			Expect(activities[0].Action).To(Equal(task.ActivityActionCreated))
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			_, err := repo.Create(ctx, newTask("Task A"))
			Expect(err).ToNot(HaveOccurred())
			agg := newTask("Task B")
			_, err = repo.Create(ctx, agg)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns all tasks for the user ordered by created_at DESC", func() {
			records, total, err := repo.List(ctx, task.ListTaskRecordParams{
				UserID:        fixtureUserID,
				PagingRequest: util.PagingRequest{Page: 1, Limit: 20},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(2))
		})

		It("filters by status", func() {
			records, total, err := repo.List(ctx, task.ListTaskRecordParams{
				UserID:        fixtureUserID,
				Status:        task.StatusCompleted,
				PagingRequest: util.PagingRequest{Page: 1, Limit: 20},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(0)))
			Expect(records).To(BeEmpty())
		})

		It("respects limit and offset", func() {
			records, total, err := repo.List(ctx, task.ListTaskRecordParams{
				UserID:        fixtureUserID,
				PagingRequest: util.PagingRequest{Page: 1, Limit: 1},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(1))
		})
	})

	Describe("Get", func() {
		var created *task.TaskRecordDTO

		BeforeEach(func() {
			var err error
			created, err = repo.Create(ctx, newTask("Find me"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns the task for the correct user and ID", func() {
			result, err := repo.Get(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(created.ID))
			Expect(result.Title).To(Equal("Find me"))
		})

		It("returns ErrNotFound for the wrong userID", func() {
			_, err := repo.Get(ctx, 9999, created.ID)
			Expect(err).To(Equal(task.ErrNotFound))
		})

		It("returns ErrNotFound for an unknown task ID", func() {
			_, err := repo.Get(ctx, fixtureUserID, 9999)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("Update", func() {
		var created *task.TaskRecordDTO

		BeforeEach(func() {
			var err error
			created, err = repo.Create(ctx, newTask("Old title"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("updates fields and records an activity for each changed field", func() {
			newTitle := "New title"
			agg := task.TaskFromRecord(*created)
			updated := agg.ApplyUpdate(task.UpdateRequestDTO{Title: &newTitle})

			result, err := repo.Update(ctx, fixtureUserID, updated)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Title).To(Equal("New title"))

			activities, err := repo.ListActivities(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			// 1 "created" activity from Create + 1 "updated" activity from Update
			Expect(activities).To(HaveLen(2))
			Expect(activities[0].Action).To(Equal(task.ActivityActionUpdated))
			Expect(activities[0].FieldName).ToNot(BeNil())
			Expect(*activities[0].FieldName).To(Equal("title"))
		})

		It("returns ErrNotFound for an unknown task ID", func() {
			agg := task.Task{ID: 9999, UserID: fixtureUserID, Title: "x"}
			_, err := repo.Update(ctx, fixtureUserID, agg)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("Complete", func() {
		var created *task.TaskRecordDTO

		BeforeEach(func() {
			var err error
			created, err = repo.Create(ctx, newTask("Finish me"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("sets status to completed and records completed_at", func() {
			result, err := repo.Complete(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Status).To(Equal(task.StatusCompleted))
			Expect(result.CompletedAt).ToNot(BeNil())
		})

		It("returns ErrNotFound for an unknown task ID", func() {
			_, err := repo.Complete(ctx, fixtureUserID, 9999)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		var created *task.TaskRecordDTO

		BeforeEach(func() {
			var err error
			created, err = repo.Create(ctx, newTask("Delete me"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("deletes the task so it can no longer be retrieved", func() {
			err := repo.Delete(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())

			_, err = repo.Get(ctx, fixtureUserID, created.ID)
			Expect(err).To(Equal(task.ErrNotFound))
		})

		It("returns ErrNotFound for an unknown task ID", func() {
			err := repo.Delete(ctx, fixtureUserID, 9999)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("ListActivities", func() {
		It("returns activities ordered by created_at DESC", func() {
			created, err := repo.Create(ctx, newTask("Activity task"))
			Expect(err).ToNot(HaveOccurred())

			newTitle := "Updated title"
			agg := task.TaskFromRecord(*created)
			updated := agg.ApplyUpdate(task.UpdateRequestDTO{Title: &newTitle})
			_, err = repo.Update(ctx, fixtureUserID, updated)
			Expect(err).ToNot(HaveOccurred())

			activities, err := repo.ListActivities(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(activities).To(HaveLen(2))
			// ListActivities orders DESC by created_at, so "updated" comes before "created"
			Expect(activities[0].Action).To(Equal(task.ActivityActionUpdated))
			Expect(activities[1].Action).To(Equal(task.ActivityActionCreated))
		})

		It("returns ErrNotFound when the task does not belong to the user", func() {
			created, err := repo.Create(ctx, newTask("Private task"))
			Expect(err).ToNot(HaveOccurred())

			_, err = repo.ListActivities(ctx, 9999, created.ID)
			Expect(err).To(Equal(task.ErrNotFound))
		})
	})

	Describe("Update with DueDate", func() {
		It("records a due_date activity when due_date changes", func() {
			dueDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
			agg := task.NewTask(task.CreateTaskRecordDTO{
				UserID:  fixtureUserID,
				Title:   "Dated task",
				DueDate: &dueDate,
			})
			created, err := repo.Create(ctx, agg)
			Expect(err).ToNot(HaveOccurred())

			newDue := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
			updated := task.TaskFromRecord(*created)
			withUpdate := updated.ApplyUpdate(task.UpdateRequestDTO{DueDate: &newDue})
			result, err := repo.Update(ctx, fixtureUserID, withUpdate)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.DueDate).ToNot(BeNil())

			activities, err := repo.ListActivities(ctx, fixtureUserID, created.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(activities).To(HaveLen(2))
			Expect(*activities[0].FieldName).To(Equal("due_date"))
		})
	})
})
```

- [ ] **Step 3: Run the tests and verify they pass**

```bash
./dx/test ./internal/task/... -v
# Expected: All specs pass under "Task Repository Suite"
# First run boots a container and runs migrations (~10-15s total)
```

- [ ] **Step 4: Run the full test suite to confirm no regressions**

```bash
./dx/test ./...
# Expected: all existing unit tests plus the new repository tests pass
```

- [ ] **Step 5: Commit**

```bash
git add internal/task/suite_test.go internal/task/repository_test.go
git commit -m "test: add testcontainers integration tests for task repository"
```

---

## Notes

### Import aliases in testinfra.go

The `postgres` name collides between testcontainers and gorm driver. Use:
```go
import (
    postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
    gormpostgres       "gorm.io/driver/postgres"
)
```

### Running a single test

```bash
./dx/test ./internal/task/... -v -run "TestTaskRepository/Complete"
```

### Test isolation guarantee

Each `BeforeEach` calls `testinfra.RestoreDB` which drops and recreates the `tudu_test` database from the clean snapshot (containing only fixture user with ID=1). Tests that seed additional data (tasks, extra users) get a clean slate on the next test.

### Why snapshot/restore instead of transactions

PostgreSQL `CREATE DATABASE ... TEMPLATE` restores are faster than per-test DDL transactions and work across connection pool boundaries. Goose schema migrations and testfixtures base data are baked in once; only test-specific data is seeded per test.
