# Project Plan: Go ToDo REST API

> **Date:** 2026-04-18
> **Last updated:** 2026-04-19
> **Type:** Greenfield
> **Estimated features:** 7
> **Estimated phases:** 3
> **Phase 1 status:** Complete ✅
> **Phase 2 status:** In Progress 🔄

## Project Summary

A RESTful ToDo API built with Go, using Gin as the HTTP framework and PostgreSQL as the database. It manages Tasks with priority, category, and completion state. The API supports full CRUD, filtering, bulk operations, and pagination — with no authentication layer for v1.

## System Boundaries

### In Scope
- Task CRUD (create, read, update, delete)
- Filter tasks by priority, category, and completed status
- Bulk complete and bulk delete operations
- Pagination on list endpoints
- Database schema migrations

### Out of Scope
- User authentication / authorization (reason: v1 scope decision)
- Multi-tenancy / per-user task isolation (reason: no auth in v1)
- File attachments or rich content (reason: not required)
- Real-time updates / WebSockets (reason: not required)
- Frontend / UI (reason: API only)

### External Integrations
- PostgreSQL — primary data store, accessed via sqlx + pgx/v5 driver

---

## Architecture Direction

### High-Level Structure

```
┌─────────────────────────────────────────────┐
│                   HTTP Layer                │
│         Gin Router + Middleware             │
└────────────────────┬────────────────────────┘
                     │
┌────────────────────▼────────────────────────┐
│               Handler Layer                 │
│     Request binding, validation, response   │
└────────────────────┬────────────────────────┘
                     │
┌────────────────────▼────────────────────────┐
│               Service Layer                 │
│       Business logic, bulk operations       │
└────────────────────┬────────────────────────┘
                     │
┌────────────────────▼────────────────────────┐
│             Repository Layer                │
│     sqlx queries, filters, pagination       │
└────────────────────┬────────────────────────┘
                     │
┌────────────────────▼────────────────────────┐
│               PostgreSQL                    │
└─────────────────────────────────────────────┘
```

### Key Technology Choices

| Choice | Decision | Rationale |
|--------|----------|-----------|
| Language | Go | Compiled, performant, strong stdlib |
| HTTP Framework | Gin | Fast, production-ready, minimal overhead |
| Database | PostgreSQL | Reliable, feature-rich relational DB |
| DB Driver | jackc/pgx/v5 | Faster and more maintained than lib/pq |
| DB Toolkit | sqlx | Thin wrapper for struct scanning, no ORM magic |
| Migrations | golang-migrate/migrate | SQL file-based, CLI + Go library, pgx/v5 support |
| Validation | go-playground/validator | Struct tag validation, native Gin integration |
| Logging | log/slog (stdlib) | Structured logging, no extra dependency |
| Testing | stretchr/testify | Assertions and mocks |
| Config | joho/godotenv | Simple .env loading |

### Patterns & Conventions

- **Functional options** — used for constructors that accept configuration
- **Repository pattern** — all DB access behind interfaces, enabling easy testing
- **Dependency injection** — manual constructor injection (no DI container needed at this scale)
- **Early return / flat happy path** — error cases handled first, minimal nesting
- **Enum types starting at 1** — zero value reserved for unknown/unset
- **Context propagation** — `context.Context` passed through all layers
- **Timeout on every DB call** — via context with timeout
- **defer rows.Close()** immediately after every query

---

## Data Models

### Task

| Column | PostgreSQL Type | Constraints |
|--------|----------------|-------------|
| `id` | `UUID` | PRIMARY KEY, NOT NULL — generated application-side via `google/uuid` |
| `title` | `TEXT` | NOT NULL, CHECK (`title <> ''`) |
| `priority` | `priority` (native ENUM) | NOT NULL — values: `low`, `medium`, `high` |
| `category` | `TEXT` | NULL allowed — normalized to lowercase on write |
| `completed` | `BOOLEAN` | NOT NULL, DEFAULT `false` |
| `created_at` | `TIMESTAMPTZ` | NOT NULL — set by application at insert |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL — set by application on every write |

**Indexes:** `idx_tasks_priority`, `idx_tasks_category`, `idx_tasks_completed` (all BTREE, single-column)

**Key decisions:**
- Native PostgreSQL ENUM enforces valid priority values at the DB level; priority set is stable so ALTER ENUM cost is acceptable
- `updated_at` is application-side, not a DB trigger — keeps migration simple and makes timestamps explicit in the service layer
- `category` is nullable (not empty string) to distinguish "not set" from "empty"
- `id` generated in Go so the application knows it before the INSERT (no extra SELECT round-trip)

---

## Feature Map

### Feature List

| # | Feature | Type | Description | Dependencies |
|---|---------|------|-------------|--------------|
| F1 ✅ | Project Bootstrap | Infrastructure | Go module, folder structure, config loading, DB connection, migration runner | None |
| F2 ✅ | Database Schema | Infrastructure | Tasks table migration, indexes on priority/category/completed | F1 |
| F7 ✅ | Observability & Graceful Shutdown | Cross-cutting | Structured slog logging, request logging middleware, graceful shutdown on SIGTERM | F1 |
| F3 ✅ | Task CRUD | Core | Create, read (single + list), update, delete endpoints | F2 |
| F6 | Input Validation & Error Handling | Cross-cutting | Struct validation, consistent error response shape, 404/400/500 handling | F3 |
| F4 | Filtering & Pagination | Core | Filter list by priority/category/completed, offset pagination | F3 |
| F5 | Bulk Operations | Core | Bulk complete and bulk delete by list of IDs | F3 |

