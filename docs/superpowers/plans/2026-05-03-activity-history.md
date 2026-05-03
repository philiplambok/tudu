# Activity History Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add task activity history so the API records audit rows whenever a task is created or updated, and exposes task-specific activity history to authenticated users.

**Architecture:** Add a new `task_activities` table and keep activity writes inside the task repository so task mutation and audit insertion happen in the same database transaction. Follow the existing DTO boundary convention: handler/service use Request/Response DTOs, service/repository use Record DTOs, repositories scan database rows into `internal/common/datamodel` structs before mapping to DTOs.

**Tech Stack:** Go, chi, GORM, PostgreSQL, goose migrations, oapi-codegen OpenAPI models, existing JWT auth middleware.

---

## File Structure

- Create: `db/migrations/20260503000004_create_task_activities.sql`
  - Defines `task_activities` table with task/user ownership, action, field-level change details, and timestamps.
- Create: `internal/common/datamodel/task_activity.go`
  - Table-representative struct for `task_activities` rows.
- Modify: `internal/task/domain.go`
  - Add task activity action constants.
- Modify: `internal/task/dto.go`
  - Add activity Record/Response DTOs.
- Modify: `internal/task/repository.go`
  - Insert activity rows inside `Create` and `Update`; add `ListActivities` query and datamodel mapping.
- Modify: `internal/task/service.go`
  - Add `ListActivities`; keep service response mapping separate from repository records.
- Modify: `internal/task/service_test.go`
  - Update mock repository and add service-level coverage for activity listing.
- Modify: `internal/task/handler.go`
  - Add HTTP handler and OpenAPI response mapping for activity list.
- Modify: `internal/task/endpoint.go`
  - Add route `GET /v1/tasks/{id}/activities`.
- Modify: `api/openapi.yml`
  - Add path, schemas, and response types for activity history.
- Generate: `pkg/openapi/v1/openapi.gen.go`
  - Regenerate after OpenAPI changes.

## Data Model

`task_activities` stores one row for task creation and one row for each changed field in task update.

Example rows:

| action | field_name | old_value | new_value |
| --- | --- | --- | --- |
| `created` | `null` | `null` | `null` |
| `updated` | `title` | `Buy milk` | `Buy oat milk` |
| `updated` | `due_date` | `2026-05-04T10:00:00Z` | `2026-05-05T10:00:00Z` |

This avoids JSONB complexity for now and keeps the first implementation simple and queryable.

---

### Task 1: Add database migration and datamodel

**Files:**
- Create: `db/migrations/20260503000004_create_task_activities.sql`
- Create: `internal/common/datamodel/task_activity.go`

- [ ] **Step 1: Create the migration**

Create `db/migrations/20260503000004_create_task_activities.sql`:

```sql
-- +goose Up
CREATE TABLE task_activities (
    id          BIGSERIAL PRIMARY KEY,
    task_id     BIGINT       NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id     BIGINT       NOT NULL REFERENCES users(id),
    action      TEXT         NOT NULL,
    field_name  TEXT,
    old_value   TEXT,
    new_value   TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX task_activities_task_id_idx ON task_activities(task_id);
CREATE INDEX task_activities_user_id_idx ON task_activities(user_id);
CREATE INDEX task_activities_created_at_idx ON task_activities(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS task_activities;
```

- [ ] **Step 2: Create the datamodel struct**

Create `internal/common/datamodel/task_activity.go`:

```go
package datamodel

import "time"

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

- [ ] **Step 3: Verify the package compiles**

Run:

```bash
go test ./internal/common/datamodel ./internal/task
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add db/migrations/20260503000004_create_task_activities.sql internal/common/datamodel/task_activity.go
git commit -m "feat: add task activity history table"
```

---

### Task 2: Add task activity DTOs and repository contract

**Files:**
- Modify: `internal/task/domain.go`
- Modify: `internal/task/dto.go`
- Modify: `internal/task/repository.go`

- [ ] **Step 1: Add activity action constants**

Update `internal/task/domain.go` so it contains these constants:

```go
package task

const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
)

const (
	ActivityActionCreated = "created"
	ActivityActionUpdated = "updated"
)
```

- [ ] **Step 2: Add activity DTOs**

Append to `internal/task/dto.go`:

```go
type TaskActivityRecordDTO struct {
	ID        int64
	TaskID    int64
	UserID    int64
	Action    string
	FieldName *string
	OldValue  *string
	NewValue  *string
	CreatedAt time.Time
}

