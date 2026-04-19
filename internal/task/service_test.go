package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	Repository
	task  *Task
	tasks []*Task
}

func (m *mockRepo) Create(ctx context.Context, task *Task) error {
	return nil
}
func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*Task, error) {
	if m.task == nil {
		return nil, ErrNotFound
	}
	return m.task, nil
}
func (m *mockRepo) GetAll(ctx context.Context) ([]*Task, error) {
	return m.tasks, nil
}
func (m *mockRepo) Update(ctx context.Context, task *Task) error {
	return nil
}
func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestService_CreateTask(t *testing.T) {
	db := &mockRepo{}
	svc := &service{repo: db}

	tests := []struct {
		name    string
		req    CreateTaskRequest
		errMsg string
	}{
		{"valid", CreateTaskRequest{Title: "Test", Priority: "medium", Category: "work"}, ""},
		{"missing title", CreateTaskRequest{Priority: "medium", Category: "work"}, "title is required"},
		{"empty title", CreateTaskRequest{Title: "", Priority: "medium", Category: "work"}, "title is required"},
		{"missing priority", CreateTaskRequest{Title: "Test", Category: "work"}, "priority is required"},
		{"invalid priority", CreateTaskRequest{Title: "Test", Priority: "critical", Category: "work"}, "priority must be low, medium, or high"},
		{"missing category", CreateTaskRequest{Title: "Test", Priority: "medium"}, "category is required"},
		{"empty category", CreateTaskRequest{Title: "Test", Priority: "medium", Category: ""}, "category is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := svc.CreateTask(context.Background(), tt.req)
			if tt.errMsg != "" {
				assert.EqualError(t, err, tt.errMsg)
				assert.Nil(t, task)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
			}
		})
	}
}

func TestService_GetTask(t *testing.T) {
	db := &mockRepo{task: NewTask("Test", PriorityMedium, "work")}
	svc := &service{repo: db}

	task, err := svc.GetTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)
	assert.NotNil(t, task)
}

func TestService_GetTask_InvalidUUID(t *testing.T) {
	db := &mockRepo{}
	svc := &service{repo: db}

	task, err := svc.GetTask(context.Background(), "invalid-uuid")
	assert.EqualError(t, err, "task not found")
	assert.Nil(t, task)
}

func TestService_UpdateTask(t *testing.T) {
	db := &mockRepo{task: NewTask("Test", PriorityMedium, "work")}
	svc := &service{repo: db}

	title := "Updated Title"
	req := UpdateTaskRequest{Title: &title}
	task, err := svc.UpdateTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000", req)

	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", task.Title)
}

func TestService_UpdateTask_NotFound(t *testing.T) {
	db := &mockRepo{}
	svc := &service{repo: db}

	title := "Updated Title"
	req := UpdateTaskRequest{Title: &title}
	task, err := svc.UpdateTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000", req)

	assert.EqualError(t, err, "task not found")
	assert.Nil(t, task)
}

func TestService_DeleteTask(t *testing.T) {
	db := &mockRepo{}
	svc := &service{repo: db}

	err := svc.DeleteTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	assert.NoError(t, err)
}

func TestService_DeleteTask_InvalidUUID(t *testing.T) {
	db := &mockRepo{}
	svc := &service{repo: db}

	err := svc.DeleteTask(context.Background(), "invalid-uuid")
	assert.EqualError(t, err, "task not found")
}