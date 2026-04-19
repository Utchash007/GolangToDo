package task

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("task not found")

type Repository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*Task, error)
	GetAll(ctx context.Context) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (id, title, priority, category, completed, created_at, updated_at)
		VALUES (:id, :title, :priority, :category, :completed, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, task)
	return err
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Task, error) {
	var task Task
	query := `SELECT * FROM tasks WHERE id = $1`
	err := r.db.GetContext(ctx, &task, query, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return &task, nil
}

func (r *repository) GetAll(ctx context.Context) ([]*Task, error) {
	var tasks []*Task
	query := `SELECT * FROM tasks ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &tasks, query)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *repository) Update(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks SET
			title = :title,
			priority = :priority,
			category = :category,
			completed = :completed,
			updated_at = :updated_at
		WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, task)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}