type TaskActivityResponseDTO struct {
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

Keep the existing `import "time"` in `dto.go`.

- [ ] **Step 3: Extend the repository interface**

Update `internal/task/repository.go` repository interface:

```go
type Repository interface {
	Create(ctx context.Context, rec CreateTaskRecordDTO) (*TaskRecordDTO, error)
	List(ctx context.Context, userID int64, status string) ([]TaskRecordDTO, error)
	Get(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error)
	Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskRecordDTO, error)
	Complete(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error)
	Delete(ctx context.Context, userID int64, id int64) error
	ListActivities(ctx context.Context, userID int64, taskID int64) ([]TaskActivityRecordDTO, error)
}
```

- [ ] **Step 4: Add the activity mapper**

Add this function to `internal/task/repository.go` after `toTaskRecordDTO`:

```go
func toTaskActivityRecordDTO(m *datamodel.TaskActivity) *TaskActivityRecordDTO {
	return &TaskActivityRecordDTO{
		ID:        m.ID,
		TaskID:    m.TaskID,
		UserID:    m.UserID,
		Action:    m.Action,
		FieldName: m.FieldName,
		OldValue:  m.OldValue,
		NewValue:  m.NewValue,
		CreatedAt: m.CreatedAt,
	}
}
```

- [ ] **Step 5: Run compile check**

Run:

```bash
go test ./internal/task
```

Expected: FAIL because the mock repository in `internal/task/service_test.go` does not implement `ListActivities` yet. This confirms the interface change is visible to tests.

Do not commit this task until Task 3 updates the service tests.

---

### Task 3: Add service activity listing with TDD

**Files:**
- Modify: `internal/task/service_test.go`
- Modify: `internal/task/service.go`

- [ ] **Step 1: Update mock repository**

Add a function field to `mockTaskRepo` in `internal/task/service_test.go`:

```go
listActivitiesFn func(ctx context.Context, userID int64, taskID int64) ([]task.TaskActivityRecordDTO, error)
```

Add this method to `mockTaskRepo`:

```go
func (m *mockTaskRepo) ListActivities(ctx context.Context, userID int64, taskID int64) ([]task.TaskActivityRecordDTO, error) {
	return m.listActivitiesFn(ctx, userID, taskID)
}
```

- [ ] **Step 2: Write failing service test**

Append to `internal/task/service_test.go`:

```go
func TestListActivities_Success(t *testing.T) {
	now := time.Now()
	fieldName := "title"
	oldValue := "Buy milk"
	newValue := "Buy oat milk"

	repo := &mockTaskRepo{
		listActivitiesFn: func(_ context.Context, userID int64, taskID int64) ([]task.TaskActivityRecordDTO, error) {
			if userID != 1 {
				t.Fatalf("expected userID 1, got %d", userID)
			}
			if taskID != 5 {
				t.Fatalf("expected taskID 5, got %d", taskID)
			}
			return []task.TaskActivityRecordDTO{
				{
					ID:        10,
					TaskID:    taskID,
					UserID:    userID,
					Action:    task.ActivityActionUpdated,
					FieldName: &fieldName,
					OldValue:  &oldValue,
					NewValue:  &newValue,
					CreatedAt: now,
				},
			}, nil
		},
	}

	svc := task.NewService(repo)
	got, err := svc.ListActivities(context.Background(), 1, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(got))
	}
	if got[0].Action != task.ActivityActionUpdated {
		t.Errorf("expected action updated, got %s", got[0].Action)
	}
	if got[0].FieldName == nil || *got[0].FieldName != "title" {
		t.Fatalf("expected field_name title, got %v", got[0].FieldName)
	}
}
```

- [ ] **Step 3: Run test to verify failure**

Run:

```bash
go test ./internal/task -run TestListActivities_Success -v
```

Expected: FAIL with `svc.ListActivities undefined` or `task.Service has no field or method ListActivities`.

- [ ] **Step 4: Add service method and response mapper**

Update `internal/task/service.go` interface:

```go
type Service interface {
	Create(ctx context.Context, userID int64, req CreateRequestDTO) (*TaskResponseDTO, error)
	List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error)
	Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error)
	Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Delete(ctx context.Context, userID int64, id int64) error
	ListActivities(ctx context.Context, userID int64, taskID int64) ([]TaskActivityResponseDTO, error)
}
```

Add this method to `internal/task/service.go` after `Delete`:

```go
func (s *service) ListActivities(ctx context.Context, userID int64, taskID int64) ([]TaskActivityResponseDTO, error) {
	recs, err := s.repo.ListActivities(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]TaskActivityResponseDTO, len(recs))
	for i := range recs {
		out[i] = *toActivityResponseDTO(&recs[i])
	}
	return out, nil
}
```

Add this mapper after `toResponseDTO`:

```go
func toActivityResponseDTO(r *TaskActivityRecordDTO) *TaskActivityResponseDTO {
	return &TaskActivityResponseDTO{
		ID:        r.ID,
		TaskID:    r.TaskID,
		UserID:    r.UserID,
		Action:    r.Action,
		FieldName: r.FieldName,
		OldValue:  r.OldValue,
		NewValue:  r.NewValue,
		CreatedAt: r.CreatedAt,
	}
}
```

- [ ] **Step 5: Run service tests**

Run:

```bash
go test ./internal/task -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/task/domain.go internal/task/dto.go internal/task/repository.go internal/task/service.go internal/task/service_test.go
git commit -m "feat: add task activity service contract"
```

---

### Task 4: Record created and updated task activities in repository

**Files:**
- Modify: `internal/task/repository.go`

- [ ] **Step 1: Add helper imports**

Update the `internal/task/repository.go` imports to include `fmt` and `time`:

```go
import (
	"context"
	"fmt"
	"time"

	"github.com/philiplambok/tudu/internal/common/datamodel"
	"gorm.io/gorm"
)
```

- [ ] **Step 2: Add activity insertion helper**

Add to `internal/task/repository.go`:

```go
func insertTaskActivity(ctx context.Context, db *gorm.DB, taskID int64, userID int64, action string, fieldName *string, oldValue *string, newValue *string) error {
	return db.WithContext(ctx).Exec(`
		INSERT INTO task_activities (task_id, user_id, action, field_name, old_value, new_value, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())`,
		taskID, userID, action, fieldName, oldValue, newValue,
	).Error
}
```

- [ ] **Step 3: Add value formatting helpers**

Add to `internal/task/repository.go`:

```go
func stringPtr(v string) *string {
	return &v
}

func timeValue(t *time.Time) *string {
	if t == nil {
		return nil
	}
	return stringPtr(t.UTC().Format(time.RFC3339))
}

func textValue(v string) *string {
	return stringPtr(v)
}

func valuesEqual(a *string, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
```

- [ ] **Step 4: Update Create to run in a transaction and insert created activity**

Replace `Create` in `internal/task/repository.go` with:

```go
func (r *repository) Create(ctx context.Context, rec CreateTaskRecordDTO) (*TaskRecordDTO, error) {
	var out *TaskRecordDTO
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row datamodel.Task
		res := tx.Raw(`
			INSERT INTO tasks (user_id, title, description, status, due_date, created_at, updated_at)
			VALUES (?, ?, ?, 'pending', ?, NOW(), NOW())
			RETURNING id, user_id, title, description, status, due_date, completed_at, created_at, updated_at`,
			rec.UserID, rec.Title, rec.Description, rec.DueDate,
		).Scan(&row)
		if res.Error != nil {
			return res.Error
		}
		if err := insertTaskActivity(ctx, tx, row.ID, row.UserID, ActivityActionCreated, nil, nil, nil); err != nil {
			return err
		}
		out = toTaskRecordDTO(&row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **Step 5: Update Update to capture old values and insert field-level activities**

Replace `Update` in `internal/task/repository.go` with:

```go
func (r *repository) Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskRecordDTO, error) {
	var out *TaskRecordDTO
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before datamodel.Task
		getBefore := tx.Raw(`
			SELECT id, user_id, title, description, status, due_date, completed_at, created_at, updated_at
			FROM tasks WHERE id = ? AND user_id = ?`, id, userID,
		).Scan(&before)
		if getBefore.Error != nil {
			return getBefore.Error
		}
		if getBefore.RowsAffected == 0 {
			return ErrNotFound
		}

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

		res := tx.Table("tasks").Where("id = ? AND user_id = ?", id, userID).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}

		var after datamodel.Task
		getAfter := tx.Raw(`
			SELECT id, user_id, title, description, status, due_date, completed_at, created_at, updated_at
			FROM tasks WHERE id = ? AND user_id = ?`, id, userID,
		).Scan(&after)
		if getAfter.Error != nil {
			return getAfter.Error
		}
		if getAfter.RowsAffected == 0 {
			return ErrNotFound
		}

		changes := []struct {
			field string
			old   *string
			new   *string
		}{
			{field: "title", old: textValue(before.Title), new: textValue(after.Title)},
			{field: "description", old: textValue(before.Description), new: textValue(after.Description)},
			{field: "due_date", old: timeValue(before.DueDate), new: timeValue(after.DueDate)},
		}

		for _, change := range changes {
			if valuesEqual(change.old, change.new) {
				continue
			}
			fieldName := change.field
			if err := insertTaskActivity(ctx, tx, after.ID, after.UserID, ActivityActionUpdated, &fieldName, change.old, change.new); err != nil {
				return fmt.Errorf("insert task activity: %w", err)
			}
		}

		out = toTaskRecordDTO(&after)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **Step 6: Run task tests**

