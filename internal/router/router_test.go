package router_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"GolangToDo/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockPinger struct{ err error }

func (m *mockPinger) PingContext(_ context.Context) error { return m.err }

func TestHealth_Healthy(t *testing.T) {
	r := gin.New()
	r.GET("/health", router.HealthHandler(&mockPinger{}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

func TestHealth_Degraded(t *testing.T) {
	r := gin.New()
	r.GET("/health", router.HealthHandler(&mockPinger{err: errors.New("db down")}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.JSONEq(t, `{"status":"degraded"}`, w.Body.String())
}

func TestUnknownRoute(t *testing.T) {
	r := gin.New()
	r.GET("/health", router.HealthHandler(&mockPinger{}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	assert.Equal(t, http.StatusNotFound, w.Code)
}
