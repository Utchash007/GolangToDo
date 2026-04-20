package task_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"GolangToDo/internal/task"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockService implements task.Service for handler tests.
type mockService struct {
	createTask   func(ctx context.Context, req task.CreateTaskRequest) (*task.Task, error)
	getTask      func(ctx context.Context, id string) (*task.Task, error)
	getAllTasks   func(ctx context.Context) ([]*task.Task, error)
	updateTask   func(ctx context.Context, id string, req task.UpdateTaskRequest) (*task.Task, error)
	deleteTask   func(ctx context.Context, id string) error
	bulkComplete func(ctx context.Context, ids []string) ([]*task.Task, error)
	bulkDelete   func(ctx context.Context, ids []string) (int64, error)
}

func (m *mockService) CreateTask(ctx context.Context, req task.CreateTaskRequest) (*task.Task, error) {
	if m.createTask != nil {
		return m.createTask(ctx, req)
	}
	return &task.Task{}, nil
}
func (m *mockService) GetTask(ctx context.Context, id string) (*task.Task, error) {
	if m.getTask != nil {
		return m.getTask(ctx, id)
	}
	return &task.Task{}, nil
}
func (m *mockService) GetAllTasks(ctx context.Context) ([]*task.Task, error) {
	if m.getAllTasks != nil {
		return m.getAllTasks(ctx)
	}
	return []*task.Task{}, nil
}
func (m *mockService) UpdateTask(ctx context.Context, id string, req task.UpdateTaskRequest) (*task.Task, error) {
	if m.updateTask != nil {
		return m.updateTask(ctx, id, req)
	}
	return &task.Task{}, nil
}
func (m *mockService) DeleteTask(ctx context.Context, id string) error {
	if m.deleteTask != nil {
		return m.deleteTask(ctx, id)
	}
	return nil
}
func (m *mockService) BulkComplete(ctx context.Context, ids []string) ([]*task.Task, error) {
	if m.bulkComplete != nil {
		return m.bulkComplete(ctx, ids)
	}
	return []*task.Task{}, nil
}
func (m *mockService) BulkDelete(ctx context.Context, ids []string) (int64, error) {
	if m.bulkDelete != nil {
		return m.bulkDelete(ctx, ids)
	}
	return 0, nil
}

func newRouter(svc task.Service) *gin.Engine {
	r := gin.New()
	task.NewHandler(svc).RegisterRoutes(r)
	return r
}

func TestCreateTask_BindingErrors(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantStatus     int
		wantCode       string
		wantFieldCount int
		wantField      string
	}{
		{
			name:           "missing title",
			body:           `{"priority":"high","category":"work"}`,
			wantStatus:     http.StatusBadRequest,
			wantCode:       "invalid_request",
			wantFieldCount: 1,
			wantField:      "Title",
		},
		{
			name:           "missing priority",
			body:           `{"title":"Test","category":"work"}`,
			wantStatus:     http.StatusBadRequest,
			wantCode:       "invalid_request",
			wantFieldCount: 1,
			wantField:      "Priority",
		},
		{
			name:           "multiple missing fields",
			body:           `{"category":"work"}`,
			wantStatus:     http.StatusBadRequest,
			wantCode:       "invalid_request",
			wantFieldCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRouter(&mockService{})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(tt.body)))

			require.Equal(t, tt.wantStatus, w.Code)

			var resp task.ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tt.wantCode, resp.Code)
			assert.Len(t, resp.Errors, tt.wantFieldCount)
			if tt.wantField != "" {
				assert.Equal(t, tt.wantField, resp.Errors[0].Field)
			}
		})
	}
}

