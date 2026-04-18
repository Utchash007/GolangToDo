package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 5
	defaultConnMaxIdleTime = 5 * time.Minute
)

type Config struct {
	Port            string
	DatabaseURL     string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		return nil, fmt.Errorf("required env var PORT is not set")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("required env var DATABASE_URL is not set")
	}

	return &Config{
		Port:            port,
		DatabaseURL:     databaseURL,
		MaxOpenConns:    parseIntEnv("DB_MAX_OPEN_CONNS", defaultMaxOpenConns),
		MaxIdleConns:    parseIntEnv("DB_MAX_IDLE_CONNS", defaultMaxIdleConns),
		ConnMaxIdleTime: parseDurationEnv("DB_CONN_MAX_IDLE_TIME", defaultConnMaxIdleTime),
	}, nil
}

func parseIntEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func parseDurationEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
