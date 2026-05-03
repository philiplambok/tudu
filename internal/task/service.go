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
	return s.repo.Create(ctx, CreateTaskRecordDTO{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		DueDate:     req.DueDate,
	})
}

func (s *service) List(ctx context.Context, userID int64, status string) ([]TaskResponseDTO, error) {
	return s.repo.List(ctx, userID, status)
}

func (s *service) Get(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *service) Update(ctx context.Context, userID int64, id int64, req UpdateRequestDTO) (*TaskResponseDTO, error) {
	if err := ValidateUpdate(req); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, userID, id, UpdateTaskRecordDTO{
		Title:       req.Title,
		Description: req.Description,
		DueDate:     req.DueDate,
	})
}

func (s *service) Complete(ctx context.Context, userID int64, id int64) (*TaskResponseDTO, error) {
	return s.repo.Complete(ctx, userID, id)
}

func (s *service) Delete(ctx context.Context, userID int64, id int64) error {
	return s.repo.Delete(ctx, userID, id)
}
