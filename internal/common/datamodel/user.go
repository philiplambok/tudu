package datamodel

import "time"

// User represents a row in the users table.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	AvatarURL    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
