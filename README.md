# GolangToDo

A RESTful ToDo API built with Go. Manage tasks with priority levels, categories, and completion state. Supports full CRUD, filtering, pagination, and bulk operations.

## Tech Stack

| Layer | Library |
|---|---|
| HTTP Framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL |
| DB Driver | [jackc/pgx/v5](https://github.com/jackc/pgx) |
| DB Toolkit | [sqlx](https://github.com/jmoiron/sqlx) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) |
| Validation | [go-playground/validator](https://github.com/go-playground/validator) |
| Logging | `log/slog` (stdlib) |
| Testing | [testify](https://github.com/stretchr/testify) |
| Config | [godotenv](https://github.com/joho/godotenv) |

## Task Entity

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Application-generated (v4) |
| `title` | string | Required, non-empty |
| `priority` | `low` / `medium` / `high` | PostgreSQL native enum |
| `category` | string | Optional, normalized to lowercase |
| `completed` | bool | Defaults to `false` |
| `created_at` | timestamp | Set on create |
| `updated_at` | timestamp | Updated on every write |

## Prerequisites

- Go 1.22+
- PostgreSQL 14+
- [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/Utchash007/GolangToDo.git
cd GolangToDo
```

### 2. Configure environment

```bash
cp .env.example .env
```

Edit `.env` and fill in your values:

```env
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/gotodo?sslmode=disable

# Optional — connection pool tuning
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_IDLE_TIME=5m
```

### 3. Run database migrations

```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```

### 4. Run the server

```bash
go run ./cmd/api
```

The server starts on the port defined in `.env` (default `8080`).

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check — returns DB connectivity status |

> More endpoints (Task CRUD, filtering, bulk ops) are coming in upcoming features.

## Common Commands

```bash
# Run all tests
go test ./...

# Run tests with race detector
go test -race ./...

# Build binary
go build -o bin/api ./cmd/api

# Roll back last migration
migrate -path ./migrations -database "$DATABASE_URL" down 1

# Lint
golangci-lint run ./...
```

## Project Structure

```
cmd/api/main.go              # Entry point — wires config → db → router → server
internal/
  config/config.go           # Env var loading and validation
  db/db.go                   # PostgreSQL connection pool (pgx + sqlx)
  router/router.go           # Gin router and route registration
  middleware/
    logger.go                # Structured request logging (slog)
    request_id.go            # Per-request UUID injection
migrations/
  001_create_tasks.up.sql    # Creates priority enum, tasks table, indexes
  001_create_tasks.down.sql  # Reverses the above migration
.env.example                 # Documents all supported env vars
```

## Architecture

Clean Architecture with four layers:

```
Handler → Service → Repository → PostgreSQL
```

Each layer depends only on the layer below it via interfaces. The `Repository` interface is the only test seam — tests mock the repository, not the database.

## Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

Returns `503` with `{"status":"degraded"}` if the database is unreachable.
