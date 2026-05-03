package task_test

import (
	"testing"
	"time"

	"github.com/philiplambok/tudu/internal/task"
)

func TestNewTask(t *testing.T) {
	dueDate := time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC)

	aggregate := task.NewTask(task.CreateTaskRecordDTO{
		UserID:      42,
		Title:       "Write tests",
		Description: "Cover aggregate behavior",
		DueDate:     &dueDate,
	})

	if aggregate.ID != 0 {
		t.Fatalf("expected zero ID, got %d", aggregate.ID)
	}
	if aggregate.UserID != 42 {
		t.Fatalf("expected user ID 42, got %d", aggregate.UserID)
	}
	if aggregate.Title != "Write tests" {
		t.Fatalf("expected title Write tests, got %q", aggregate.Title)
	}
	if aggregate.Description != "Cover aggregate behavior" {
		t.Fatalf("expected description Cover aggregate behavior, got %q", aggregate.Description)
	}
	if aggregate.Status != task.StatusPending {
		t.Fatalf("expected pending status, got %q", aggregate.Status)
	}
	if aggregate.DueDate != &dueDate {
		t.Fatalf("expected due date pointer to be preserved")
	}
	if len(aggregate.Activities) != 1 {
		t.Fatalf("expected one activity, got %d", len(aggregate.Activities))
	}

	activity := aggregate.Activities[0]
	if activity.UserID != 42 {
		t.Fatalf("expected activity user ID 42, got %d", activity.UserID)
	}
	if activity.Action != task.ActivityActionCreated {
		t.Fatalf("expected created activity, got %q", activity.Action)
	}
	if activity.FieldName != nil || activity.OldValue != nil || activity.NewValue != nil {
		t.Fatalf("expected created activity values to be nil")
	}
}

func TestTaskFromRecord(t *testing.T) {
	dueDate := time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC)
	completedAt := time.Date(2026, 5, 5, 11, 30, 0, 0, time.UTC)
	createdAt := time.Date(2026, 5, 3, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	aggregate := task.TaskFromRecord(task.TaskRecordDTO{
		ID:          10,
		UserID:      42,
		Title:       "Write tests",
		Description: "Cover aggregate behavior",
		Status:      task.StatusCompleted,
		DueDate:     &dueDate,
		CompletedAt: &completedAt,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	})

	if aggregate.ID != 10 || aggregate.UserID != 42 {
		t.Fatalf("expected IDs 10/42, got %d/%d", aggregate.ID, aggregate.UserID)
	}
	if aggregate.Title != "Write tests" || aggregate.Description != "Cover aggregate behavior" {
		t.Fatalf("unexpected task text: %q / %q", aggregate.Title, aggregate.Description)
	}
	if aggregate.Status != task.StatusCompleted {
		t.Fatalf("expected completed status, got %q", aggregate.Status)
	}
	if aggregate.DueDate != &dueDate || aggregate.CompletedAt != &completedAt {
		t.Fatalf("expected time pointers to be preserved")
	}
	if !aggregate.CreatedAt.Equal(createdAt) || !aggregate.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected timestamps to be preserved")
	}
	if len(aggregate.Activities) != 0 {
		t.Fatalf("expected no pending activities, got %d", len(aggregate.Activities))
	}
}

func TestTaskApplyUpdateTitleChanged(t *testing.T) {
	before := task.Task{ID: 10, UserID: 42, Title: "Before", Description: "Description"}
	title := "After"

	after := before.ApplyUpdate(task.UpdateRequestDTO{Title: &title})

	if after.Title != "After" {
		t.Fatalf("expected title to be updated, got %q", after.Title)
	}
	assertActivity(t, after.Activities, "title", "Before", "After")
}

func TestTaskApplyUpdateTitleAndDescriptionChanged(t *testing.T) {
	before := task.Task{ID: 10, UserID: 42, Title: "Before", Description: "Old description"}
	title := "After"
	description := "New description"

	after := before.ApplyUpdate(task.UpdateRequestDTO{Title: &title, Description: &description})

	if after.Title != "After" || after.Description != "New description" {
		t.Fatalf("expected text fields to be updated")
	}
	if len(after.Activities) != 2 {
		t.Fatalf("expected two activities, got %d", len(after.Activities))
	}
	assertActivityAt(t, after.Activities[0], "title", "Before", "After")
	assertActivityAt(t, after.Activities[1], "description", "Old description", "New description")
}

func TestTaskApplyUpdateNoChanges(t *testing.T) {
	dueDate := time.Date(2026, 5, 4, 10, 30, 0, 123, time.UTC)
	before := task.Task{ID: 10, UserID: 42, Title: "Title", Description: "Description", DueDate: &dueDate}
	title := "Title"
	description := "Description"

	after := before.ApplyUpdate(task.UpdateRequestDTO{Title: &title, Description: &description, DueDate: &dueDate})

	if len(after.Activities) != 0 {
		t.Fatalf("expected no activities, got %d", len(after.Activities))
	}
}

func TestTaskApplyUpdateDueDateChanged(t *testing.T) {
	oldDueDate := time.Date(2026, 5, 4, 10, 30, 0, 123, time.UTC)
	newDueDate := time.Date(2026, 5, 5, 11, 30, 0, 456, time.UTC)
	before := task.Task{ID: 10, UserID: 42, Title: "Title", DueDate: &oldDueDate}

	after := before.ApplyUpdate(task.UpdateRequestDTO{DueDate: &newDueDate})

	if after.DueDate != &newDueDate {
		t.Fatalf("expected due date pointer to be updated")
	}
	assertActivity(t, after.Activities, "due_date", "2026-05-04T10:30:00.000000123Z", "2026-05-05T11:30:00.000000456Z")
}

func assertActivity(t *testing.T, activities []task.TaskActivity, fieldName string, oldValue string, newValue string) {
	t.Helper()
	if len(activities) != 1 {
		t.Fatalf("expected one activity, got %d", len(activities))
	}
	assertActivityAt(t, activities[0], fieldName, oldValue, newValue)
}

func assertActivityAt(t *testing.T, activity task.TaskActivity, fieldName string, oldValue string, newValue string) {
	t.Helper()
	if activity.Action != task.ActivityActionUpdated {
		t.Fatalf("expected updated activity, got %q", activity.Action)
	}
	if activity.TaskID != 10 {
		t.Fatalf("expected activity task ID 10, got %d", activity.TaskID)
	}
	if activity.UserID != 42 {
		t.Fatalf("expected activity user ID 42, got %d", activity.UserID)
	}
	if activity.FieldName == nil || *activity.FieldName != fieldName {
		t.Fatalf("expected field name %q, got %v", fieldName, activity.FieldName)
	}
	if activity.OldValue == nil || *activity.OldValue != oldValue {
		t.Fatalf("expected old value %q, got %v", oldValue, activity.OldValue)
	}
	if activity.NewValue == nil || *activity.NewValue != newValue {
		t.Fatalf("expected new value %q, got %v", newValue, activity.NewValue)
	}
}
