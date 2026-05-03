package task

import "time"

type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }

type CreateRequestDTO struct {
	Title       string
	Description string
	DueDate     *time.Time
}

type CreateTaskRecordDTO struct {
	UserID      int64
	Title       string
	Description string
	DueDate     *time.Time
}

type UpdateRequestDTO struct {
	Title       *string
	Description *string
	DueDate     *time.Time
}


type TaskRecordDTO struct {
	ID          int64
	UserID      int64
	Title       string
	Description string
	Status      string
	DueDate     *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TaskResponseDTO struct {
	ID          int64
	UserID      int64
	Title       string
	Description string
	Status      string
	DueDate     *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

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
