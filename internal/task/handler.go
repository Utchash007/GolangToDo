package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/tasks", h.CreateTask)
	r.GET("/tasks", h.ListTasks)
	r.GET("/tasks/:id", h.GetTask)
	r.PATCH("/tasks/:id", h.UpdateTask)
	r.DELETE("/tasks/:id", h.DeleteTask)
}

func (h *Handler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:   "invalid_request",
			Errors: bindingErrors(err),
		})
		return
	}

	task, err := h.svc.CreateTask(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *Handler) ListTasks(c *gin.Context) {
	var f TaskFilter

	if p := c.Query("priority"); p != "" {
		f.Priority = ParsePriority(p)
		if !f.Priority.IsValid() {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:   "invalid_request",
				Errors: []FieldError{{Field: "priority", Message: "must be low, medium, or high"}},
			})
			return
		}
	}
	if cat := c.Query("category"); cat != "" {
		lower := strings.ToLower(cat)
		f.Category = &lower
	}
	if comp := c.Query("completed"); comp != "" {
		b, err := strconv.ParseBool(comp)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:   "invalid_request",
				Errors: []FieldError{{Field: "completed", Message: "must be true or false"}},
			})
			return
		}
		f.Completed = &b
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err != nil || v < 1 || v > 100 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:   "invalid_request",
				Errors: []FieldError{{Field: "limit", Message: "must be between 1 and 100"}},
			})
			return
		}
		limit = v
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		v, err := strconv.Atoi(o)
		if err != nil || v < 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:   "invalid_request",
				Errors: []FieldError{{Field: "offset", Message: "must be 0 or greater"}},
			})
			return
		}
		offset = v
	}

	result, err := h.svc.ListTasks(c.Request.Context(), ListParams{Filter: f, Limit: limit, Offset: offset})
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetTask(c *gin.Context) {
	id := c.Param("id")

	task, err := h.svc.GetTask(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *Handler) UpdateTask(c *gin.Context) {
	id := c.Param("id")

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:   "invalid_request",
			Errors: bindingErrors(err),
		})
		return
	}

	if isEmptyUpdate(req) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:   "invalid_request",
			Errors: []FieldError{{Message: "request body must not be empty"}},
		})
		return
	}

	task, err := h.svc.UpdateTask(c.Request.Context(), id, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *Handler) DeleteTask(c *gin.Context) {
	id := c.Param("id")

	if err := h.svc.DeleteTask(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// handleError maps domain errors to the unified HTTP error shape.
func handleError(c *gin.Context, err error) {
	var ve *ValidationError
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:   "not_found",
			Errors: []FieldError{{Message: "task not found"}},
		})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:   "invalid_request",
			Errors: []FieldError{{Message: ve.Error()}},
		})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:   "internal_error",
			Errors: []FieldError{{Message: "internal server error"}},
		})
	}
}

// bindingErrors translates a ShouldBindJSON error into []FieldError.
// validator.ValidationErrors produce one entry per field; JSON errors produce a single generic entry.
func bindingErrors(err error) []FieldError {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		out := make([]FieldError, len(ve))
		for i, fe := range ve {
			out[i] = FieldError{Field: fe.Field(), Message: fe.Tag()}
		}
		return out
	}

	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
		return []FieldError{{Message: "invalid request body"}}
	}

	return []FieldError{{Message: "invalid request body"}}
}

// isEmptyUpdate returns true when no fields are set on an UpdateTaskRequest.
func isEmptyUpdate(req UpdateTaskRequest) bool {
	return req.Title == nil && req.Priority == nil && req.Category == nil && req.Completed == nil
}
