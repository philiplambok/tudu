package datamodel

import "time"

// Task represents a row in the tasks table.
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
}
