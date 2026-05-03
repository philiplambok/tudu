# Task Pagination Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add offset-based pagination to `GET /v1/tasks` with nested `{ data: { tasks, page_info } }` responses.

**Architecture:** Follow the approved design in `docs/superpowers/specs/2026-05-03-pagination-design.md`. Add a reusable `internal/common/util` pagination helper, update OpenAPI/generated models, then thread `util.PagingRequest` through handler → service → repository. Repository returns page rows plus total count; service wraps rows in `util.PagingResponse[T]`; handler maps it to the generated v1 response.

**Tech Stack:** Go 1.26, chi, GORM, PostgreSQL, oapi-codegen, existing OpenAPI models.

---

## File Structure

- Create `internal/common/util/pagination.go`
  - Owns pagination request parsing, offset calculation, response metadata, and generic response wrapper.
- Create `internal/common/util/pagination_test.go`
  - Unit tests for defaults, normalization, offset, and page info.
- Modify `api/openapi.yml`
  - Add `page`/`limit` query params to `GET /v1/tasks`.
  - Add `PageInfo` and `TaskListData` schemas.
  - Change `TaskListResponse.data` from array to nested object.
- Regenerate `pkg/openapi/v1/openapi.gen.go`
  - Use `go tool oapi-codegen -config ./oapi_codegen.yml ./api/openapi.yml`.
- Modify `internal/task/dto.go`
  - Add `ListTaskRecordParams` with embedded `util.PagingRequest`.
- Modify `internal/task/repository.go`
  - Change `Repository.List` signature to return rows and total count.
  - Add `LIMIT`, `OFFSET`, and `COUNT(*)` queries.
- Modify `internal/task/service.go`
  - Change `Service.List` signature to accept `util.PagingRequest` and return `util.PagingResponse[TaskResponseDTO]`.
- Modify `internal/task/handler.go`
  - Parse pagination from request.
  - Return nested `TaskListResponse` with `tasks` and `page_info`.
- Modify `internal/task/service_test.go`
  - Update mock repository signature and add assertions for paging propagation/metadata.

---

### Task 1: Add reusable pagination utility

**Files:**
- Create: `internal/common/util/pagination.go`
- Create: `internal/common/util/pagination_test.go`

- [ ] **Step 1: Write failing tests for pagination behavior**

Create `internal/common/util/pagination_test.go`:

```go
package util_test

import (
	"net/http/httptest"
	"testing"

	"github.com/philiplambok/tudu/internal/common/util"
)

func TestNewPagingRequestDefaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/tasks", nil)

	paging := util.NewPagingRequest(req)

	if paging.Page != 1 {
		t.Fatalf("expected page 1, got %d", paging.Page)
	}
	if paging.Limit != 20 {
		t.Fatalf("expected limit 20, got %d", paging.Limit)
	}
}

func TestNewPagingRequestValidParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/tasks?page=2&limit=5", nil)

	paging := util.NewPagingRequest(req)

	if paging.Page != 2 {
		t.Fatalf("expected page 2, got %d", paging.Page)
	}
	if paging.Limit != 5 {
		t.Fatalf("expected limit 5, got %d", paging.Limit)
	}
}

func TestNewPagingRequestNormalizesInvalidParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/tasks?page=abc&limit=-1", nil)

	paging := util.NewPagingRequest(req)

	if paging.Page != 1 {
		t.Fatalf("expected page 1, got %d", paging.Page)
	}
	if paging.Limit != 20 {
		t.Fatalf("expected limit 20, got %d", paging.Limit)
	}
}

func TestNewPagingRequestCapsLimit(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/tasks?page=1&limit=999", nil)

	paging := util.NewPagingRequest(req)

	if paging.Limit != 100 {
		t.Fatalf("expected limit 100, got %d", paging.Limit)
	}
}

func TestPagingRequestOffset(t *testing.T) {
	paging := util.PagingRequest{Page: 2, Limit: 20}

	if paging.Offset() != 20 {
		t.Fatalf("expected offset 20, got %d", paging.Offset())
	}
}

func TestPagingResponsePageInfoMiddlePage(t *testing.T) {
	response := util.NewPagingResponse([]int{1, 2, 3, 4, 5}, 12, 2, 5)

	info := response.PageInfo()

	if info.Count != 5 {
		t.Fatalf("expected count 5, got %d", info.Count)
	}
	if info.CurrentPage != 2 {
		t.Fatalf("expected current page 2, got %d", info.CurrentPage)
	}
	if info.TotalData != 12 {
		t.Fatalf("expected total data 12, got %d", info.TotalData)
	}
	if info.TotalPage != 3 {
		t.Fatalf("expected total page 3, got %d", info.TotalPage)
	}
	if info.PreviousPage == nil || *info.PreviousPage != 1 {
		t.Fatalf("expected previous page 1, got %v", info.PreviousPage)
	}
	if info.NextPage == nil || *info.NextPage != 3 {
		t.Fatalf("expected next page 3, got %v", info.NextPage)
	}
}

func TestPagingResponsePageInfoFirstPage(t *testing.T) {
	response := util.NewPagingResponse([]int{1, 2, 3}, 8, 1, 3)

	info := response.PageInfo()

	if info.PreviousPage != nil {
		t.Fatalf("expected previous page nil, got %v", info.PreviousPage)
	}
	if info.NextPage == nil || *info.NextPage != 2 {
		t.Fatalf("expected next page 2, got %v", info.NextPage)
	}
}

func TestPagingResponsePageInfoLastPage(t *testing.T) {
	response := util.NewPagingResponse([]int{1, 2}, 8, 3, 3)

	info := response.PageInfo()

	if info.NextPage != nil {
		t.Fatalf("expected next page nil, got %v", info.NextPage)
	}
	if info.PreviousPage == nil || *info.PreviousPage != 2 {
		t.Fatalf("expected previous page 2, got %v", info.PreviousPage)
	}
}

func TestPagingResponsePageInfoZeroTotal(t *testing.T) {
	response := util.NewPagingResponse([]int{}, 0, 1, 20)

	info := response.PageInfo()

	if info.TotalPage != 1 {
		t.Fatalf("expected total page 1, got %d", info.TotalPage)
	}
	if info.NextPage != nil {
		t.Fatalf("expected next page nil, got %v", info.NextPage)
	}
	if info.PreviousPage != nil {
		t.Fatalf("expected previous page nil, got %v", info.PreviousPage)
	}
}
```

