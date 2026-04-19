package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	Repository
	task  *Task
	tasks []*Task
}

func (m *mockRepo) Create(_ context.Context, _ *Task) error         { return nil }
func (m *mockRepo) GetAll(_ context.Context) ([]*Task, error)       { return m.tasks, nil }
func (m *mockRepo) Update(_ context.Context, _ *Task) error         { return nil }
func (m *mockRepo) Delete(_ context.Context, _ uuid.UUID) error     { return nil }

func (m *mockRepo) GetByID(_ context.Context, _ uuid.UUID) (*Task, error) {
	if m.task == nil {
		return nil, ErrNotFound
	}
	return m.task, nil
}

func TestService_CreateTask(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	tests := []struct {
		name   string
		req    CreateTaskRequest
		errMsg string
	}{
		{"valid", CreateTaskRequest{Title: "Test", Priority: "medium", Category: "work"}, ""},
		{"invalid priority", CreateTaskRequest{Title: "Test", Priority: "critical", Category: "work"}, "priority must be low, medium, or high"},
		{"missing category", CreateTaskRequest{Title: "Test", Priority: "medium"}, "category is required"},
		{"whitespace category", CreateTaskRequest{Title: "Test", Priority: "medium", Category: "   "}, "category is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := svc.CreateTask(context.Background(), tt.req)
			if tt.errMsg != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
				assert.Nil(t, task)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, task)
			}
		})
	}
}

func TestService_CreateTask_CategoryNormalized(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	task, err := svc.CreateTask(context.Background(), CreateTaskRequest{
		Title:    "Test",
		Priority: "medium",
		Category: "  Work  ",
	})

	require.NoError(t, err)
	require.NotNil(t, task.Category)
	assert.Equal(t, "work", *task.Category)
}


func TestService_GetTask(t *testing.T) {
	svc := &service{repo: &mockRepo{task: NewTask("Test", PriorityMedium, "work")}}

	task, err := svc.GetTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000")

	require.NoError(t, err)
	assert.NotNil(t, task)
}

func TestService_GetTask_InvalidUUID(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	task, err := svc.GetTask(context.Background(), "invalid-uuid")

	assert.ErrorIs(t, err, ErrNotFound)
	assert.Nil(t, task)
}

func TestService_UpdateTask(t *testing.T) {
	svc := &service{repo: &mockRepo{task: NewTask("Test", PriorityMedium, "work")}}

	title := "Updated Title"
	task, err := svc.UpdateTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000", UpdateTaskRequest{Title: &title})

	require.NoError(t, err)
	assert.Equal(t, "Updated Title", task.Title)
}

func TestService_UpdateTask_InvalidPriority(t *testing.T) {
	svc := &service{repo: &mockRepo{task: NewTask("Test", PriorityMedium, "work")}}

	p := "invalid"
	_, err := svc.UpdateTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000", UpdateTaskRequest{Priority: &p})

	var ve *ValidationError
	assert.ErrorAs(t, err, &ve)
	assert.EqualError(t, err, "priority must be low, medium, or high")
}

func TestService_UpdateTask_EmptyTitle(t *testing.T) {
	svc := &service{repo: &mockRepo{task: NewTask("Test", PriorityMedium, "work")}}

	empty := ""
	_, err := svc.UpdateTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000", UpdateTaskRequest{Title: &empty})

	var ve *ValidationError
	assert.ErrorAs(t, err, &ve)
}

func TestService_UpdateTask_NotFound(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	title := "Updated"
	_, err := svc.UpdateTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000", UpdateTaskRequest{Title: &title})

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_DeleteTask(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	err := svc.DeleteTask(context.Background(), "550e8400-e29b-41d4-a716-446655440000")

	assert.NoError(t, err)
}

func TestService_DeleteTask_InvalidUUID(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	err := svc.DeleteTask(context.Background(), "invalid-uuid")

	assert.ErrorIs(t, err, ErrNotFound)
}
