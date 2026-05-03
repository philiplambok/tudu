package task

import (
	"errors"
	"time"
)

const (
	StatusPending   = "pending"
	StatusCompleted = "completed"

	ActivityActionCreated = "created"
	ActivityActionUpdated = "updated"
)

var ErrNotFound = errors.New("task not found")

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
	TaskID    int64
	UserID    int64
	Action    string
	FieldName *string
	OldValue  *string
	NewValue  *string
}

func TaskFromRecord(rec TaskRecordDTO) Task {
	return Task{
		ID:          rec.ID,
		UserID:      rec.UserID,
		Title:       rec.Title,
		Description: rec.Description,
		Status:      rec.Status,
		DueDate:     rec.DueDate,
		CompletedAt: rec.CompletedAt,
		CreatedAt:   rec.CreatedAt,
		UpdatedAt:   rec.UpdatedAt,
	}
}

func NewTask(rec CreateTaskRecordDTO) Task {
	return Task{
		UserID:      rec.UserID,
		Title:       rec.Title,
		Description: rec.Description,
		Status:      StatusPending,
		DueDate:     rec.DueDate,
		Activities: []TaskActivity{
			{UserID: rec.UserID, Action: ActivityActionCreated},
		},
	}
}

func (t Task) ApplyUpdate(req UpdateRequestDTO) Task {
	after := t
	after.Activities = nil

	if req.Title != nil && !valuesEqual(textValue(t.Title), textValue(*req.Title)) {
		field := "title"
		after.Activities = append(after.Activities, TaskActivity{
			TaskID:    t.ID,
			UserID:    t.UserID,
			Action:    ActivityActionUpdated,
			FieldName: &field,
			OldValue:  textValue(t.Title),
			NewValue:  textValue(*req.Title),
		})
		after.Title = *req.Title
	}

	if req.Description != nil && !valuesEqual(textValue(t.Description), textValue(*req.Description)) {
		field := "description"
		after.Activities = append(after.Activities, TaskActivity{
			TaskID:    t.ID,
			UserID:    t.UserID,
			Action:    ActivityActionUpdated,
			FieldName: &field,
			OldValue:  textValue(t.Description),
			NewValue:  textValue(*req.Description),
		})
		after.Description = *req.Description
	}

	if req.DueDate != nil && !valuesEqual(timeValue(t.DueDate), timeValue(req.DueDate)) {
		field := "due_date"
		after.Activities = append(after.Activities, TaskActivity{
			TaskID:    t.ID,
			UserID:    t.UserID,
			Action:    ActivityActionUpdated,
			FieldName: &field,
			OldValue:  timeValue(t.DueDate),
			NewValue:  timeValue(req.DueDate),
		})
		after.DueDate = req.DueDate
	}

	return after
}

func ValidateCreate(req CreateRequestDTO) error {
	if req.Title == "" {
		return &ValidationError{"title is required"}
	}
	return nil
}

func ValidateUpdate(req UpdateRequestDTO) error {
	if req.Title == nil && req.Description == nil && req.DueDate == nil {
		return &ValidationError{"at least one field is required"}
	}
	if req.Title != nil && *req.Title == "" {
		return &ValidationError{"title cannot be empty"}
	}
	return nil
}

func stringPtr(v string) *string {
	return &v
}

func timeValue(t *time.Time) *string {
	if t == nil {
		return nil
	}
	return stringPtr(t.UTC().Format(time.RFC3339Nano))
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
