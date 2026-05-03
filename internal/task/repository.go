package task

import (
	"context"

	"github.com/philiplambok/tudu/internal/common/datamodel"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, rec CreateTaskRecordDTO) (*TaskRecordDTO, error)
	List(ctx context.Context, userID int64, status string) ([]TaskRecordDTO, error)
	Get(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error)
	Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskRecordDTO, error)
	Complete(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error)
	Delete(ctx context.Context, userID int64, id int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, rec CreateTaskRecordDTO) (*TaskRecordDTO, error) {
	var row datamodel.Task
	res := r.db.WithContext(ctx).Raw(`
		INSERT INTO tasks (user_id, title, description, status, due_date, created_at, updated_at)
		VALUES (?, ?, ?, 'pending', ?, NOW(), NOW())
		RETURNING id, user_id, title, description, status, due_date, completed_at, created_at, updated_at`,
		rec.UserID, rec.Title, rec.Description, rec.DueDate,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	return toTaskRecordDTO(&row), nil
}

func (r *repository) List(ctx context.Context, userID int64, status string) ([]TaskRecordDTO, error) {
	q := r.db.WithContext(ctx).
		Select("id, user_id, title, description, status, due_date, completed_at, created_at, updated_at").
		Table("tasks").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if status != "" {
		q = q.Where("status = ?", status)
	}

	var rows []datamodel.Task
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]TaskRecordDTO, len(rows))
	for i := range rows {
		out[i] = *toTaskRecordDTO(&rows[i])
	}
	return out, nil
}

func (r *repository) Get(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error) {
	var row datamodel.Task
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
	return toTaskRecordDTO(&row), nil
}

func (r *repository) Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskRecordDTO, error) {
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

func (r *repository) Complete(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error) {
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

func toTaskRecordDTO(m *datamodel.Task) *TaskRecordDTO {
	return &TaskRecordDTO{
		ID:          m.ID,
		UserID:      m.UserID,
		Title:       m.Title,
		Description: m.Description,
		Status:      m.Status,
		DueDate:     m.DueDate,
		CompletedAt: m.CompletedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
