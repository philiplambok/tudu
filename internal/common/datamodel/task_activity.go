package datamodel

import "time"

type TaskActivity struct {
	ID        int64
	TaskID    int64
	UserID    int64
	Action    string
	FieldName *string
	OldValue  *string
	NewValue  *string
	CreatedAt time.Time
}
