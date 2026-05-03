# Pagination Design

## Goal

Add offset-based pagination to `GET /v1/tasks`. Callers control page and page size via query parameters; the response returns the task list alongside metadata describing total records and available pages.

## Motivation

The current `List` endpoint returns all tasks for the authenticated user in a single response. As the task count grows, this becomes a full-table scan with an unbounded payload. Pagination caps the response size, makes memory usage predictable, and aligns with the xm-backend pattern used across the wider platform.

---

## Shared Utility — `internal/common/util/pagination.go`

A new file in the existing `internal/common/util` package. Mirrors `xm-backend`'s `internal/common/util/pagination.go`.

### Constants

```go
const (
    DefaultPage  = 1
    DefaultLimit = 20
    MaxLimit     = 100
)
```

### Request type

```go
type PagingRequest struct {
    Page  int
    Limit int
}

func NewPagingRequest(r *http.Request) PagingRequest
func (p PagingRequest) Offset() int
```

`NewPagingRequest` reads `?page=` and `?limit=` from the request query string. Normalisation rules:

- Non-numeric or `<= 0` page → `DefaultPage` (1)
- Non-numeric or `<= 0` limit → `DefaultLimit` (20)
- limit `> MaxLimit` → `MaxLimit` (100)

`Offset()` returns `(page - 1) * limit`.

### Response type

```go
type PagingResponse[T any] struct {
    data      []T
    totalData int64
    page      int
    limit     int
}

func NewPagingResponse[T any](data []T, total int64, page, limit int) PagingResponse[T]

func (p PagingResponse[T]) Data() []T
func (p PagingResponse[T]) PageInfo() PageInfo
```

### Metadata type

```go
type PageInfo struct {
    Count        int  `json:"count"`
    CurrentPage  int  `json:"current_page"`
    NextPage     *int `json:"next_page,omitempty"`
    PreviousPage *int `json:"previous_page,omitempty"`
    TotalData    int  `json:"total_data"`
    TotalPage    int  `json:"total_page"`
}
```

`PageInfo()` derivations:

- `count` = `len(data)`
- `current_page` = `page`
- `total_page` = `ceil(totalData / limit)` (minimum 1 when totalData == 0)
- `next_page` = `current_page + 1` if `current_page < total_page`, else nil
- `previous_page` = `current_page - 1` if `current_page > 1`, else nil
- `total_data` = `totalData`

---

## OpenAPI Changes — `api/openapi.yml`

### New schemas

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
```

### Updated schema

```yaml
TaskListResponse:
  type: object
  required: [data]
  properties:
    data:
      $ref: "#/components/schemas/TaskListData"
```

### Updated endpoint

`GET /v1/tasks` gains two new query parameters:

```yaml
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

After regenerating the OpenAPI client, the generated types will include `TaskListData`, `TaskListResponse`, and `PageInfo`.

---

## Application Flow

### DTO — `internal/task/dto.go`

New params DTO for the repo boundary:

```go
type ListTaskRecordParams struct {
    UserID int64
    Status string
    util.PagingRequest
}
```

### Repository — `internal/task/repository.go`

Interface change:

```go
// Before
List(ctx context.Context, userID int64, status string) ([]TaskRecordDTO, error)

// After
List(ctx context.Context, params ListTaskRecordParams) ([]TaskRecordDTO, int64, error)
```

Implementation runs two queries inside the same db context (not a transaction):

1. Data query: applies filters + `LIMIT` + `OFFSET` + `ORDER BY created_at DESC`.
2. Count query: applies same filters + `COUNT(*)`.

```go
// data query
q.Order("created_at DESC").
    Limit(params.Limit).
    Offset(params.Offset()).
    Scan(&rows)

// count query
q.Count(&total)
```

### Service — `internal/task/service.go`

Interface change:

```go
// Before
List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error)

// After
List(ctx context.Context, userID int64, status string, paging util.PagingRequest) (util.PagingResponse[TaskResponseDTO], error)
```

Implementation:

```go
recs, total, err := s.repo.List(ctx, ListTaskRecordParams{
    UserID:        userID,
    Status:        status,
    PagingRequest: paging,
})
// map recs to []TaskResponseDTO
return util.NewPagingResponse(out, total, paging.Page, paging.Limit), nil
```

### Handler — `internal/task/handler.go`

```go
paging := util.NewPagingRequest(r)
result, err := h.svc.List(r.Context(), userID, status, paging)

render.JSON(w, r, v1.TaskListResponse{
    Data: v1.TaskListData{
        Tasks:    toV1Tasks(result.Data()),
        PageInfo: toV1PageInfo(result.PageInfo()),
    },
})
```

A small private helper `toV1PageInfo` converts `util.PageInfo` to the generated `v1.PageInfo`.

---

## Files Changed

| File | Change |
|------|--------|
| `internal/common/util/pagination.go` | New file: `PagingRequest`, `PagingResponse[T]`, `PageInfo`, `NewPagingRequest` |
| `internal/common/util/pagination_test.go` | New file: unit tests for utility |
| `api/openapi.yml` | Add `PageInfo`, `TaskListData` schemas; update `TaskListResponse`; add `page`/`limit` params |
| `pkg/openapi/v1/openapi.gen.go` | Regenerated |
| `internal/task/dto.go` | Add `ListTaskRecordParams` |
| `internal/task/repository.go` | Change `List` signature and implementation |
| `internal/task/service.go` | Change `List` signature and implementation |
| `internal/task/handler.go` | Parse paging from request; update response mapping |
| `internal/task/service_test.go` | Update `List` mock signature and assertions |

---

## Testing

### `pagination_test.go` (unit)

- `NewPagingRequest` with valid params → correct page/limit
- `NewPagingRequest` with `page=0` → defaults to 1
- `NewPagingRequest` with `limit=999` → caps at 100
- `Offset()` for page 1 → 0; page 2, limit 20 → 20
- `PageInfo()` with mid-page result → correct next/previous
- `PageInfo()` on first page → `previous_page` is nil
- `PageInfo()` on last page → `next_page` is nil
- `PageInfo()` with zero total → `total_page` is 1, both next/previous nil

### `service_test.go` (unit)

- `List` mock updated for new signature
- Verify service passes correct `PagingRequest` to repo
- Verify service returns `PagingResponse` with correct metadata

### Manual smoke test

```bash
GET /v1/tasks?page=1&limit=5
# → data.tasks has ≤ 5 items, data.page_info reflects total count

GET /v1/tasks?page=2&limit=5
# → data.tasks is next 5 items, previous_page = 1

GET /v1/tasks        # no params
# → uses defaults: page=1, limit=20
```

---

## Non-goals

- No cursor-based pagination.
- No changes to `GET /v1/tasks/{id}/activities` — that list is bounded by a single task's lifetime and does not need pagination now.
- No sorting controls — `created_at DESC` is the stable default.
- No changes to any other endpoint.
