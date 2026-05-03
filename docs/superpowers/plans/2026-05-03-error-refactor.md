# Error Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move package-local `ValidationError` definitions from `dto.go` into focused `error.go` files without changing behavior.

**Architecture:** Keep `ValidationError` package-local in both `internal/task` and `internal/user`. The new `error.go` files own error types; `dto.go` files own DTO/data shape definitions only.

**Tech Stack:** Go, existing package structure, existing test command `./dx/test ./...`.

---

## File Structure

- Create `internal/task/error.go` — owns `task.ValidationError`.
- Modify `internal/task/dto.go` — remove `ValidationError` and keep DTOs only.
- Create `internal/user/error.go` — owns `user.ValidationError`.
- Modify `internal/user/dto.go` — remove `ValidationError` and keep DTOs only.

---

### Task 1: Move task validation error

**Files:**
- Create: `internal/task/error.go`
- Modify: `internal/task/dto.go:5-7`

- [ ] **Step 1: Create `internal/task/error.go`**

```go
package task

type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }
```

- [ ] **Step 2: Remove `ValidationError` from `internal/task/dto.go`**

Remove these lines:

```go
type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }
```

After the change, the top of `internal/task/dto.go` should be:

```go
package task

import "time"

type CreateRequestDTO struct {
	Title       string
	Description string
	DueDate     *time.Time
}
```

- [ ] **Step 3: Format the task package files**

Run:

```bash
gofmt -w internal/task/error.go internal/task/dto.go
```

Expected: command exits with code 0 and no output.

---

### Task 2: Move user validation error

**Files:**
- Create: `internal/user/error.go`
- Modify: `internal/user/dto.go:5-7`

- [ ] **Step 1: Create `internal/user/error.go`**

```go
package user

type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }
```

- [ ] **Step 2: Remove `ValidationError` from `internal/user/dto.go`**

Remove these lines:

```go
type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }
```

After the change, the top of `internal/user/dto.go` should be:

```go
package user

import "time"

type RegisterRequestDTO struct {
	Email    string
	Password string
}
```

- [ ] **Step 3: Format the user package files**

Run:

```bash
gofmt -w internal/user/error.go internal/user/dto.go
```

Expected: command exits with code 0 and no output.

---

### Task 3: Verify no behavior changed

**Files:**
- Verify: `internal/task/error.go`
- Verify: `internal/user/error.go`
- Verify: `internal/task/dto.go`
- Verify: `internal/user/dto.go`

- [ ] **Step 1: Run the full test suite**

Run:

```bash
./dx/test ./...
```

Expected: all packages pass.

- [ ] **Step 2: Confirm validation error definitions moved**

Run:

```bash
grep -R "type ValidationError" internal/task internal/user
```

Expected output contains exactly:

```text
internal/task/error.go:type ValidationError struct{ msg string }
internal/user/error.go:type ValidationError struct{ msg string }
```

- [ ] **Step 3: Check git diff**

Run:

```bash
git diff -- internal/task/error.go internal/task/dto.go internal/user/error.go internal/user/dto.go
```

Expected: diff only creates `error.go` files and removes `ValidationError` from the two `dto.go` files.
