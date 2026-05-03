package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, rec CreateUserRecordDTO) (*UserResponseDTO, error)
	FindByEmailForAuth(ctx context.Context, email string) (*AuthRecord, error)
	FindByID(ctx context.Context, id int64) (*UserResponseDTO, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, rec CreateUserRecordDTO) (*UserResponseDTO, error) {
	var row UserResponseDTO
	res := r.db.WithContext(ctx).Raw(`
		INSERT INTO users (email, password_hash, avatar_url, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
		RETURNING id, email, avatar_url, created_at, updated_at`,
		rec.Email, rec.PasswordHash, rec.AvatarURL,
	).Scan(&row)
	if res.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(res.Error, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailConflict
		}
		return nil, res.Error
	}
	return &row, nil
}

func (r *repository) FindByEmailForAuth(ctx context.Context, email string) (*AuthRecord, error) {
	var row AuthRecord
	res := r.db.WithContext(ctx).Raw(`
		SELECT id, email, password_hash, avatar_url, created_at, updated_at
		FROM users WHERE email = ?`, email,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &row, nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*UserResponseDTO, error) {
	var row UserResponseDTO
	res := r.db.WithContext(ctx).Raw(`
		SELECT id, email, avatar_url, created_at, updated_at
		FROM users WHERE id = ?`, id,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &row, nil
}