- [ ] **Step 2: Run tests and verify they fail because the package/file does not exist**

Run:

```bash
go test ./internal/common/util -v
```

Expected: FAIL with an error like `package github.com/philiplambok/tudu/internal/common/util is not in std` or undefined `util.NewPagingRequest` if the package directory already exists.

- [ ] **Step 3: Implement pagination utility**

Create `internal/common/util/pagination.go`:

```go
package util

import (
	"math"
	"net/http"
	"strconv"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type PagingRequest struct {
	Page  int
	Limit int
}

func NewPagingRequest(r *http.Request) PagingRequest {
	page := parsePositiveInt(r.URL.Query().Get("page"), DefaultPage)
	limit := parsePositiveInt(r.URL.Query().Get("limit"), DefaultLimit)
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return PagingRequest{Page: page, Limit: limit}
}

func (p PagingRequest) Offset() int {
	return (p.Page - 1) * p.Limit
}

type PageInfo struct {
	Count        int  `json:"count"`
	CurrentPage  int  `json:"current_page"`
	NextPage     *int `json:"next_page,omitempty"`
	PreviousPage *int `json:"previous_page,omitempty"`
	TotalData    int  `json:"total_data"`
	TotalPage    int  `json:"total_page"`
}

type PagingResponse[T any] struct {
	data      []T
	totalData int64
	page      int
	limit     int
}

func NewPagingResponse[T any](data []T, total int64, page int, limit int) PagingResponse[T] {
	return PagingResponse[T]{
		data:      data,
		totalData: total,
		page:      page,
		limit:     limit,
	}
}

func (p PagingResponse[T]) Data() []T {
	return p.data
}

func (p PagingResponse[T]) PageInfo() PageInfo {
	totalPage := 1
	if p.totalData > 0 {
		totalPage = int(math.Ceil(float64(p.totalData) / float64(p.limit)))
	}

	var previousPage *int
	if p.page > 1 {
		previousPage = intPtr(p.page - 1)
	}

	var nextPage *int
	if p.page < totalPage {
		nextPage = intPtr(p.page + 1)
	}

	return PageInfo{
		Count:        len(p.data),
		CurrentPage:  p.page,
		NextPage:     nextPage,
		PreviousPage: previousPage,
		TotalData:    int(p.totalData),
		TotalPage:    totalPage,
	}
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func intPtr(v int) *int {
	return &v
}
```

- [ ] **Step 4: Run utility tests and verify they pass**

Run:

```bash
go test ./internal/common/util -v
```

Expected: PASS.

- [ ] **Step 5: Commit utility**

Use the commit-message skill rules: branch prefix is `main`, no `Co-Authored-By`.

