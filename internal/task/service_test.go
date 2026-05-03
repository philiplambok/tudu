package task_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/philiplambok/tudu/internal/task"
)

type mockTaskRepo struct {
	createFn   func(ctx context.Context, rec task.CreateTaskRecordDTO) (*task.TaskResponseDTO, error)
	listFn     func(ctx context.Context, userID int64, status string) ([]task.TaskResponseDTO, error)
	getFn      func(ctx context.Context, userID int64, id int64) (*task.TaskResponseDTO, error)
	updateFn   func(ctx context.Context, userID int64, id int64, rec task.UpdateTaskRecordDTO) (*task.TaskResponseDTO, error)
	completeFn func(ctx context.Context, userID int64, id int64) (*task.TaskResponseDTO, error)
	deleteFn   func(ctx context.Context, userID int64, id int64) error
}

func (m *mockTaskRepo) Create(ctx context.Context, rec task.CreateTaskRecordDTO) (*task.TaskResponseDTO, error) {
	return m.createFn(ctx, rec)
}
func (m *mockTaskRepo) List(ctx context.Context, userID int64, status string) ([]task.TaskResponseDTO, error) {
	return m.listFn(ctx, userID, status)
}
func (m *mockTaskRepo) Get(ctx context.Context, userID int64, id int64) (*task.TaskResponseDTO, error) {
	return m.getFn(ctx, userID, id)
}
func (m *mockTaskRepo) Update(ctx context.Context, userID int64, id int64, rec task.UpdateTaskRecordDTO) (*task.TaskResponseDTO, error) {
	return m.updateFn(ctx, userID, id, rec)
}
func (m *mockTaskRepo) Complete(ctx context.Context, userID int64, id int64) (*task.TaskResponseDTO, error) {
	return m.completeFn(ctx, userID, id)
}
func (m *mockTaskRepo) Delete(ctx context.Context, userID int64, id int64) error {
	return m.deleteFn(ctx, userID, id)
}

func TestCreate_Success(t *testing.T) {
	repo := &mockTaskRepo{
		createFn: func(_ context.Context, rec task.CreateTaskRecordDTO) (*task.TaskResponseDTO, error) {
			return &task.TaskResponseDTO{
				ID:     1,
				UserID: rec.UserID,
				Title:  rec.Title,
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
