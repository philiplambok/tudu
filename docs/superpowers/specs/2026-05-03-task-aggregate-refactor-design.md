# Task Aggregate Refactor Design

## Goal

Move task activity creation logic out of the repository layer into the domain layer. The repository becomes a dumb persistence layer that inserts pre-built records. The domain owns what activity rows to create and when.

## Motivation

Currently `repository.Update` reads the task before and after applying changes, diffs the fields, and decides which activity rows to insert — all inside a single transaction. This means the repository contains business logic (which fields are audited, how they are compared, what constitutes a change). The domain layer (domain.go) holds only constants and validation.

The refactor makes the domain aggregate responsible for that logic. The repository is left with a single job: persist whatever it is given.

---

## Domain Aggregate

### Structs — `internal/task/domain.go`

Two domain structs are added alongside the existing constants and validation functions:

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

    Activities []TaskActivity
}

type TaskActivity struct {
    ID        int64
    TaskID    int64
    UserID    int64
    Action    string
    FieldName *string
    OldValue  *string
    NewValue  *string
    CreatedAt time.Time
}
```

`Task.Activities` is a pending-activity list. It is populated by domain methods, then handed to the repository which flushes it. After persistence the field is not used again in that request.

### Domain methods — `internal/task/domain.go`

```go
// NewCreatedTask builds a Task aggregate with a single "created" activity.
func NewCreatedTask(rec TaskRecordDTO) Task

// UpdatedActivities computes which activity rows result from a change
// between this Task's state (before) and after.
func (t Task) UpdatedActivities(after Task) []TaskActivity
```

`UpdatedActivities` compares `title`, `description`, and `due_date` between `t` (before) and `after`, returning one `TaskActivity` per changed field with `Action = ActivityActionUpdated`. Fields are compared as formatted strings (RFC3339Nano for time, raw string otherwise) consistent with the current comparison logic.

These are pure functions with no side effects and no dependencies — straightforward to unit-test.

---

## Repository Contract

### New repository methods

```go
// CreateWithActivity persists a task row and its activity in one transaction.
CreateWithActivity(ctx context.Context, rec CreateTaskRecordDTO, activity TaskActivityRecordDTO) (*TaskRecordDTO, error)

// UpdateWithActivities locks the task row, applies the update, and inserts
// the pre-computed activity rows — all in one transaction. Returns the
// updated task record.
UpdateWithActivities(ctx context.Context, userID int64, id int64, req UpdateRequestDTO, activities []TaskActivityRecordDTO) (*TaskRecordDTO, error)
```

The existing `Create` and `Update` methods are **removed** and replaced by the above. All other methods (`List`, `Get`, `Complete`, `Delete`, `ListActivities`) are unchanged.

`insertTaskActivity`, `stringPtr`, `timeValue`, `textValue`, `valuesEqual` are **removed** from repository.go. String/time formatting helpers move to domain.go where they are needed by `UpdatedActivities`.

### Why not a generic `CreateActivities` separate call?

Separating activity writes from task writes would require the service to manage transactions across two repository calls. Keeping them in one atomic repository method preserves the existing audit guarantee: task mutation and audit record always land together or not at all.

---

## Service Flow

### `service.Create`

```
1. validate request
2. build CreateTaskRecordDTO
3. build activity = TaskActivityRecordDTO{Action: ActivityActionCreated}
4. repo.CreateWithActivity(ctx, rec, activity) → TaskRecordDTO
5. return toResponseDTO
```

The domain produces the activity record before the repo call; repo persists both atomically. `NewCreatedTask` is a convenience constructor for producing the activity record in step 3.

### `service.Update`

The service cannot compute the final activity rows before the transaction, because the "after" state is only known once the update is applied. The solution is a callback pattern: the service supplies a pure function to the repo; the repo calls it with the real after-row inside the transaction, then inserts the results.

The cleanest split is: the service computes activities inside a callback, then passes the callback to the repo which does the locked update + insert atomically:

```
1. validate request
2. before := repo.Get(ctx, userID, id)            → TaskRecordDTO
3. beforeTask := domain.TaskFromRecord(before)
4. after, err := repo.UpdateWithActivities(
       ctx, userID, id, req,
       func(after TaskRecordDTO) []TaskActivityRecordDTO {
           afterTask := domain.TaskFromRecord(after)
           activities := beforeTask.UpdatedActivities(afterTask)
           return toActivityRecordDTOs(activities)
       },
   )
```

This is the most correct form: the repo reads-with-lock, applies the update, gets the after state, calls the callback to compute activities with the real after values, then inserts them — all in one transaction. The callback is a pure function supplied by the service.

The `UpdateWithActivities` signature becomes:

```go
UpdateWithActivities(
    ctx context.Context,
    userID int64,
    id int64,
    req UpdateRequestDTO,
    activityFn func(after TaskRecordDTO) []TaskActivityRecordDTO,
) (*TaskRecordDTO, error)
```

This keeps the repository's transaction boundary intact while keeping activity-building logic in the domain/service side. The repository decides only *when* to call the callback (after the locked update has produced the after-row); the callback decides *what* activities should be created.

---

## DTO changes

`TaskActivityRecordDTO` stays as the service↔repo boundary type. Domain-side `TaskActivity` is a value type used only within the service call scope — it is not persisted directly, it is mapped to `TaskActivityRecordDTO` before being handed to the repo.

A helper `toActivityRecordDTOs([]TaskActivity) []TaskActivityRecordDTO` lives in service.go.

`TaskRecordDTO` gains a helper `domain.TaskFromRecord(rec TaskRecordDTO) Task` to convert a flat DTO into a domain Task for the purpose of running `UpdatedActivities`.

---

## Files Changed

| File | Change |
|------|--------|
| `internal/task/domain.go` | Add `Task`, `TaskActivity` structs; add `NewCreatedTask`, `Task.UpdatedActivities`, `TaskFromRecord` methods; move string/time formatting helpers here |
| `internal/task/repository.go` | Replace `Create`/`Update` with `CreateWithActivity`/`UpdateWithActivities(activityFn)`; remove helper functions now in domain |
| `internal/task/service.go` | Update `Create` and `Update` to use new repo methods; add `toActivityRecordDTOs` helper |
| `internal/task/service_test.go` | Update mock and existing tests to match new repo interface |

---

## Testing

Domain methods (`UpdatedActivities`, `NewCreatedTask`, `TaskFromRecord`) are pure functions — test them directly in `domain_test.go`:

- `UpdatedActivities` with changed title, changed description, changed due_date, no changes, multiple changes
- `NewCreatedTask` returns a Task with exactly one `ActivityActionCreated` activity

Service tests (`service_test.go`) mock the repository interface (which changes signature). Existing service tests are updated to match the new `CreateWithActivity`/`UpdateWithActivities` signatures.

---

## Non-goals

- No changes to handler, endpoint, or OpenAPI spec.
- No new endpoints.
- No changes to `Complete` or `Delete` (they do not produce activity records).
- No migration — schema is unchanged.
