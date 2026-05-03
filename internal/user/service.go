package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, req RegisterRequestDTO) (*AuthResponseDTO, error)
	Login(ctx context.Context, req LoginRequestDTO) (*AuthResponseDTO, error)
	Me(ctx context.Context, userID int64) (*UserResponseDTO, error)
}

type service struct {
	repo      Repository
	jwtSecret string
}

func NewService(repo Repository, jwtSecret string) Service {
	return &service{repo: repo, jwtSecret: jwtSecret}
}

func (s *service) Register(ctx context.Context, req RegisterRequestDTO) (*AuthResponseDTO, error) {
	if err := ValidateRegister(req); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.Create(ctx, CreateUserRecordDTO{
		Email:        req.Email,
		PasswordHash: string(hash),
	})
	if err != nil {
		return nil, err
	}

	token, err := s.generateToken(u.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponseDTO{Token: token, User: *u}, nil
}

func (s *service) Login(ctx context.Context, req LoginRequestDTO) (*AuthResponseDTO, error) {
	if err := ValidateLogin(req); err != nil {
		return nil, err
	}

	rec, err := s.repo.FindByEmailForAuth(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidCreds
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(rec.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCreds
	}

	token, err := s.generateToken(rec.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponseDTO{
		Token: token,
		User: UserResponseDTO{
			ID:        rec.ID,
			Email:     rec.Email,
			CreatedAt: rec.CreatedAt,
			UpdatedAt: rec.UpdatedAt,
		},
	}, nil
}

func (s *service) Me(ctx context.Context, userID int64) (*UserResponseDTO, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *service) generateToken(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
