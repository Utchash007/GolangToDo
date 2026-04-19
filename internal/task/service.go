package task

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Service interface {
	CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error)
	GetTask(ctx context.Context, id string) (*Task, error)
	GetAllTasks(ctx context.Context) ([]*Task, error)
	UpdateTask(ctx context.Context, id string, req UpdateTaskRequest) (*Task, error)
	DeleteTask(ctx context.Context, id string) error
}

type service struct {
	repo Repository
}

func NewService(db *sqlx.DB) Service {
	return &service{repo: NewRepository(db)}
}

func (s *service) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	priority := ParsePriority(req.Priority)
	if !priority.IsValid() {
		return nil, &ValidationError{"priority must be low, medium, or high"}
	}

	category := strings.ToLower(strings.TrimSpace(req.Category))
	if category == "" {
		return nil, &ValidationError{"category is required"}
	}
	task := NewTask(req.Title, priority, category)
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *service) GetTask(ctx context.Context, id string) (*Task, error) {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.repo.GetByID(ctx, taskID)
}

func (s *service) GetAllTasks(ctx context.Context) ([]*Task, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) UpdateTask(ctx context.Context, id string, req UpdateTaskRequest) (*Task, error) {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}

	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, &ValidationError{"title cannot be empty"}
		}
		task.Title = *req.Title
	}
	if req.Priority != nil {
		priority := ParsePriority(*req.Priority)
		if !priority.IsValid() {
			return nil, &ValidationError{"priority must be low, medium, or high"}
		}
		task.Priority = priority
	}
	if req.Category != nil {
		lower := strings.ToLower(strings.TrimSpace(*req.Category))
		if lower == "" {
			return nil, &ValidationError{"category cannot be empty"}
		}
		task.Category = &lower
	}
	if req.Completed != nil {
		task.Completed = *req.Completed
	}
	task.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *service) DeleteTask(ctx context.Context, id string) error {
	taskID, err := uuid.Parse(id)
	if err != nil {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, taskID)
}