### Feature Dependencies

```
F1 (Bootstrap)
└── F2 (Schema)
    └── F3 (Task CRUD)
        ├── F4 (Filtering & Pagination)
        ├── F5 (Bulk Operations)
        └── F6 (Validation & Error Handling)
F1
└── F7 (Observability & Graceful Shutdown)
```

### Cross-Cutting Concerns

- **Error responses** — affects F3/F4/F5, strategy: single `ErrorResponse` struct with `code` + `message`
- **Context + timeouts** — affects all DB calls, strategy: wrap every handler context with a timeout
- **Logging** — affects all layers, strategy: slog with request ID per request via middleware

---

## Delivery Phases

### Phase 1: Foundation ✅ Complete
**Goal:** A running Go HTTP server connected to PostgreSQL with migrations applied
**Features:** F1 ✅, F2 ✅, F7 ✅
**Delivered:**
- Go module, config loading (`internal/config`), PostgreSQL connection pool with retries (`internal/db`)
- Gin server with graceful shutdown on SIGTERM/SIGINT (`cmd/api/main.go`)
- `GET /health` endpoint with DB ping
- `tasks` table migration — `priority` enum, all columns, 3 BTREE indexes (`migrations/001_create_tasks.*`)
- `github.com/google/uuid` added for application-side UUID generation (consumed by F3)
- Structured JSON logging via `log/slog`, per-request UUID middleware (`internal/middleware`)

### Phase 2: Core API — In Progress 🔄
**Goal:** Full CRUD on Tasks with validation and consistent error handling
**Features:** F3 ✅, F6
**Depends on:** Phase 1 ✅
**Delivered so far:**
- `internal/task/` — Handler → Service → Repository wiring, all 5 endpoints
- `Priority` type with `driver.Valuer` + `sql.Scanner` for PostgreSQL ENUM, `MarshalJSON`/`UnmarshalJSON` for string JSON
- `ValidationError` type — service returns typed errors, handler maps to 400/404/500
- `context.WithTimeout(5s)` on every repository DB call
- `GET /tasks` returns `[]` (not `null`) on empty result
- `router.HealthHandler` exported for unit-testable mock injection
**Risk:** Validation edge cases on priority/category enum types

### Phase 3: Advanced Queries
**Goal:** Filterable, paginated task list and bulk operations
**Features:** F4, F5
**Depends on:** Phase 2 complete
**Risk:** Pagination strategy (offset vs keyset) — offset is simpler but keyset scales better

---

## Decisions Log

| # | Decision | Alternatives Considered | Chosen Because |
|---|----------|------------------------|----------------|
| 1 | sqlx over GORM | GORM, sqlc | sqlx gives raw SQL control with minimal boilerplate; sqlc requires codegen step |
| 2 | pgx/v5 as driver | lib/pq | pgx/v5 is actively maintained, faster, and officially recommended for new projects |
| 3 | golang-migrate for migrations | goose, atlas, GORM AutoMigrate | File-based SQL migrations, framework-agnostic, widely adopted |
| 4 | Clean Architecture (layered) | Flat layout, Hexagonal | Right balance of structure and simplicity for this project size |
| 5 | No authentication in v1 | JWT, API keys | Out of scope for v1; can be added as F8 in a future phase |
| 6 | slog (stdlib) for logging | zap, zerolog, samber/slog-* | Zero dependency, sufficient for this scale |

## Open Questions

- **Pagination strategy:** Offset-based (simple) vs keyset/cursor (scalable)?
  - **Impact if unresolved:** Offset is fine for small datasets; cursor needed if tasks grow large
  - **Decision:** Offset pagination — simpler to implement, sufficient for v1; keyset noted as future upgrade

- **Category type:** ~~Free-text string vs predefined enum?~~ **Resolved:** Free-text with lowercase normalization on write — schema uses nullable `TEXT`, application lowercases before every insert/update.

---

## Next Steps

Phase 1 is complete. Phase 2 is next.

**Remaining features — plan each before implementing:**

1. ~~**F1: Project Bootstrap**~~ ✅
2. ~~**F2: Database Schema**~~ ✅
3. ~~**F7: Observability & Graceful Shutdown**~~ ✅
4. ~~**F3: Task CRUD**~~ ✅
5. **F6: Input Validation & Error Handling** — error response shape, validator integration with Gin ← **start here**
6. **F4: Filtering & Pagination** — dynamic filter query builder, offset pagination
7. **F5: Bulk Operations** — bulk complete + bulk delete, transaction handling

Start with: `/plan-feature for: Task CRUD (from ARCHITECTURE.md, feature F3)`

---
_This project plan is the input for individual plan-feature sessions._
_Each feature listed above should be planned separately before task generation._