```bash
git add internal/common/util/pagination.go internal/common/util/pagination_test.go
git commit -m "main add pagination utility"
```

---

### Task 2: Update OpenAPI list task response shape

**Files:**
- Modify: `api/openapi.yml`
- Regenerate: `pkg/openapi/v1/openapi.gen.go`

- [ ] **Step 1: Update `GET /v1/tasks` query parameters in OpenAPI**

In `api/openapi.yml`, under `paths./v1/tasks.get.parameters`, keep `status` and add `page`/`limit` after it:

```yaml
      parameters:
        - name: status
          in: query
          schema:
            type: string
            enum: [pending, completed]
        - name: page
          in: query
          schema:
            type: integer
            minimum: 1
            default: 1
        - name: limit
          in: query
          schema:
            type: integer
            minimum: 1
            maximum: 100
            default: 20
```

- [ ] **Step 2: Update task list schemas in OpenAPI**

In `api/openapi.yml`, replace the current `TaskListResponse` schema:

```yaml
    TaskListResponse:
      type: object
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: "#/components/schemas/Task"
```

with:

```yaml
    PageInfo:
      type: object
      required: [count, current_page, total_data, total_page]
      properties:
        count:
          type: integer
        current_page:
          type: integer
        next_page:
          type: integer
          nullable: true
        previous_page:
          type: integer
          nullable: true
        total_data:
          type: integer
        total_page:
          type: integer

    TaskListData:
      type: object
      required: [tasks, page_info]
      properties:
        tasks:
          type: array
          items:
            $ref: "#/components/schemas/Task"
        page_info:
          $ref: "#/components/schemas/PageInfo"

    TaskListResponse:
      type: object
      required: [data]
      properties:
        data:
          $ref: "#/components/schemas/TaskListData"
```

- [ ] **Step 3: Regenerate OpenAPI models**

Run:

```bash
go tool oapi-codegen -config ./oapi_codegen.yml ./api/openapi.yml
```

Expected: command exits 0 and updates `pkg/openapi/v1/openapi.gen.go`.

- [ ] **Step 4: Verify generated types exist**

Run:

```bash
grep -n "type PageInfo\|type TaskListData\|type TaskListResponse" pkg/openapi/v1/openapi.gen.go
```

Expected output includes all three type declarations.

- [ ] **Step 5: Commit OpenAPI changes**

```bash
git add api/openapi.yml pkg/openapi/v1/openapi.gen.go
git commit -m "main update task list pagination schema"
```

---

### Task 3: Thread paging through repository and service

**Files:**
- Modify: `internal/task/dto.go`
- Modify: `internal/task/repository.go`
- Modify: `internal/task/service.go`
- Modify: `internal/task/service_test.go`

- [ ] **Step 1: Add repository params DTO**

In `internal/task/dto.go`, change the import from:

```go
import "time"
```

to:

```go
import (
	"time"

	"github.com/philiplambok/tudu/internal/common/util"
)
```

Then add this type after `CreateTaskRecordDTO`:

```go
type ListTaskRecordParams struct {
	UserID int64
	Status string
	util.PagingRequest
}
```

- [ ] **Step 2: Update repository interface and implementation**

In `internal/task/repository.go`, change the `Repository` interface method from:

```go
List(ctx context.Context, userID int64, status string) ([]TaskRecordDTO, error)
```

to:

```go
List(ctx context.Context, params ListTaskRecordParams) ([]TaskRecordDTO, int64, error)
```

Replace `func (r *repository) List(...)` with:

```go
func (r *repository) List(ctx context.Context, params ListTaskRecordParams) ([]TaskRecordDTO, int64, error) {
	base := r.db.WithContext(ctx).
		Table("tasks").
		Where("user_id = ?", params.UserID)

	if params.Status != "" {
		base = base.Where("status = ?", params.Status)
	}

	var rows []datamodel.Task
	if err := base.
		Select("id, user_id, title, description, status, due_date, completed_at, created_at, updated_at").
		Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	out := make([]TaskRecordDTO, len(rows))
	for i := range rows {
		out[i] = *toTaskRecordDTO(&rows[i])
	}
	return out, total, nil
}
```

- [ ] **Step 3: Update service interface and implementation**

In `internal/task/service.go`, add util import. Change:

```go
import "context"
```

to:

```go
import (
	"context"

	"github.com/philiplambok/tudu/internal/common/util"
)
```

Change service interface method from:

```go
List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error)
```

to:

```go
List(ctx context.Context, userID int64, status string, paging util.PagingRequest) (util.PagingResponse[TaskResponseDTO], error)
```

