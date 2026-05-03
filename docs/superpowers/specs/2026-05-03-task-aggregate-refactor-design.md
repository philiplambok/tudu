# Task Aggregate Refactor Design

## Goal

Move task activity creation logic out of the repository layer into the domain layer. The repository becomes a dumb persistence layer: it receives a fully-formed domain aggregate and persists it. The domain owns what activity rows to create and when.

## Motivation

Currently `repository.Update` reads the task before and after applying changes, diffs the fields, and decides which activity rows to insert — all inside a single transaction. This means the repository contains business logic (which fields are audited, how they are compared, what constitutes a change). The domain layer (domain.go) holds only constants and validation.

The refactor makes the domain aggregate responsible for that logic. Every mutation follows the same three-step pattern:

1. **Repo read** — fetch existing data needed to build the aggregate.
2. **Domain** — build or mutate the aggregate; it computes its own pending activities.
3. **Repo write** — pass the aggregate to the repo; the repo generates datamodel structs and runs the SQL atomically.

---

## Domain Aggregate

### Structs — `internal/task/domain.go`

Two domain structs alongside existing constants and validation:

```go
type Task struct {
    ID          int64
    UserID      int64
    Title       string
    Description string
    Status      string
    DueDate     *time.Time
    CompletedAt *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time

    Activities []TaskActivity  // pending activity rows to be flushed by the repo
}

type TaskActivity struct {
    TaskID    int64
    UserID    int64
    Action    string
    FieldName *string
    OldValue  *string
    NewValue  *string
}
```

`Task.Activities` holds activity records that should be inserted alongside the task write. The repo flushes them atomically; after the write they are not used again in that request. `TaskActivity` has no `ID` or `CreatedAt` — the repo sets those at insert time.

### Domain methods — `internal/task/domain.go`

```go
// TaskFromRecord converts a flat TaskRecordDTO into a domain Task.
func TaskFromRecord(rec TaskRecordDTO) Task

// NewTask builds a new (unsaved) Task aggregate from a CreateTaskRecordDTO,
// with a single "created" activity pre-populated.
func NewTask(rec CreateTaskRecordDTO) Task

// ApplyUpdate returns a new Task with the requested fields changed and
// the field-level "updated" activities populated for each changed field.
func (t Task) ApplyUpdate(req UpdateRequestDTO) Task
```

`ApplyUpdate` compares `title`, `description`, and `due_date` between the receiver (before) and the new values in `req`, returning one `TaskActivity` per changed field. Fields are compared as strings (RFC3339Nano for times, raw string for text). This logic currently lives in `repository.Update`; it moves here as a pure function.

String/time formatting helpers (`stringPtr`, `timeValue`, `textValue`, `valuesEqual`) move from `repository.go` to `domain.go`.

---

## Repository Contract

### Simplified interface

```go
type Repository interface {
    Create(ctx context.Context, task Task) (*TaskRecordDTO, error)
    List(ctx context.Context, userID int64, status string) ([]TaskRecordDTO, error)
    Get(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error)
    Update(ctx context.Context, userID int64, task Task) (*TaskRecordDTO, error)
    Complete(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error)
    Delete(ctx context.Context, userID int64, id int64) error
    ListActivities(ctx context.Context, userID int64, taskID int64) ([]TaskActivityRecordDTO, error)
}
```

`Create` and `Update` now accept a domain `Task` aggregate instead of a flat DTO or inline update map. The repo is responsible for:

- Converting `Task` fields into `datamodel.Task` and running the INSERT/UPDATE.
- Converting each `Task.Activities` entry into `datamodel.TaskActivity` rows and inserting them.
- Wrapping both in a single transaction.

Everything else (`List`, `Get`, `Complete`, `Delete`, `ListActivities`) is unchanged.

The helpers `insertTaskActivity`, `stringPtr`, `timeValue`, `textValue`, `valuesEqual` are **removed** from `repository.go` — they move to `domain.go`.

---

## Service Flow

### `service.Create`

```
1. validate request
2. task := domain.NewTask(CreateTaskRecordDTO{UserID, Title, Description, DueDate})
   // task.Activities = [{Action: "created"}]
3. saved, err := repo.Create(ctx, task)
4. return toResponseDTO(saved)
```

One repo call. The aggregate carries the activity; the repo inserts both atomically.

### `service.Update`

```
1. validate request
2. rec, err := repo.Get(ctx, userID, id)     // read current state
3. before := domain.TaskFromRecord(rec)
4. after := before.ApplyUpdate(req)           // mutates fields + populates Activities
   // after.Activities = [{Action: "updated", FieldName: "title", ...}, ...]
5. saved, err := repo.Update(ctx, userID, after)
6. return toResponseDTO(saved)
```

Two repo calls (one read, one write), but a single atomic write. `repo.Update` locks the row, applies the update, and inserts `after.Activities` in one transaction.

The "before" read outside the transaction is acceptable because `repo.Update` internally does `SELECT … FOR UPDATE` before applying changes, guaranteeing that the row seen during activity insertion matches the row that was actually updated. The service-side `before` is used only to compute activities — the repo's locked read is the source of truth for the update.

---

## DTO changes

`TaskRecordDTO` is unchanged — it remains the repo→service return type.

`TaskActivityRecordDTO` is no longer needed as a service↔repo input type. The repo accepts `TaskActivity` (domain) directly via the aggregate and converts it to `datamodel.TaskActivity` internally. `TaskActivityRecordDTO` is kept only as the repo→service return type for `ListActivities`.

`toActivityRecordDTOs` helper in `service.go` is removed (no longer needed).

---

## Files Changed

| File | Change |
|------|--------|
| `internal/task/domain.go` | Add `Task`, `TaskActivity` structs; add `TaskFromRecord`, `NewTask`, `Task.ApplyUpdate`; move string/time helpers here; add `import "time"` |
| `internal/task/repository.go` | Change `Create(rec) → Create(task Task)`; change `Update(userID, id, req) → Update(userID, task Task)`; repo converts domain→datamodel internally; remove helpers now in domain |
| `internal/task/service.go` | Update `Create` and `Update` to use domain aggregate pattern; remove `toActivityRecordDTOs` |
| `internal/task/service_test.go` | Update mock signatures and tests to match new `Create(Task)` / `Update(userID, Task)` interface |
| `internal/task/domain_test.go` | New file: pure unit tests for `TaskFromRecord`, `NewTask`, `ApplyUpdate` |

---

## Testing

`domain_test.go` tests the pure domain functions:

- `NewTask` returns a Task with exactly one `ActivityActionCreated` activity and no ID.
- `ApplyUpdate` with only title changed → one updated activity for title.
- `ApplyUpdate` with title and description changed → two updated activities.
- `ApplyUpdate` with no changes → zero activities.
- `ApplyUpdate` with due_date changed → one updated activity for due_date.
- `TaskFromRecord` round-trips a `TaskRecordDTO` correctly.

`service_test.go` mock is updated for the new `Create(Task)` / `Update(userID, Task)` signatures. Existing service behaviour tests remain — only the mock interface changes.

---

## Non-goals

- No changes to handler, endpoint, or OpenAPI spec.
- No new endpoints.
- No changes to `Complete` or `Delete` (they do not produce activity records).
- No migration — schema is unchanged.
