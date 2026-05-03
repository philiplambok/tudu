package task

import (
	"context"
	"fmt"

	"github.com/philiplambok/tudu/internal/common/datamodel"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, task Task) (*TaskRecordDTO, error)
	List(ctx context.Context, userID int64, status string) ([]TaskRecordDTO, error)
	Get(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error)
	Update(ctx context.Context, userID int64, task Task) (*TaskRecordDTO, error)
	Complete(ctx context.Context, userID int64, id int64) (*TaskRecordDTO, error)
	Delete(ctx context.Context, userID int64, id int64) error
	ListActivities(ctx context.Context, userID int64, taskID int64) ([]TaskActivityRecordDTO, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, agg Task) (*TaskRecordDTO, error) {
	var out *TaskRecordDTO
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row datamodel.Task
		res := tx.Raw(`
			INSERT INTO tasks (user_id, title, description, status, due_date, created_at, updated_at)
			VALUES (?, ?, ?, 'pending', ?, NOW(), NOW())
			RETURNING id, user_id, title, description, status, due_date, completed_at, created_at, updated_at`,
			agg.UserID, agg.Title, agg.Description, agg.DueDate,
		).Scan(&row)
		if res.Error != nil {
			return res.Error
		}
		for _, act := range agg.Activities {
			if err := insertTaskActivity(ctx, tx, row.ID, row.UserID, act.Action, act.FieldName, act.OldValue, act.NewValue); err != nil {
				return err
			}
		}
		out = toTaskRecordDTO(&row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
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

func (r *repository) Update(ctx context.Context, userID int64, agg Task) (*TaskRecordDTO, error) {
	var out *TaskRecordDTO
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row datamodel.Task
		res := tx.Raw(`
			UPDATE tasks SET title = ?, description = ?, due_date = ?, updated_at = NOW()
			WHERE id = ? AND user_id = ?
			RETURNING id, user_id, title, description, status, due_date, completed_at, created_at, updated_at`,
			agg.Title, agg.Description, agg.DueDate, agg.ID, userID,
		).Scan(&row)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}

		for _, act := range agg.Activities {
			if err := insertTaskActivity(ctx, tx, agg.ID, userID, act.Action, act.FieldName, act.OldValue, act.NewValue); err != nil {
				return fmt.Errorf("insert task activity: %w", err)
			}
		}

		out = toTaskRecordDTO(&row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
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

func (r *repository) ListActivities(ctx context.Context, userID int64, taskID int64) ([]TaskActivityRecordDTO, error) {
	var taskRow datamodel.Task
	taskRes := r.db.WithContext(ctx).Raw(`
		SELECT id, user_id, title, description, status, due_date, completed_at, created_at, updated_at
		FROM tasks WHERE id = ? AND user_id = ?`, taskID, userID,
	).Scan(&taskRow)
	if taskRes.Error != nil {
		return nil, taskRes.Error
	}
	if taskRes.RowsAffected == 0 {
		return nil, ErrNotFound
	}

	var rows []datamodel.TaskActivity
	if err := r.db.WithContext(ctx).
		Select("id, task_id, user_id, action, field_name, old_value, new_value, created_at").
		Table("task_activities").
		Where("task_id = ? AND user_id = ?", taskID, userID).
		Order("created_at DESC, id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]TaskActivityRecordDTO, len(rows))
	for i := range rows {
		out[i] = *toTaskActivityRecordDTO(&rows[i])
	}
	return out, nil
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

func toTaskActivityRecordDTO(m *datamodel.TaskActivity) *TaskActivityRecordDTO {
	return &TaskActivityRecordDTO{
		ID:        m.ID,
		TaskID:    m.TaskID,
		UserID:    m.UserID,
		Action:    m.Action,
		FieldName: m.FieldName,
		OldValue:  m.OldValue,
		NewValue:  m.NewValue,
		CreatedAt: m.CreatedAt,
	}
}

func insertTaskActivity(ctx context.Context, db *gorm.DB, taskID int64, userID int64, action string, fieldName *string, oldValue *string, newValue *string) error {
	return db.WithContext(ctx).Exec(`
		INSERT INTO task_activities (task_id, user_id, action, field_name, old_value, new_value, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())`,
		taskID, userID, action, fieldName, oldValue, newValue,
	).Error
}
