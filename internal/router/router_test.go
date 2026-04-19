package router_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"GolangToDo/internal/router"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func mustConnectDB(t *testing.T) *sqlx.DB {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := sqlx.Connect("pgx", databaseURL)
	if err != nil {
		t.Skip("cannot connect to DB")
	}
	return db
}

func TestHealth(t *testing.T) {
	db := mustConnectDB(t)
	r := router.New(db)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUnknownRoute(t *testing.T) {
	db := mustConnectDB(t)
	r := router.New(db)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	assert.Equal(t, http.StatusNotFound, w.Code)
}
