package task

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustConnectDB(t *testing.T) *sqlx.DB {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := sqlx.Connect("pgx", databaseURL)
	require.NoError(t, err)
	return db
}

func TestRepository_Create(t *testing.T) {
	db := mustConnectDB(t)
	repo := NewRepository(db)

	task := NewTask("Test Task", PriorityMedium, "work")
	err := repo.Create(context.Background(), task)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, task.ID)
}