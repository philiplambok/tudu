package task

import (
	"context"

	"github.com/philiplambok/tudu/internal/common/util"
)

type Service interface {
	Create(ctx context.Context, userID int64, req CreateRequestDTO) (*TaskResponseDTO, error)
	List(ctx context.Context, userID int64, status string, paging util.PagingRequest) (util.PagingResponse[TaskResponseDTO], error)
	Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error)
	Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Delete(ctx context.Context, userID int64, id int64) error
	ListActivities(ctx context.Context, userID int64, taskID int64) ([]TaskActivityResponseDTO, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, userID int64, req CreateRequestDTO) (*TaskResponseDTO, error) {
	if err := ValidateCreate(req); err != nil {
		return nil, err
	}
	agg := NewTask(CreateTaskRecordDTO{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		DueDate:     req.DueDate,
	})
	rec, err := s.repo.Create(ctx, agg)
	if err != nil {
		return nil, err
	}
	return toResponseDTO(rec), nil
}

func (s *service) List(ctx context.Context, userID int64, status string, paging util.PagingRequest) (util.PagingResponse[TaskResponseDTO], error) {
	recs, total, err := s.repo.List(ctx, ListTaskRecordParams{
		UserID:        userID,
		Status:        status,
		PagingRequest: paging,
	})
	if err != nil {
		return util.PagingResponse[TaskResponseDTO]{}, err
	}
	out := make([]TaskResponseDTO, len(recs))
	for i := range recs {
		out[i] = *toResponseDTO(&recs[i])
	}
	return util.NewPagingResponse(out, total, paging.Page, paging.Limit), nil
}

func (s *service) Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	rec, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return toResponseDTO(rec), nil
}

func (s *service) Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error) {
	if err := ValidateUpdate(req); err != nil {
		return nil, err
	}
	rec, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	agg := TaskFromRecord(*rec).ApplyUpdate(req)
	saved, err := s.repo.Update(ctx, userID, agg)
	if err != nil {
		return nil, err
	}
	return toResponseDTO(saved), nil
}

func (s *service) Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	rec, err := s.repo.Complete(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return toResponseDTO(rec), nil
}

func (s *service) Delete(ctx context.Context, userID int64, id int64) error {
	return s.repo.Delete(ctx, userID, id)
}

func (s *service) ListActivities(ctx context.Context, userID int64, taskID int64) ([]TaskActivityResponseDTO, error) {
	recs, err := s.repo.ListActivities(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}
	out := make([]TaskActivityResponseDTO, len(recs))
	for i := range recs {
		out[i] = *toActivityResponseDTO(&recs[i])
	}
	return out, nil
}

func toResponseDTO(r *TaskRecordDTO) *TaskResponseDTO {
	return &TaskResponseDTO{
		ID:          r.ID,
		UserID:      r.UserID,
		Title:       r.Title,
		Description: r.Description,
		Status:      r.Status,
		DueDate:     r.DueDate,
		CompletedAt: r.CompletedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func toActivityResponseDTO(r *TaskActivityRecordDTO) *TaskActivityResponseDTO {
	return &TaskActivityResponseDTO{
		ID:        r.ID,
		TaskID:    r.TaskID,
		UserID:    r.UserID,
		Action:    r.Action,
		FieldName: r.FieldName,
		OldValue:  r.OldValue,
		NewValue:  r.NewValue,
		CreatedAt: r.CreatedAt,
	}
}