Run:

```bash
go test ./internal/task -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/task/repository.go
git commit -m "feat: record task create and update activities"
```

---

### Task 5: Query task activity history from repository

**Files:**
- Modify: `internal/task/repository.go`

- [ ] **Step 1: Add repository method**

Add to `internal/task/repository.go` before `toTaskRecordDTO`:

```go
func (r *repository) ListActivities(ctx context.Context, userID int64, taskID int64) ([]TaskActivityRecordDTO, error) {
	var taskRow datamodel.Task
	taskRes := r.db.WithContext(ctx).Raw(`
		SELECT id, user_id, title, description, status, due_date, completed_at, created_at, updated_at
		FROM tasks WHERE id = ? AND user_id = ?`, taskID, userID,
	).Scan(&taskRow)
	if taskRes.Error != nil {
		return nil, taskRes.Error
	}
	if taskRes.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	var rows []datamodel.TaskActivity
	if err := r.db.WithContext(ctx).
		Select("id, task_id, user_id, action, field_name, old_value, new_value, created_at").
		Table("task_activities").
		Where("task_id = ? AND user_id = ?", taskID, userID).
		Order("created_at DESC, id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]TaskActivityRecordDTO, len(rows))
	for i := range rows {
		out[i] = *toTaskActivityRecordDTO(&rows[i])
	}
	return out, nil
}
```

