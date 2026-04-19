# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A RESTful ToDo API built with Go. Single `Task` entity, no authentication. Full project plan is in `ARCHITECTURE.md`.

## Tech Stack

| Layer | Choice |
|---|---|
| HTTP Framework | Gin |
| Database | PostgreSQL |
| DB Driver | jackc/pgx/v5 |
| DB Toolkit | sqlx |
| Migrations | golang-migrate/migrate |
| Validation | go-playground/validator |
| Logging | log/slog (stdlib) |
| Testing | stretchr/testify |
| Config | joho/godotenv |

## Common Commands

```bash
# Run the server (requires .env with PORT and DATABASE_URL)
go run ./cmd/api

# Build binary
go build -o bin/api ./cmd/api

# Run all tests (DB integration tests skip if DATABASE_URL is not set)
go test ./...

# Run a single test
go test ./internal/config/... -run TestLoad_RequiredVars

# Run tests with race detector
go test -race ./...

# Apply database migrations (up)
# NOTE: golang-migrate CLI does not work with Nile (hosted PostgreSQL) — apply SQL directly:
psql "$DATABASE_URL" -f migrations/001_create_tasks.up.sql

# Roll back last migration
psql "$DATABASE_URL" -f migrations/001_create_tasks.down.sql

# Lint
golangci-lint run ./...
```

## Directory Layout

```
cmd/api/main.go                  # Entry point — wires config → db → router → server
internal/
  config/config.go               # Load() validates env vars, returns Config struct
  db/db.go                       # Connect() opens sqlx+pgx pool, retries 5×2s
  router/router.go               # New(*sqlx.DB) registers all routes; HealthHandler(Pinger) exported
  middleware/
    request_id.go                # UUID v4 per request → context + X-Request-ID header
    logger.go                    # slog request logging (method, path, status, latency_ms)
  task/
    model.go                     # Task struct, Priority enum (Value/Scan/JSON), DTOs, ValidationError
    repository.go                # Repository interface + sqlx impl; 5s timeout on every DB call
    service.go                   # Business logic, category normalization, ValidationError returns
    handler.go                   # Gin handlers; handleError maps ValidationError→400, rest→500
migrations/
  001_create_tasks.up.sql        # priority ENUM, tasks table, 3 BTREE indexes
  001_create_tasks.down.sql      # reverse migration
specs/plans/                     # Feature plan documents (PLAN-*.md)
.env.example                     # Documents all supported env vars
```

## Architecture

Clean Architecture — 4 layers: Handler → Service → Repository → PostgreSQL.

Each layer depends only on the layer below it via interfaces. The `Repository` interface lives in `internal/task/` and is the only seam used for testing (mock the repo, not the DB).

## Task Entity

```go
Task {
    id         uuid
    title      string
    priority   enum(low, medium, high)  // iota starts at 1; 0 = unknown
    category   string                   // free-text, normalized to lowercase on write
    completed  bool
    created_at time.Time
    updated_at time.Time
}
```

## API Operations

- CRUD on Task
- Filter by priority / category / completed
- Pagination (offset-based)
- Bulk complete and bulk delete

## Key Conventions

- Functional options for constructors
- Repository pattern — all DB access behind interfaces
- Manual constructor injection (no DI container)
- Early return, flat happy path
- `context.Context` propagated through all layers with DB timeouts
- `defer rows.Close()` immediately after every query
- Single `ErrorResponse{code, message}` shape for all errors
- Enum types start at 1 — zero value means unknown/unset
- Narrow interfaces for testability — e.g. `Pinger` in `internal/router` accepts any type with `PingContext(ctx) error`, not `*sqlx.DB` directly
- DB integration tests use `t.Skip` when `DATABASE_URL` is unset — safe to run in CI without a real DB

## Delivery Phases

1. **Foundation** — Bootstrap, DB connection, migrations, observability (F1 ✅, F2 ✅, F7 ✅)
2. **Core API** — Task CRUD + validation/error handling (F3 ✅, F6)
3. **Advanced Queries** — Filtering, pagination, bulk ops (F4, F5)

## Skills Available

Installed via `.agents/skills/`. Invoke with `/skill-name`.

**Workflow:**
- `plan-project` — top-level project planning (done)
- `plan-feature` — plan individual features before implementation
- `generate-tasks` — break a feature plan into TDD tasks
- `start-task` — begin a task from the task list
- `tdd` — implement tasks via RED-GREEN-REFACTOR
- `commit` — conventional commits
- `review` — code review
- `create-worktrees` — isolated git worktrees for parallel work

**Go — language & patterns:**
- `golang-code-style` — formatting and conventions
- `golang-naming` — naming conventions
- `golang-design-patterns` — functional options, graceful shutdown, resilience
- `golang-structs-interfaces` — struct/interface design, embedding, receivers
- `golang-error-handling` — wrapping, sentinel errors, logging
- `golang-concurrency` — goroutines, channels, worker pools
- `golang-context` — context propagation, cancellation, timeouts
- `golang-safety` — nil panics, numeric conversions, resource lifecycle
- `golang-modernize` — upgrade to modern Go idioms

**Go — infrastructure:**
- `golang-database` — sqlx queries, scanning, transactions, connection pool
- `golang-testing` — table-driven tests, mocks, integration tests, goleak
- `golang-stretchr-testify` — assert/require/mock/suite
- `golang-observability` — slog, Prometheus, OpenTelemetry, pprof
- `golang-security` — injection, crypto, secrets, input handling
- `golang-lint` — golangci-lint configuration and suppressions
- `golang-continuous-integration` — GitHub Actions, coverage, releases
- `golang-dependency-management` — go.mod, upgrades, vulnerability scanning
- `golang-project-layout` — directory structure for Go projects

**Go — reference:**
- `golang-benchmark` — pprof, benchstat, CI regression detection
- `golang-performance` — allocation reduction, GC tuning, hot-path optimization
- `golang-troubleshooting` — debugging, race detection, GODEBUG tracing
- `golang-popular-libraries` — vetted library recommendations
- `golang-data-structures` — slices, maps, generics, container packages

## Git Workflow

- Branch per feature: `feature/f<n>-<slug>` (e.g. `feature/f1-project-bootstrap`)
- Implement all tasks for a feature on the same branch
- Open a PR into `main` when the feature is complete — never commit directly to `main`
- Commit message style: `conventional commits` (use the `commit` skill)

## Repository

GitHub: https://github.com/Utchash007/GolangToDo
