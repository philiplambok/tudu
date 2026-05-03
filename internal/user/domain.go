package user

import "errors"

var (
	ErrNotFound      = errors.New("user not found")
	ErrEmailConflict = errors.New("email already registered")
	ErrInvalidCreds  = errors.New("invalid email or password")
)

func ValidateRegister(req RegisterRequestDTO) error {
	if req.Email == "" {
		return &ValidationError{"email is required"}
	}
	if len(req.Password) < 8 {
		return &ValidationError{"password must be at least 8 characters"}
	}
	return nil
}

func ValidateLogin(req LoginRequestDTO) error {
	if req.Email == "" {
		return &ValidationError{"email is required"}
	}
	if req.Password == "" {
		return &ValidationError{"password is required"}
	}
	return nil
}