- [ ] **Step 2: Run compile check**

Run:

```bash
go test ./internal/task -v
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/task/repository.go
git commit -m "feat: list task activity history"
```

---

### Task 6: Add OpenAPI schema and regenerate models

**Files:**
- Modify: `api/openapi.yml`
- Generate: `pkg/openapi/v1/openapi.gen.go`

- [ ] **Step 1: Add activity history endpoint to OpenAPI**

Insert this path after `/v1/tasks/{id}` and before `/v1/tasks/{id}/complete` in `api/openapi.yml`:

```yaml
  /v1/tasks/{id}/activities:
    parameters:
      - $ref: "#/components/parameters/IDParam"
    get:
      tags: [Tasks]
      summary: List task activity history
      operationId: listTaskActivities
      security:
        - bearerAuth: []
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/TaskActivityListResponse"
        "404":
          $ref: "#/components/responses/NotFound"
```

- [ ] **Step 2: Add activity schemas to OpenAPI**

Insert these schemas after `TaskListResponse` in `api/openapi.yml`:

```yaml
    TaskActivityAction:
      type: string
      enum: [created, updated]

    TaskActivity:
      type: object
      required: [id, task_id, user_id, action, created_at]
      properties:
        id:
          type: integer
          format: int64
        task_id:
          type: integer
          format: int64
        user_id:
          type: integer
          format: int64
        action:
          $ref: "#/components/schemas/TaskActivityAction"
        field_name:
          type: string
          nullable: true
        old_value:
          type: string
          nullable: true
        new_value:
          type: string
          nullable: true
        created_at:
          type: string
          format: date-time

    TaskActivityListResponse:
      type: object
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: "#/components/schemas/TaskActivity"
```

- [ ] **Step 3: Regenerate OpenAPI models**

Run:

```bash
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config oapi_codegen.yml api/openapi.yml
```

Expected: `pkg/openapi/v1/openapi.gen.go` updates with `TaskActivity`, `TaskActivityAction`, and `TaskActivityListResponse` types.

- [ ] **Step 4: Run build**

