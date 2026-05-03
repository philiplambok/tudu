# Error Refactor Design

**Date:** 2026-05-03  
**Scope:** Small housekeeping refactor — no behaviour change

## Goal

Move `ValidationError` out of `dto.go` into a dedicated `error.go` per package, so `dto.go` contains only data shapes.

## Affected Packages

| Package | File to create | File to update |
|---|---|---|
| `internal/task` | `error.go` | `dto.go` |
| `internal/user` | `error.go` | `dto.go` |

## Changes

Each new `error.go` will contain exactly:

```go
package <pkg>

type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }
```

The same two lines are removed from each `dto.go`.

## Constraints

- Type name unchanged — no callers require updating
- No new imports or shared packages
- Each package stays fully self-contained
