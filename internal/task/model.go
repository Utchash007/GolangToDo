package task

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ValidationError is returned by the service for invalid input — maps to HTTP 400.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

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

// Value implements driver.Valuer — writes Priority as a string to the PostgreSQL ENUM column.
func (p Priority) Value() (driver.Value, error) {
	if !p.IsValid() {
		return nil, fmt.Errorf("invalid priority value: %d", p)
	}
	return p.String(), nil
}

// Scan implements sql.Scanner — reads a PostgreSQL ENUM string into Priority.
func (p *Priority) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("priority: expected string, got %T", src)
	}
	*p = ParsePriority(s)
	if !p.IsValid() {
		return fmt.Errorf("invalid priority value: %q", s)
	}
	return nil
}

// MarshalJSON serializes Priority as a string ("low", "medium", "high").
func (p Priority) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// UnmarshalJSON parses Priority from a JSON string.
func (p *Priority) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*p = ParsePriority(s)
	return nil
}

type Task struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	Title     string    `db:"title"      json:"title"`
	Priority  Priority  `db:"priority"   json:"priority"`
	Category  *string   `db:"category"   json:"category"`
	Completed bool      `db:"completed"  json:"completed"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// NewTask creates a Task. category is optional; empty string → nil.
func NewTask(title string, priority Priority, category string) *Task {
	var cat *string
	if category != "" {
		lower := strings.ToLower(category)
		cat = &lower
	}
	now := time.Now().UTC()
	return &Task{
		ID:        uuid.New(),
		Title:     title,
		Priority:  priority,
		Category:  cat,
		Completed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type CreateTaskRequest struct {
	Title    string `json:"title"    binding:"required"`
	Priority string `json:"priority" binding:"required"`
	Category string `json:"category" binding:"required"`
}

type UpdateTaskRequest struct {
	Title     *string `json:"title,omitempty"`
	Priority  *string `json:"priority,omitempty"`
	Category  *string `json:"category,omitempty"`
	Completed *bool   `json:"completed,omitempty"`
}

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ErrorResponse is the single error shape returned by all endpoints.
type ErrorResponse struct {
	Code   string       `json:"code"`
	Errors []FieldError `json:"errors"`
}
