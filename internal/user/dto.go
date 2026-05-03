package user

import "time"

type RegisterRequestDTO struct {
	Email    string
	Password string
}

type LoginRequestDTO struct {
	Email    string
	Password string
}

type UserResponseDTO struct {
	ID        int64
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AuthResponseDTO struct {
	Token string
	User  UserResponseDTO
}

type CreateUserRecordDTO struct {
	Email        string
	PasswordHash string
}

// AuthRecord carries the password hash alongside user fields for login verification.
type AuthRecord struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
