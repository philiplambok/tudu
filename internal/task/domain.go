package task

import "errors"

const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
)

var ErrNotFound = errors.New("task not found")

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