Replace `func (s *service) List(...)` with:

```go
func (s *service) List(ctx context.Context, userID int64, status string, paging util.PagingRequest) (util.PagingResponse[TaskResponseDTO], error) {
	recs, total, err := s.repo.List(ctx, ListTaskRecordParams{
		UserID:        userID,
		Status:        status,
		PagingRequest: paging,
	})
	if err != nil {
		return util.PagingResponse[TaskResponseDTO]{}, err
	}
	out := make([]TaskResponseDTO, len(recs))
	for i := range recs {
		out[i] = *toResponseDTO(&recs[i])
	}
	return util.NewPagingResponse(out, total, paging.Page, paging.Limit), nil
}
```

- [ ] **Step 4: Update service tests for the new List signature**

In `internal/task/service_test.go`, add the util import:

```go
"github.com/philiplambok/tudu/internal/common/util"
```

Change the mock field:

```go
listFn func(ctx context.Context, userID int64, status string) ([]task.TaskRecordDTO, error)
```

to:

```go
listFn func(ctx context.Context, params task.ListTaskRecordParams) ([]task.TaskRecordDTO, int64, error)
```

Change the mock method:

```go
func (m *mockTaskRepo) List(ctx context.Context, userID int64, status string) ([]task.TaskRecordDTO, error) {
	return m.listFn(ctx, userID, status)
}
```

to:

```go
func (m *mockTaskRepo) List(ctx context.Context, params task.ListTaskRecordParams) ([]task.TaskRecordDTO, int64, error) {
	return m.listFn(ctx, params)
}
```

Add this test after `TestCreate_EmptyTitle`:

```go
func TestList_WithPagination(t *testing.T) {
	repo := &mockTaskRepo{
		listFn: func(_ context.Context, params task.ListTaskRecordParams) ([]task.TaskRecordDTO, int64, error) {
			if params.UserID != 1 {
				t.Fatalf("expected userID 1, got %d", params.UserID)
			}
			if params.Status != task.StatusPending {
				t.Fatalf("expected status pending, got %q", params.Status)
			}
			if params.Page != 2 {
				t.Fatalf("expected page 2, got %d", params.Page)
			}
			if params.Limit != 5 {
				t.Fatalf("expected limit 5, got %d", params.Limit)
			}
			return []task.TaskRecordDTO{
				{ID: 6, UserID: 1, Title: "Task 6", Status: task.StatusPending},
				{ID: 7, UserID: 1, Title: "Task 7", Status: task.StatusPending},
			}, 12, nil
		},
	}
	svc := task.NewService(repo)

	got, err := svc.List(context.Background(), 1, task.StatusPending, util.PagingRequest{Page: 2, Limit: 5})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got.Data()) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got.Data()))
	}
	info := got.PageInfo()
	if info.TotalData != 12 {
		t.Fatalf("expected total data 12, got %d", info.TotalData)
	}
	if info.TotalPage != 3 {
		t.Fatalf("expected total page 3, got %d", info.TotalPage)
	}
	if info.NextPage == nil || *info.NextPage != 3 {
		t.Fatalf("expected next page 3, got %v", info.NextPage)
	}
	if info.PreviousPage == nil || *info.PreviousPage != 1 {
		t.Fatalf("expected previous page 1, got %v", info.PreviousPage)
	}
}
```

- [ ] **Step 5: Run task service tests and verify they pass**

Run:

```bash
go test ./internal/task -run 'TestList_WithPagination|TestCreate|TestUpdate|TestListActivities' -v
```

Expected: PASS.

- [ ] **Step 6: Commit service/repository changes**

```bash
git add internal/task/dto.go internal/task/repository.go internal/task/service.go internal/task/service_test.go
git commit -m "main thread pagination through task list flow"
```

---

### Task 4: Update handler response mapping

**Files:**
- Modify: `internal/task/handler.go`

- [ ] **Step 1: Update handler imports**

In `internal/task/handler.go`, add util import:

```go
"github.com/philiplambok/tudu/internal/common/util"
```

The import block should contain:

```go
import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/philiplambok/tudu/internal"
	"github.com/philiplambok/tudu/internal/common/util"
	v1 "github.com/philiplambok/tudu/pkg/openapi/v1"
)
```

- [ ] **Step 2: Replace `Handler.List` implementation**

Replace the existing `List` method with:

