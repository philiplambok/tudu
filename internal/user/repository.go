package user

import (
	"context"
	"errors"

	"github.com/philiplambok/tudu/internal/common/datamodel"
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
	var row datamodel.User
	res := r.db.WithContext(ctx).Raw(`
		INSERT INTO users (email, password_hash, created_at, updated_at)
		VALUES (?, ?, NOW(), NOW())
		RETURNING id, email, password_hash, created_at, updated_at`,
		rec.Email, rec.PasswordHash,
	).Scan(&row)
	if res.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(res.Error, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailConflict
		}
		return nil, res.Error
	}
	return toUserResponseDTO(&row), nil
}

func (r *repository) FindByEmailForAuth(ctx context.Context, email string) (*AuthRecord, error) {
	var row datamodel.User
	res := r.db.WithContext(ctx).Raw(`
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE email = ?`, email,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return toAuthRecord(&row), nil
}

func (r *repository) FindByID(ctx context.Context, id int64) (*UserResponseDTO, error) {
	var row datamodel.User
	res := r.db.WithContext(ctx).Raw(`
		SELECT id, email, password_hash, created_at, updated_at
		FROM users WHERE id = ?`, id,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return toUserResponseDTO(&row), nil
}

func toUserResponseDTO(m *datamodel.User) *UserResponseDTO {
	return &UserResponseDTO{
		ID:        m.ID,
		Email:     m.Email,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func toAuthRecord(m *datamodel.User) *AuthRecord {
	return &AuthRecord{
		ID:           m.ID,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