func TestCreateTask_MalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"syntax error", `{bad json}`},
		{"wrong type", `{"title":123,"priority":"high","category":"work"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRouter(&mockService{})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(tt.body)))

			require.Equal(t, http.StatusBadRequest, w.Code)

			var resp task.ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, "invalid_request", resp.Code)
			require.Len(t, resp.Errors, 1)
			assert.Empty(t, resp.Errors[0].Field)
			assert.Equal(t, "invalid request body", resp.Errors[0].Message)
		})
	}
}

func TestCreateTask_ServiceValidationError(t *testing.T) {
	svc := &mockService{
		createTask: func(_ context.Context, _ task.CreateTaskRequest) (*task.Task, error) {
			return nil, &task.ValidationError{Message: "priority must be low, medium, or high"}
		},
	}
	r := newRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks",
		strings.NewReader(`{"title":"T","priority":"critical","category":"work"}`)))

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp task.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid_request", resp.Code)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, "priority must be low, medium, or high", resp.Errors[0].Message)
}

func TestCreateTask_InternalError(t *testing.T) {
	svc := &mockService{
		createTask: func(_ context.Context, _ task.CreateTaskRequest) (*task.Task, error) {
			return nil, errors.New("db connection lost")
		},
	}
	r := newRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks",
		strings.NewReader(`{"title":"T","priority":"high","category":"work"}`)))

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp task.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "internal_error", resp.Code)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, "internal server error", resp.Errors[0].Message)
	assert.NotContains(t, w.Body.String(), "db connection lost")
}

func TestGetTask_NotFound(t *testing.T) {
	svc := &mockService{
		getTask: func(_ context.Context, _ string) (*task.Task, error) {
			return nil, task.ErrNotFound
		},
	}
	r := newRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tasks/nonexistent-id", nil))

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp task.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "not_found", resp.Code)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, "task not found", resp.Errors[0].Message)

	// field must be omitted (omitempty) for non-field errors
	assert.NotContains(t, w.Body.String(), `"field"`)
}

func TestUpdateTask_EmptyBody(t *testing.T) {
	r := newRouter(&mockService{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/tasks/some-id",
		strings.NewReader(`{}`)))

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp task.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid_request", resp.Code)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, "request body must not be empty", resp.Errors[0].Message)
}

func TestUpdateTask_ValidPartialUpdate(t *testing.T) {
	called := false
	svc := &mockService{
		updateTask: func(_ context.Context, _ string, _ task.UpdateTaskRequest) (*task.Task, error) {
			called = true
			return &task.Task{}, nil
		},
	}
	r := newRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPatch, "/tasks/some-id",
		strings.NewReader(`{"title":"New Title"}`)))

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "service.UpdateTask should have been called")
}

func TestErrorResponse_Shape(t *testing.T) {
	svc := &mockService{
		getTask: func(_ context.Context, _ string) (*task.Task, error) {
			return nil, task.ErrNotFound
		},
	}
	r := newRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tasks/x", nil))

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	// must have code and errors, must NOT have top-level message
	assert.Contains(t, resp, "code")
	assert.Contains(t, resp, "errors")
	assert.NotContains(t, resp, "message")
}

func TestBulkComplete_Success(t *testing.T) {
	svc := &mockService{
		bulkComplete: func(_ context.Context, ids []string) ([]*task.Task, error) {
			tasks := make([]*task.Task, len(ids))
			for i := range ids {
				t := &task.Task{Completed: true}
				tasks[i] = t
			}
			return tasks, nil
		},
	}
	r := newRouter(svc)
	w := httptest.NewRecorder()
	body := `{"ids":["550e8400-e29b-41d4-a716-446655440000","550e8400-e29b-41d4-a716-446655440001"]}`
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks/bulk-complete", strings.NewReader(body)))

	require.Equal(t, http.StatusOK, w.Code)
	var tasks []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tasks))
	assert.Len(t, tasks, 2)
}

func TestBulkComplete_EmptyIDs(t *testing.T) {
	r := newRouter(&mockService{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks/bulk-complete",
		strings.NewReader(`{"ids":[]}`)))

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp task.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid_request", resp.Code)
}

func TestBulkComplete_MissingIDs(t *testing.T) {
	r := newRouter(&mockService{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks/bulk-complete",
		strings.NewReader(`{}`)))

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkComplete_InvalidUUID(t *testing.T) {
	svc := &mockService{
		bulkComplete: func(_ context.Context, _ []string) ([]*task.Task, error) {
			return nil, &task.ValidationError{Message: `invalid id: "bad-id"`}
		},
	}
	r := newRouter(svc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks/bulk-complete",
		strings.NewReader(`{"ids":["bad-id"]}`)))

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp task.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid_request", resp.Code)
}

func TestBulkDelete_Success(t *testing.T) {
	svc := &mockService{
		bulkDelete: func(_ context.Context, ids []string) (int64, error) {
			return int64(len(ids)), nil
		},
	}
	r := newRouter(svc)
	w := httptest.NewRecorder()
	body := `{"ids":["550e8400-e29b-41d4-a716-446655440000"]}`
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/tasks/bulk", strings.NewReader(body)))

	require.Equal(t, http.StatusOK, w.Code)
	var resp task.BulkDeleteResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Deleted)
}

func TestBulkDelete_EmptyIDs(t *testing.T) {
	r := newRouter(&mockService{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/tasks/bulk",
		strings.NewReader(`{"ids":[]}`)))

	require.Equal(t, http.StatusBadRequest, w.Code)
}
