package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"GolangToDo/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecovery_PanicReturns500(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("something went wrong")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "internal_error", resp["code"])
	errs, ok := resp["errors"].([]any)
	require.True(t, ok)
	require.Len(t, errs, 1)
	entry := errs[0].(map[string]any)
	assert.Equal(t, "internal server error", entry["message"])
}

func TestRecovery_LogsPanicWithStack(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))

	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "panic recovered", entry["msg"])
	assert.Equal(t, "boom", entry["panic"])
	assert.NotEmpty(t, entry["stack"])
}

func TestRecovery_NormalRequestUnaffected(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ok"`)
}

func TestRecovery_IncludesRequestID(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("with-id")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Contains(t, entry, "request_id")
}
