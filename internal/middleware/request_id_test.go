package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"GolangToDo/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID_SetsHeader(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	id := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, id)
}

func TestRequestID_InjectsContext(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())

	var capturedID string
	r.GET("/", func(c *gin.Context) {
		capturedID = middleware.RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	require.NotEmpty(t, capturedID)
	assert.Equal(t, w.Header().Get("X-Request-ID"), capturedID)
}

func TestRequestIDFromContext_EmptyWhenMissing(t *testing.T) {
	id := middleware.RequestIDFromContext(t.Context())
	assert.Empty(t, id)
}