```go
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())
	status := r.URL.Query().Get("status")
	paging := util.NewPagingRequest(r)

	result, err := h.svc.List(r.Context(), userID, status, paging)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	data := result.Data()
	tasks := make([]v1.Task, len(data))
	for i := range data {
		tasks[i] = toV1Task(&data[i])
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskListResponse{
		Data: v1.TaskListData{
			Tasks:    tasks,
			PageInfo: toV1PageInfo(result.PageInfo()),
		},
	})
}
```

- [ ] **Step 3: Add `toV1PageInfo` helper**

Add this helper near `toV1TaskActivity`:

```go
func toV1PageInfo(info util.PageInfo) v1.PageInfo {
	return v1.PageInfo{
		Count:        info.Count,
		CurrentPage:  info.CurrentPage,
		NextPage:     info.NextPage,
		PreviousPage: info.PreviousPage,
		TotalData:    info.TotalData,
		TotalPage:    info.TotalPage,
	}
}
```

- [ ] **Step 4: Run gofmt and full tests**

Run:

```bash
gofmt -w internal/common/util/pagination.go internal/common/util/pagination_test.go internal/task/dto.go internal/task/repository.go internal/task/service.go internal/task/service_test.go internal/task/handler.go
go test ./...
go build ./...
```

Expected: all commands pass.

- [ ] **Step 5: Commit handler mapping**

```bash
git add internal/task/handler.go
git commit -m "main return paginated task list response"
```

---

### Task 5: Manual smoke test paginated endpoint

**Files:**
- No source changes expected unless smoke testing finds a bug.

- [ ] **Step 1: Start or rebuild the dev app if needed**

If the Docker dev app is already serving the latest code, skip to Step 2. If not, run:

```bash
docker compose -f docker-compose.dev.yml exec -d app go run -a . serve
```

Expected: app server starts on the configured local port.

- [ ] **Step 2: Run migrations inside Docker app container**

Run:

```bash
docker compose -f docker-compose.dev.yml exec -T app go run . migrate
```

Expected: migrations complete successfully.

- [ ] **Step 3: Register or login and capture token**

Use the same smoke-test auth flow used elsewhere in the project. Example using unique email:

```bash
EMAIL="pagination-$(date +%s)@example.com"
TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"password123\"}" | jq -r '.token')
```

Expected: `TOKEN` is not empty and not `null`.

- [ ] **Step 4: Create at least seven tasks**

Run:

```bash
for i in 1 2 3 4 5 6 7; do
  curl -s -X POST http://localhost:8080/v1/tasks \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"title\":\"pagination task $i\"}" >/dev/null
done
```

Expected: seven tasks are created.

- [ ] **Step 5: Verify page 1 response shape and metadata**

Run:

```bash
curl -s "http://localhost:8080/v1/tasks?page=1&limit=5" \
  -H "Authorization: Bearer $TOKEN" | jq '{count: (.data.tasks | length), page_info: .data.page_info}'
```

Expected:

```json
{
  "count": 5,
  "page_info": {
    "count": 5,
    "current_page": 1,
    "next_page": 2,
    "previous_page": null,
    "total_data": 7,
    "total_page": 2
  }
}
```

- [ ] **Step 6: Verify page 2 response shape and metadata**

Run:

```bash
curl -s "http://localhost:8080/v1/tasks?page=2&limit=5" \
  -H "Authorization: Bearer $TOKEN" | jq '{count: (.data.tasks | length), page_info: .data.page_info}'
```

Expected:

```json
{
  "count": 2,
  "page_info": {
    "count": 2,
    "current_page": 2,
    "next_page": null,
    "previous_page": 1,
    "total_data": 7,
    "total_page": 2
  }
}
```

- [ ] **Step 7: Verify defaults and limit cap**

Run:

```bash
curl -s "http://localhost:8080/v1/tasks?page=abc&limit=999" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.page_info'
```

Expected:

```json
{
  "count": 7,
  "current_page": 1,
  "next_page": null,
  "previous_page": null,
  "total_data": 7,
  "total_page": 1
}
```

- [ ] **Step 8: Commit smoke-test fixes if any**

If smoke testing required source changes, commit only those files:

```bash
git add <changed-files>
git commit -m "main fix task pagination smoke test issue"
```

If no source changes were needed, do not create an empty commit.

---

## Self-Review

- Spec coverage: utility, OpenAPI, DTO/repository/service/handler flow, tests, and smoke tests are all covered.
- Placeholder scan: no TBD/TODO/fill-in placeholders are present; each code-changing step includes exact code.
- Type consistency: `util.PagingRequest`, `util.PagingResponse[T]`, `util.PageInfo`, `task.ListTaskRecordParams`, generated `v1.PageInfo`, and generated `v1.TaskListData` are used consistently across tasks.
