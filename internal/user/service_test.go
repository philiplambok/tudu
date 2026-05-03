package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/philiplambok/tudu/internal/user"
	"github.com/philiplambok/tudu/pkg/avatar"
)

// mockRepo satisfies user.Repository without a database.
type mockRepo struct {
	createFn             func(ctx context.Context, rec user.CreateUserRecordDTO) (*user.UserResponseDTO, error)
	findByEmailForAuthFn func(ctx context.Context, email string) (*user.AuthRecord, error)
	findByIDFn           func(ctx context.Context, id int64) (*user.UserResponseDTO, error)
}

func (m *mockRepo) Create(ctx context.Context, rec user.CreateUserRecordDTO) (*user.UserResponseDTO, error) {
	return m.createFn(ctx, rec)
}
func (m *mockRepo) FindByEmailForAuth(ctx context.Context, email string) (*user.AuthRecord, error) {
	return m.findByEmailForAuthFn(ctx, email)
}
func (m *mockRepo) FindByID(ctx context.Context, id int64) (*user.UserResponseDTO, error) {
	return m.findByIDFn(ctx, id)
}

func newTestService(repo user.Repository) user.Service {
	return user.NewService(repo, avatar.NewMock(), "test-secret")
}

func TestRegister_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, rec user.CreateUserRecordDTO) (*user.UserResponseDTO, error) {
			return &user.UserResponseDTO{
				ID:        1,
				Email:     rec.Email,
				AvatarURL: rec.AvatarURL,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}
	svc := newTestService(repo)
	resp, err := svc.Register(context.Background(), user.RegisterRequestDTO{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty JWT token")
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", resp.User.Email)
	}
}

func TestRegister_ValidationError_ShortPassword(t *testing.T) {
	svc := newTestService(&mockRepo{})
	_, err := svc.Register(context.Background(), user.RegisterRequestDTO{
		Email:    "alice@example.com",
		Password: "short",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *user.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected *user.ValidationError, got %T", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ user.CreateUserRecordDTO) (*user.UserResponseDTO, error) {
			return nil, user.ErrEmailConflict
		},
	}
	svc := newTestService(repo)
	_, err := svc.Register(context.Background(), user.RegisterRequestDTO{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if !errors.Is(err, user.ErrEmailConflict) {
		t.Errorf("expected ErrEmailConflict, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, rec user.CreateUserRecordDTO) (*user.UserResponseDTO, error) {
			return &user.UserResponseDTO{ID: 1, Email: rec.Email, AvatarURL: rec.AvatarURL}, nil
		},
		findByEmailForAuthFn: func(_ context.Context, email string) (*user.AuthRecord, error) {
			return &user.AuthRecord{
				ID:           1,
				Email:        email,
				PasswordHash: "$2a$10$i4sb3JfXsPn/W98kI2qZYuWjbN0mSPiulNey4i4DXX0VJfwRxyVYq",
				AvatarURL:    "https://example.com/avatar.png",
			}, nil
		},
	}
	svc := newTestService(repo)
	resp, err := svc.Login(context.Background(), user.LoginRequestDTO{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty JWT token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &mockRepo{
		findByEmailForAuthFn: func(_ context.Context, email string) (*user.AuthRecord, error) {
			return &user.AuthRecord{
				ID:           1,
				Email:        email,
				PasswordHash: "$2a$10$i4sb3JfXsPn/W98kI2qZYuWjbN0mSPiulNey4i4DXX0VJfwRxyVYq",
			}, nil
		},
	}
	svc := newTestService(repo)
	_, err := svc.Login(context.Background(), user.LoginRequestDTO{
		Email:    "alice@example.com",
		Password: "wrongpassword",
	})
	if !errors.Is(err, user.ErrInvalidCreds) {
		t.Errorf("expected ErrInvalidCreds, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockRepo{
		findByEmailForAuthFn: func(_ context.Context, _ string) (*user.AuthRecord, error) {
			return nil, user.ErrNotFound
		},
	}
	svc := newTestService(repo)
	_, err := svc.Login(context.Background(), user.LoginRequestDTO{
		Email:    "nobody@example.com",
		Password: "password123",
	})
	if !errors.Is(err, user.ErrInvalidCreds) {
		t.Errorf("expected ErrInvalidCreds (not ErrNotFound) to avoid leaking user existence, got %v", err)
	}
}
