package task_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/philiplambok/tudu/internal/task"
)

type mockTaskRepo struct {
	createFn         func(ctx context.Context, agg task.Task) (*task.TaskRecordDTO, error)
	listFn           func(ctx context.Context, userID int64, status string) ([]task.TaskRecordDTO, error)
	getFn            func(ctx context.Context, userID int64, id int64) (*task.TaskRecordDTO, error)
	updateFn         func(ctx context.Context, userID int64, agg task.Task) (*task.TaskRecordDTO, error)
	completeFn       func(ctx context.Context, userID int64, id int64) (*task.TaskRecordDTO, error)
	deleteFn         func(ctx context.Context, userID int64, id int64) error
	listActivitiesFn func(ctx context.Context, userID int64, taskID int64) ([]task.TaskActivityRecordDTO, error)
}

func (m *mockTaskRepo) Create(ctx context.Context, agg task.Task) (*task.TaskRecordDTO, error) {
	return m.createFn(ctx, agg)
}
func (m *mockTaskRepo) List(ctx context.Context, userID int64, status string) ([]task.TaskRecordDTO, error) {
	return m.listFn(ctx, userID, status)
}
func (m *mockTaskRepo) Get(ctx context.Context, userID int64, id int64) (*task.TaskRecordDTO, error) {
	return m.getFn(ctx, userID, id)
}
func (m *mockTaskRepo) Update(ctx context.Context, userID int64, agg task.Task) (*task.TaskRecordDTO, error) {
	return m.updateFn(ctx, userID, agg)
}
func (m *mockTaskRepo) Complete(ctx context.Context, userID int64, id int64) (*task.TaskRecordDTO, error) {
	return m.completeFn(ctx, userID, id)
}
func (m *mockTaskRepo) Delete(ctx context.Context, userID int64, id int64) error {
	return m.deleteFn(ctx, userID, id)
}
func (m *mockTaskRepo) ListActivities(ctx context.Context, userID int64, taskID int64) ([]task.TaskActivityRecordDTO, error) {
	return m.listActivitiesFn(ctx, userID, taskID)
}

func TestCreate_Success(t *testing.T) {
	repo := &mockTaskRepo{
		createFn: func(_ context.Context, agg task.Task) (*task.TaskRecordDTO, error) {
			if len(agg.Activities) != 1 || agg.Activities[0].Action != task.ActivityActionCreated {
				t.Errorf("expected one created activity, got %v", agg.Activities)
			}
			return &task.TaskRecordDTO{
				ID:     1,
				UserID: agg.UserID,
				Title:  agg.Title,
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
		getFn: func(_ context.Context, _ int64, _ int64) (*task.TaskRecordDTO, error) {
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
		completeFn: func(_ context.Context, userID int64, id int64) (*task.TaskRecordDTO, error) {
			return &task.TaskRecordDTO{
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

func TestUpdate_TitleChanged(t *testing.T) {
	newTitle := "Updated title"
	repo := &mockTaskRepo{
		getFn: func(_ context.Context, _ int64, id int64) (*task.TaskRecordDTO, error) {
			return &task.TaskRecordDTO{
				ID:     id,
				UserID: 1,
				Title:  "Old title",
				Status: task.StatusPending,
			}, nil
		},
		updateFn: func(_ context.Context, userID int64, agg task.Task) (*task.TaskRecordDTO, error) {
			if agg.Title != "Updated title" {
				t.Errorf("expected aggregate title 'Updated title', got %q", agg.Title)
			}
			if len(agg.Activities) != 1 {
				t.Fatalf("expected one activity, got %d", len(agg.Activities))
			}
			act := agg.Activities[0]
			if act.Action != task.ActivityActionUpdated {
				t.Errorf("expected updated action, got %q", act.Action)
			}
			if act.FieldName == nil || *act.FieldName != "title" {
				t.Errorf("expected field_name title, got %v", act.FieldName)
			}
			return &task.TaskRecordDTO{
				ID:     agg.ID,
				UserID: userID,
				Title:  agg.Title,
				Status: task.StatusPending,
			}, nil
		},
	}
	svc := task.NewService(repo)
	got, err := svc.Update(context.Background(), 1, 1, task.UpdateRequestDTO{Title: &newTitle})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Title != "Updated title" {
		t.Errorf("expected title 'Updated title', got %s", got.Title)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &mockTaskRepo{
		getFn: func(_ context.Context, _ int64, _ int64) (*task.TaskRecordDTO, error) {
			return nil, task.ErrNotFound
		},
	}
	title := "x"
	svc := task.NewService(repo)
	_, err := svc.Update(context.Background(), 1, 999, task.UpdateRequestDTO{Title: &title})
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

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
