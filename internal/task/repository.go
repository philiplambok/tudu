package task

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, rec CreateTaskRecordDTO) (*TaskResponseDTO, error)
	List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error)
	Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error)
	Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Delete(ctx context.Context, userID int64, id int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, rec CreateTaskRecordDTO) (*TaskResponseDTO, error) {
	var row TaskResponseDTO
	res := r.db.WithContext(ctx).Raw(`
		INSERT INTO tasks (user_id, title, description, status, due_date, created_at, updated_at)
		VALUES (?, ?, ?, 'pending', ?, NOW(), NOW())
		RETURNING id, user_id, title, description, status, due_date, completed_at, created_at, updated_at`,
		rec.UserID, rec.Title, rec.Description, rec.DueDate,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	return &row, nil
}

func (r *repository) List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error) {
	q := r.db.WithContext(ctx).
		Select("id, user_id, title, description, status, due_date, completed_at, created_at, updated_at").
		Table("tasks").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if status != "" {
		q = q.Where("status = ?", status)
	}

	var rows []TaskResponseDTO
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []TaskResponseDTO{}
	}
	return rows, nil
}

func (r *repository) Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	var row TaskResponseDTO
	res := r.db.WithContext(ctx).Raw(`
		SELECT id, user_id, title, description, status, due_date, completed_at, created_at, updated_at
		FROM tasks WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &row, nil
}

func (r *repository) Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error) {
	updates := map[string]any{"updated_at": gorm.Expr("NOW()")}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DueDate != nil {
		updates["due_date"] = *req.DueDate
	}

	res := r.db.WithContext(ctx).
		Table("tasks").
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, userID, id)
}

func (r *repository) Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	res := r.db.WithContext(ctx).
		Table("tasks").
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]any{
			"status":       StatusCompleted,
			"completed_at": gorm.Expr("NOW()"),
			"updated_at":   gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, userID, id)
}

func (r *repository) Delete(ctx context.Context, userID int64, id int64) error {
	res := r.db.WithContext(ctx).
		Table("tasks").
		Where("id = ? AND user_id = ?", id, userID).
		Delete(nil)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