Run:

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add api/openapi.yml pkg/openapi/v1/openapi.gen.go
git commit -m "feat: document task activity history API"
```

---

### Task 7: Add activity history HTTP handler and route

**Files:**
- Modify: `internal/task/handler.go`
- Modify: `internal/task/endpoint.go`

- [ ] **Step 1: Add handler method**

Add to `internal/task/handler.go` after `Get` and before `Update`:

```go
func (h *Handler) ListActivities(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	activities, err := h.svc.ListActivities(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to list task activities")
		return
	}

	data := make([]v1.TaskActivity, len(activities))
	for i := range activities {
		data[i] = toV1TaskActivity(&activities[i])
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskActivityListResponse{Data: data})
}
```

- [ ] **Step 2: Add OpenAPI mapper**

Add to `internal/task/handler.go` after `toV1Task`:

```go
func toV1TaskActivity(a *TaskActivityResponseDTO) v1.TaskActivity {
	return v1.TaskActivity{
		Id:        a.ID,
		TaskId:    a.TaskID,
		UserId:    a.UserID,
		Action:    v1.TaskActivityAction(a.Action),
		FieldName: a.FieldName,
		OldValue:  a.OldValue,
		NewValue:  a.NewValue,
		CreatedAt: a.CreatedAt,
	}
}
```

- [ ] **Step 3: Add route**

Update `internal/task/endpoint.go` routes:

```go
func (e *Endpoint) Routes() *chi.Mux {
	r := chi.NewMux()
	r.Post("/", e.handler.Create)
	r.Get("/", e.handler.List)
	r.Get("/{id}", e.handler.Get)
	r.Get("/{id}/activities", e.handler.ListActivities)
	r.Patch("/{id}", e.handler.Update)
	r.Post("/{id}/complete", e.handler.Complete)
	r.Delete("/{id}", e.handler.Delete)
	return r
}
```

- [ ] **Step 4: Run build and tests**

Run:

```bash
go build ./...
go test ./...
```

Expected: both PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/task/handler.go internal/task/endpoint.go
git commit -m "feat: expose task activity history endpoint"
```

---

### Task 8: Smoke test activity history manually

**Files:**
- No code changes expected.

- [ ] **Step 1: Start database**

Run:

```bash
docker compose -f docker-compose.dev.yml up -d postgres
```

Expected: PostgreSQL container starts.

- [ ] **Step 2: Run migrations**

Run:

```bash
go run . migrate
```

Expected: migration `20260503000004_create_task_activities.sql` is applied.

- [ ] **Step 3: Start API server**

Run:

```bash
go run . serve
```

Expected: server listens on configured port, usually `:8080`.

- [ ] **Step 4: Register and capture token**

Run in another terminal:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"activity@example.com","password":"password123"}' | jq -r '.token')
```

Expected: `TOKEN` is non-empty.

- [ ] **Step 5: Create task and capture id**

Run:

```bash
TASK_ID=$(curl -s -X POST http://localhost:8080/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Buy milk","description":"Original"}' | jq -r '.data.id')
```

Expected: `TASK_ID` is numeric.

- [ ] **Step 6: Update task**

Run:

```bash
curl -s -X PATCH "http://localhost:8080/v1/tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Buy oat milk","description":"Updated"}' | jq .
```

Expected: response data has updated title and description.

- [ ] **Step 7: List activity history**

Run:

```bash
curl -s "http://localhost:8080/v1/tasks/$TASK_ID/activities" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

Expected response shape:

```json
{
  "data": [
    {
      "action": "updated",
      "field_name": "description",
      "old_value": "Original",
      "new_value": "Updated"
    },
    {
      "action": "updated",
      "field_name": "title",
      "old_value": "Buy milk",
      "new_value": "Buy oat milk"
    },
    {
      "action": "created",
      "field_name": null,
      "old_value": null,
      "new_value": null
    }
  ]
}
```

The exact order of same-timestamp update rows can vary by id, but `created` must exist and at least one `updated` row must exist for each changed field.

- [ ] **Step 8: Commit smoke-test fixes only if needed**

If manual testing required code fixes, commit them:

```bash
git add <fixed-files>
git commit -m "fix: stabilize task activity history"
```

---

### Task 9: Final verification

**Files:**
- No code changes expected.

- [ ] **Step 1: Run formatting**

Run:

```bash
gofmt -w internal/common/datamodel/task_activity.go internal/task/domain.go internal/task/dto.go internal/task/repository.go internal/task/service.go internal/task/service_test.go internal/task/handler.go internal/task/endpoint.go
```

Expected: no output.

- [ ] **Step 2: Run full build**

Run:

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 3: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Check working tree**

Run:

```bash
git status --short
```

Expected: clean working tree or only intentionally uncommitted local files outside this feature.

---

## Self-Review

**Spec coverage:**
- Records audit when task is created: Task 4 inserts `created` activity in the same transaction as task creation.
- Records audit when task is updated: Task 4 inserts field-level `updated` activities in the same transaction as task update.
- Uses datamodel module for database scan targets: Task 1 creates `datamodel.TaskActivity`; Tasks 2 and 5 map datamodel to RecordDTO.
- Exposes activity history through API: Tasks 6 and 7 add OpenAPI schema, generated model, handler, and route.

**Placeholder scan:** No TBD/TODO/fill-in-later placeholders remain. Code steps include concrete snippets and exact commands.

**Type consistency:** Activity types use `TaskActivityRecordDTO` at repository boundary, `TaskActivityResponseDTO` at service/handler boundary, `datamodel.TaskActivity` for database rows, and `v1.TaskActivity` for HTTP output.