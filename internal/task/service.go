package task

import "context"

type Service interface {
	Create(ctx context.Context, userID int64, req CreateRequestDTO) (*TaskResponseDTO, error)
	List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error)
	Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error)
	Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error)
	Delete(ctx context.Context, userID int64, id int64) error
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
	rec, err := s.repo.Create(ctx, CreateTaskRecordDTO{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		DueDate:     req.DueDate,
	})
	if err != nil {
		return nil, err
	}
	return toResponseDTO(rec), nil
}

func (s *service) List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error) {
	recs, err := s.repo.List(ctx, userID, status)
	if err != nil {
		return nil, err
	}
	out := make([]TaskResponseDTO, len(recs))
	for i := range recs {
		out[i] = *toResponseDTO(&recs[i])
	}
	return out, nil
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
	rec, err := s.repo.Update(ctx, userID, id, req)
	if err != nil {
		return nil, err
	}
	return toResponseDTO(rec), nil
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
