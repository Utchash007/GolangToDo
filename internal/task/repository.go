package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const dbTimeout = 5 * time.Second

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
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()
	query := `
		INSERT INTO tasks (id, title, priority, category, completed, created_at, updated_at)
		VALUES (:id, :title, :priority, :category, :completed, :created_at, :updated_at)`
	if _, err := r.db.NamedExecContext(ctx, query, task); err != nil {
		return fmt.Errorf("creating task: %w", err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Task, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()
	var task Task
	const query = `
		SELECT id, title, priority, category, completed, created_at, updated_at
		FROM tasks WHERE id = $1`
	if err := r.db.GetContext(ctx, &task, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting task %s: %w", id, err)
	}
	return &task, nil
}

func (r *repository) GetAll(ctx context.Context) ([]*Task, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()
	tasks := make([]*Task, 0)
	const query = `
		SELECT id, title, priority, category, completed, created_at, updated_at
		FROM tasks ORDER BY created_at DESC`
	if err := r.db.SelectContext(ctx, &tasks, query); err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	return tasks, nil
}

func (r *repository) Update(ctx context.Context, task *Task) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()
	const query = `
		UPDATE tasks SET
			title      = :title,
			priority   = :priority,
			category   = :category,
			completed  = :completed,
			updated_at = :updated_at
		WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, task)
	if err != nil {
		return fmt.Errorf("updating task %s: %w", task.ID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()
	result, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting task %s: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
