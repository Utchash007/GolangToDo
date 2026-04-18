package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"GolangToDo/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	maxRetries    = 5
	retryInterval = 2 * time.Second
	pingTimeout   = 5 * time.Second
)

func Connect(cfg *config.Config) (*sqlx.DB, error) {
	database, err := sqlx.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	database.SetMaxOpenConns(cfg.MaxOpenConns)
	database.SetMaxIdleConns(cfg.MaxIdleConns)
	database.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
		err = database.PingContext(ctx)
		cancel()

		if err == nil {
			slog.Info("database connection established")
			return database, nil
		}

		slog.Warn("database ping failed", "attempt", attempt, "of", maxRetries, "error", err)

		if attempt < maxRetries {
			time.Sleep(retryInterval)
		}
	}

	database.Close()
	return nil, fmt.Errorf("could not connect to database after %d attempts: %w", maxRetries, err)
}
