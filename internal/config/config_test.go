package config_test

import (
	"testing"
	"time"

	"GolangToDo/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_RequiredVars(t *testing.T) {
	t.Run("loads required vars", func(t *testing.T) {
		t.Setenv("PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/todo")

		cfg, err := config.Load()

		require.NoError(t, err)
		assert.Equal(t, "8080", cfg.Port)
		assert.Equal(t, "postgres://user:pass@localhost:5432/todo", cfg.DatabaseURL)
	})

	t.Run("missing PORT", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/todo")

		_, err := config.Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "PORT")
	})

	t.Run("missing DATABASE_URL", func(t *testing.T) {
		t.Setenv("PORT", "8080")

		_, err := config.Load()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "DATABASE_URL")
	})

	t.Run("both missing", func(t *testing.T) {
		_, err := config.Load()

		require.Error(t, err)
	})
}

func TestLoad_PoolDefaults(t *testing.T) {
	t.Run("DB_MAX_OPEN_CONNS absent uses default", func(t *testing.T) {
		t.Setenv("PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/todo")

		cfg, err := config.Load()

		require.NoError(t, err)
		assert.Equal(t, 25, cfg.MaxOpenConns)
	})

	t.Run("DB_MAX_IDLE_CONNS absent uses default", func(t *testing.T) {
		t.Setenv("PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/todo")

		cfg, err := config.Load()

		require.NoError(t, err)
		assert.Equal(t, 5, cfg.MaxIdleConns)
	})

	t.Run("DB_CONN_MAX_IDLE_TIME absent uses default", func(t *testing.T) {
		t.Setenv("PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/todo")

		cfg, err := config.Load()

		require.NoError(t, err)
		assert.Equal(t, 5*time.Minute, cfg.ConnMaxIdleTime)
	})

	t.Run("DB_MAX_OPEN_CONNS unparseable uses default", func(t *testing.T) {
		t.Setenv("PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/todo")
		t.Setenv("DB_MAX_OPEN_CONNS", "abc")

		cfg, err := config.Load()

		require.NoError(t, err)
		assert.Equal(t, 25, cfg.MaxOpenConns)
	})
}

func TestLoad_PoolVars(t *testing.T) {
	t.Run("applies all valid pool vars", func(t *testing.T) {
		t.Setenv("PORT", "8080")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/todo")
		t.Setenv("DB_MAX_OPEN_CONNS", "10")
		t.Setenv("DB_MAX_IDLE_CONNS", "3")
		t.Setenv("DB_CONN_MAX_IDLE_TIME", "10m")

		cfg, err := config.Load()

		require.NoError(t, err)
		assert.Equal(t, 10, cfg.MaxOpenConns)
		assert.Equal(t, 3, cfg.MaxIdleConns)
		assert.Equal(t, 10*time.Minute, cfg.ConnMaxIdleTime)
	})
}
