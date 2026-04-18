package db_test

import (
	"os"
	"testing"

	"GolangToDo/internal/config"
	"GolangToDo/internal/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig builds a Config from environment variables.
// Integration tests are skipped if DATABASE_URL is not set.
func testConfig(t *testing.T) *config.Config {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	t.Setenv("PORT", "8080")
	cfg, err := config.Load()
	require.NoError(t, err)
	return cfg
}

func TestConnect_HappyPath(t *testing.T) {
	t.Run("connects on first attempt", func(t *testing.T) {
		cfg := testConfig(t)

		database, err := db.Connect(cfg)

		require.NoError(t, err)
		require.NotNil(t, database)
		database.Close()
	})

	t.Run("pool settings applied", func(t *testing.T) {
		if os.Getenv("DATABASE_URL") == "" {
			t.Skip("DATABASE_URL not set — skipping integration test")
		}
		t.Setenv("PORT", "8080")
		t.Setenv("DB_MAX_OPEN_CONNS", "10")
		t.Setenv("DB_MAX_IDLE_CONNS", "3")
		cfg, err := config.Load()
		require.NoError(t, err)

		database, err := db.Connect(cfg)

		require.NoError(t, err)
		require.NotNil(t, database)
		assert.Equal(t, 10, database.Stats().MaxOpenConnections)
		database.Close()
	})
}

func TestConnect_Retry(t *testing.T) {
	t.Run("returns error after all retries exhausted", func(t *testing.T) {
		t.Setenv("PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://invalid:invalid@localhost:9999/nonexistent")
		cfg, err := config.Load()
		require.NoError(t, err)

		_, err = db.Connect(cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "5 attempts")
	})
}
