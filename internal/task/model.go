package task

import (
	"time"

	"github.com/google/uuid"
)

type Priority int

const (
	PriorityUnknown Priority = iota
	PriorityLow
	PriorityMedium
	PriorityHigh
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	default:
		return "unknown"
	}
}

func ParsePriority(s string) Priority {
	switch s {
	case "low":
		return PriorityLow
	case "medium":
		return PriorityMedium
	case "high":
		return PriorityHigh
	default:
		return PriorityUnknown
	}
}

func (p Priority) IsValid() bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh
}

type Task struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	Title     string    `db:"title" json:"title"`
	Priority Priority  `db:"priority" json:"priority"`
	Category string    `db:"category" json:"category"`
	Completed bool    `db:"completed" json:"completed"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func NewTask(title string, priority Priority, category string) *Task {
	now := time.Now().UTC()
	return &Task{
		ID:        uuid.New(),
		Title:     title,
		Priority: priority,
		Category: category,
		Completed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type CreateTaskRequest struct {
	Title     string `json:"title" binding:"required"`
	Priority string `json:"priority" binding:"required"`
	Category string `json:"category" binding:"required"`
}

type UpdateTaskRequest struct {
	Title     *string `json:"title,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Category *string `json:"category,omitempty"`
	Completed *bool  `json:"completed,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}