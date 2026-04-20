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
func (m *mockRepo) BulkComplete(_ context.Context, ids []uuid.UUID) ([]*Task, error) {
	tasks := make([]*Task, len(ids))
	for i := range ids {
		t := NewTask("t", PriorityLow, "c")
		t.Completed = true
		tasks[i] = t
	}
	return tasks, nil
}
func (m *mockRepo) BulkDelete(_ context.Context, ids []uuid.UUID) (int64, error) {
	return int64(len(ids)), nil
}

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

func TestService_BulkComplete(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	ids := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
	}
	tasks, err := svc.BulkComplete(context.Background(), ids)

	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.True(t, task.Completed)
	}
}

func TestService_BulkComplete_InvalidUUID(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	_, err := svc.BulkComplete(context.Background(), []string{"not-a-uuid"})

	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, ve.Error(), "invalid id")
}

func TestService_BulkDelete(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	ids := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
	}
	n, err := svc.BulkDelete(context.Background(), ids)

	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestService_BulkDelete_InvalidUUID(t *testing.T) {
	svc := &service{repo: &mockRepo{}}

	_, err := svc.BulkDelete(context.Background(), []string{"bad-id"})

	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
}
