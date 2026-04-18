# Plan: Project Bootstrap

> **Date:** 2026-04-18
> **Project source:** ARCHITECTURE.md (Feature F1)
> **Estimated tasks:** 6–8
> **Planning session:** detailed

## Summary

Stand up the Go project skeleton that every other feature builds on: module init, folder layout, config loading, DB connection pool with retry, and a running HTTP server. No business logic — just a provably working foundation that future layers can plug into.

## Requirements

### Functional Requirements

1. `go run ./cmd/api` starts an HTTP server on the port specified by `PORT`
2. All required env vars are validated at startup; any missing var exits the process immediately with a descriptive error
3. A `sqlx` + `pgx/v5` connection pool is established using `DATABASE_URL`
4. If the DB is unreachable at startup, the app retries 5 times with 2-second intervals before exiting with a clear error
5. Pool limits are read from env vars with hardcoded defaults when the var is absent
6. A `GET /health` endpoint returns `200 OK` and confirms the DB is reachable (ping)
7. A `.env.example` file documents every supported env var with placeholder values
8. Database migrations are managed via the `golang-migrate` CLI — the app itself never runs migrations

### Non-Functional Requirements

1. The app must start in under 3 seconds under normal conditions
2. Startup errors must name the missing/invalid var and tell the developer what to fix
3. DB retry attempts must be logged so the developer can see what's happening

## Behaviors

**Why fail-fast on missing env vars matters:**
Silent fallbacks (e.g., defaulting `DATABASE_URL` to `localhost`) mask misconfiguration and cause confusing runtime failures far from the actual mistake. Explicit exits at startup force the problem to be fixed before the app runs.

**Why retry DB connection instead of failing immediately:**
In Docker Compose or CI, the DB container often starts 1–3 seconds after the app container. An immediate exit would make the app permanently unusable in those environments without external orchestration (e.g., `depends_on: condition: service_healthy`).

**What's optional vs required:**
- `PORT` is required — no sensible default avoids port conflicts
- `DATABASE_URL` is required — no DB means nothing works
- Pool settings (`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_IDLE_TIME`) are optional — defaults apply silently

**Common mistakes:**
- Forgetting to call `db.Close()` — use `defer db.Close()` in `main`
- Checking only that `sql.Open()` succeeds — it doesn't actually connect; always follow with `db.PingContext()`
- Not propagating `context.Context` into the ping — use a timeout context so a hung DB doesn't block startup forever

## Detailed Specifications

### Config Loading

**Purpose:** Load env vars from `.env` (if present) and validate required ones are set.

**Behavior:**
- Call `godotenv.Load()` at the very start of `main`; if `.env` is absent, continue silently (env vars may be injected by the runtime)
- After loading, validate that `PORT` and `DATABASE_URL` are non-empty strings; if either is missing, print a human-readable error naming the var and call `os.Exit(1)`
- Pool setting vars (`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_IDLE_TIME`) are parsed as integers/durations; if absent or unparseable, log a warning and apply the default

**Defaults:**
| Env Var | Default |
|---|---|
| `DB_MAX_OPEN_CONNS` | `25` |
| `DB_MAX_IDLE_CONNS` | `5` |
| `DB_CONN_MAX_IDLE_TIME` | `5m` |

**Error Scenarios:**
| Condition | Expected Behavior |
|-----------|-------------------|
| `PORT` not set | Print `"required env var PORT is not set"`, exit 1 |
| `DATABASE_URL` not set | Print `"required env var DATABASE_URL is not set"`, exit 1 |
| Pool var present but not a valid integer | Log warning with var name and value, use default, continue |

---

### DB Connection Pool

**Purpose:** Establish a reusable `*sqlx.DB` configured with pgx/v5 driver and pool limits.

**Behavior:**
- Open connection using `sqlx.Open("pgx", DATABASE_URL)`
- Apply pool settings from config
- Ping the DB using a context with a 5-second timeout
- On ping failure: log the attempt number and error, wait 2 seconds, retry
- After 5 failed attempts: log a final error and `os.Exit(1)`
- On success: log `"database connection established"`

**Error Scenarios:**
| Condition | Expected Behavior |
|-----------|-------------------|
| DB unreachable on first ping | Log attempt 1/5, wait 2s, retry |
| DB unreachable after 5 pings | Log `"could not connect to database after 5 attempts"`, exit 1 |
| Invalid `DATABASE_URL` format | `sqlx.Open` or first ping returns error → treated same as unreachable |

---

### HTTP Server

**Purpose:** Start a Gin engine listening on `PORT`.

**Behavior:**
- Create a `gin.New()` instance (not `gin.Default()` — middleware added explicitly)
- Register `GET /health` route that pings the DB and returns `{"status":"ok"}` on success or `{"status":"degraded"}` with `503` if ping fails
- Call `router.Run(":" + port)` — blocks until exit

**Error Scenarios:**
| Condition | Expected Behavior |
|-----------|-------------------|
| Port already in use | Gin returns error from `Run`; log it and exit 1 |
| `/health` DB ping fails | Return `503` with `{"status":"degraded"}` |

---

### Migration CLI Setup

**Purpose:** Document how migrations are managed — the app has no migration code.

**Behavior:**
- Migrations live in `./migrations/` as `*.up.sql` / `*.down.sql` files
- Developers run: `migrate -path ./migrations -database "$DATABASE_URL" up`
- The app does not call migrate at startup under any circumstances
- An empty `migrations/` directory is created as part of bootstrap (ready for F2)

## Key Constraints

| Constraint | Why It Matters |
|------------|----------------|
| App never runs migrations | Mixing startup migration with app logic causes race conditions in multi-replica deployments and makes rollbacks unsafe |
| `gin.New()` not `gin.Default()` | `gin.Default()` registers Logger and Recovery middleware automatically; we add middleware explicitly so we control what runs and in what order |
| Always ping after open | `sqlx.Open` / `sql.Open` never dials the DB — a successful open with an invalid URL will only fail on first use |

## Edge Cases & Failure Modes

| Scenario | Decision | Rationale |
|----------|----------|-----------|
| `.env` file absent | Continue silently | Env vars may be injected by Docker/CI — absence of file is not an error |
| `DB_CONN_MAX_IDLE_TIME` is `"abc"` | Warn + use default | Unparseable optional config should not crash the app |
| DB comes up on retry attempt 3 | Connect successfully, log attempt number, proceed | Retry loop exits as soon as ping succeeds |
| `/health` called while DB is down | Return `503 {"status":"degraded"}` | Health check must reflect real DB state for load balancer use |

## Decisions Log

| # | Decision | Alternatives Considered | Chosen Because |
|---|----------|------------------------|----------------|
| 1 | `gin.New()` over `gin.Default()` | `gin.Default()` | Explicit middleware registration; aligns with project convention of controlled layering |
| 2 | 5 retries × 2s for DB | Immediate fail; exponential backoff | Simple and sufficient for Docker Compose; exponential backoff is over-engineering at this scale |
| 3 | `DATABASE_URL` as single var | Separate host/port/name/user/pass vars | Single URL is standard (12-factor app), easier to copy between environments |
| 4 | Pool settings via env vars with defaults | Hardcoded only; env vars only | Env vars allow per-environment tuning; defaults prevent mandatory config for simple local dev |
| 5 | CLI-only migrations | Embedded on startup | Avoids race conditions in multi-replica deployments; gives explicit operator control |

## Scope Boundaries

### In Scope
- Go module init (`go mod init GolangToDo`)
- Folder structure creation
- Config loading and validation
- DB connection pool with retry
- Gin HTTP server on `PORT`
- `GET /health` endpoint
- `.env.example` with all vars documented
- Empty `migrations/` directory

### Out of Scope
- Any Task-related routes or handlers (→ F3)
- Actual migration SQL files (→ F2)
- Request logging middleware (→ F7)
- Graceful shutdown on SIGTERM (→ F7)
- Authentication (→ out of v1 scope)

## Dependencies

### Depends On
- None — this is the foundation

### Depended On By
- F2 (Database Schema) — needs the migrations directory and DB connection
- F3 (Task CRUD) — needs the Gin server and DB pool
- F7 (Observability) — needs the Gin engine to attach middleware

## Architecture Notes

Layer structure mirrors the project's clean architecture:
```
cmd/api/main.go        → wires everything together, owns process lifecycle
internal/config/       → loads and validates env vars, exposes a Config struct
internal/db/           → opens pool, runs ping-with-retry, returns *sqlx.DB
```

`main.go` calls `config.Load()` → `db.Connect(cfg)` → `router.New(db)` → `router.Run()`. Each step returns an error or panics early — no partial startup.

## Open Questions

- None — all decisions resolved in planning session.

---
_This plan is the input for the generate-tasks skill._
_Review this document, then run: `/generate-tasks` from plan: `specs/plans/PLAN-project-bootstrap.md`_

---

# Tasks

## Task T1: Config Loading & Validation

> **Status:** done
> **Effort:** s
> **Priority:** critical
> **Depends on:** None

### Description

Implement `internal/config/config.go` — load `.env` via `joho/godotenv`, validate that `PORT` and `DATABASE_URL` are present, and parse optional pool settings with hardcoded defaults. Returns a `Config` struct or an error. This is the first thing `main.go` calls.

### Test Plan

#### Test File(s)
- `internal/config/config_test.go`

#### Test Scenarios

##### Config — Happy Path
- **loads required vars** — GIVEN `PORT` and `DATABASE_URL` are set in env WHEN `Load()` is called THEN returns a populated `Config` with no error
- **applies all valid pool vars** — GIVEN all pool env vars set to valid integers/durations WHEN `Load()` is called THEN `Config` fields reflect those values exactly

##### Config — Required Var Validation
- **missing PORT** — GIVEN `PORT` is not set WHEN `Load()` is called THEN returns an error whose message contains `"PORT"`
- **missing DATABASE_URL** — GIVEN `DATABASE_URL` is not set WHEN `Load()` is called THEN returns an error whose message contains `"DATABASE_URL"`
- **both missing** — GIVEN neither `PORT` nor `DATABASE_URL` is set WHEN `Load()` is called THEN returns an error naming at least one missing var

##### Config — Optional Pool Vars with Defaults
- **DB_MAX_OPEN_CONNS absent** — GIVEN var is not set WHEN `Load()` is called THEN `Config.MaxOpenConns` equals `25`
- **DB_MAX_IDLE_CONNS absent** — GIVEN var is not set WHEN `Load()` is called THEN `Config.MaxIdleConns` equals `5`
- **DB_CONN_MAX_IDLE_TIME absent** — GIVEN var is not set WHEN `Load()` is called THEN `Config.ConnMaxIdleTime` equals `5 * time.Minute`
- **DB_MAX_OPEN_CONNS unparseable** — GIVEN var is set to `"abc"` WHEN `Load()` is called THEN returns no error and `Config.MaxOpenConns` equals `25`

### Implementation Notes

- **Layer:** `internal/config/`
- **Libraries:** `github.com/joho/godotenv`, stdlib `os`, `strconv`, `time`
- **Key decisions:** `godotenv.Load()` failure (file absent) is silently ignored — env vars may be injected by runtime. Only missing required vars produce errors.
- **Config struct fields:** `Port string`, `DatabaseURL string`, `MaxOpenConns int`, `MaxIdleConns int`, `ConnMaxIdleTime time.Duration`

### Scope Boundaries

- Do NOT connect to the database — config only reads and validates env vars
- Do NOT call `os.Exit` — return errors; `main.go` owns process lifecycle
- Do NOT add any vars beyond those in the plan (PORT, DATABASE_URL, three pool settings)
- Only implement `Load() (*Config, error)` — no sub-commands, no watchers

### Files Expected

**New files:**
- `internal/config/config.go`
- `internal/config/config_test.go`

**Must NOT modify:**
- Any other file — this task is purely additive

---

## Task T2: DB Connection Pool with Retry

> **Status:** not started
> **Effort:** m
> **Priority:** critical
> **Depends on:** T1

### Description

Implement `internal/db/db.go` — open a `sqlx` + `pgx/v5` connection pool using `Config`, apply pool settings, and ping the DB with a 5-second timeout per attempt. Retry up to 5 times with 2-second gaps before returning an error. Returns `*sqlx.DB` on success.

### Test Plan

#### Test File(s)
- `internal/db/db_test.go` (integration — requires a real PostgreSQL instance)

#### Test Scenarios

##### DB Connection — Happy Path
- **connects on first attempt** — GIVEN a valid `DATABASE_URL` pointing to a running DB WHEN `Connect(cfg)` is called THEN returns a non-nil `*sqlx.DB` and no error
- **pool settings applied** — GIVEN config with specific `MaxOpenConns`, `MaxIdleConns`, `ConnMaxIdleTime` WHEN `Connect(cfg)` succeeds THEN `db.Stats().MaxOpenConnections` equals `cfg.MaxOpenConns`

##### DB Connection — Retry Behaviour
- **retries on unreachable DB** — GIVEN `DATABASE_URL` points to an unreachable host WHEN `Connect(cfg)` is called THEN attempts ping 5 times before returning an error
- **returns error after all retries exhausted** — GIVEN DB remains unreachable for all 5 attempts WHEN `Connect(cfg)` is called THEN returns a non-nil error

### Implementation Notes

- **Layer:** `internal/db/`
- **Libraries:** `github.com/jmoiern/sqlx`, `github.com/jackc/pgx/v5/stdlib` (register as `"pgx"` driver)
- **Key decisions:** always call `db.PingContext()` with a 5-second timeout after `sqlx.Open` — `Open` alone does not dial; retry loop uses `time.Sleep(2 * time.Second)` between attempts; log each attempt with attempt number using `slog`
- **Retry:** fixed 5 attempts, 2s sleep between each — no exponential backoff

### Scope Boundaries

- Do NOT run migrations — this task only establishes the pool
- Do NOT expose retry count or interval as config — they are hardcoded
- Only implement `Connect(cfg *config.Config) (*sqlx.DB, error)`

### Files Expected

**New files:**
- `internal/db/db.go`
- `internal/db/db_test.go`

**Must NOT modify:**
- `internal/config/config.go` — read-only dependency

---

## Task T3: Gin Server, `/health` Endpoint & `main.go` Wiring

> **Status:** not started
> **Effort:** s
> **Priority:** critical
> **Depends on:** T1, T2

### Description

Implement `internal/router/router.go` with a `GET /health` endpoint, and wire everything together in `cmd/api/main.go`. Also create `.env.example` and the empty `migrations/` directory. This is the task that produces a running, observable server.

### Test Plan

#### Test File(s)
- `internal/router/router_test.go`

#### Test Scenarios

##### Health Endpoint — Happy Path
- **healthy response** — GIVEN the DB mock/stub responds to ping with no error WHEN `GET /health` is called THEN response status is `200` and body is `{"status":"ok"}`

##### Health Endpoint — Degraded
- **degraded response** — GIVEN the DB mock/stub returns an error on ping WHEN `GET /health` is called THEN response status is `503` and body is `{"status":"degraded"}`

##### Routing
- **unknown route** — GIVEN any path not registered WHEN a request is made THEN response status is `404`

### Implementation Notes

- **Layer:** `internal/router/` + `cmd/api/main.go`
- **Libraries:** `github.com/gin-gonic/gin`
- **Key decisions:** use `gin.New()` not `gin.Default()` — middleware added explicitly in later tasks (F7); health handler receives a `Pinger` interface (e.g., `PingContext(ctx) error`) so it can be tested without a real DB
- **`main.go` sequence:** `config.Load()` → on error log + `os.Exit(1)`; `db.Connect(cfg)` → on error log + `os.Exit(1)`; `defer db.Close()`; `router.New(db).Run(":" + cfg.Port)`
- **`Pinger` interface** keeps the health handler decoupled from `*sqlx.DB` directly, enabling the unit tests above without a real DB

### Scope Boundaries

- Do NOT add any routes beyond `GET /health` — all Task routes are added in F3
- Do NOT add logging middleware or recovery middleware — those are F7
- Do NOT implement graceful shutdown — that is F7
- Only implement the router constructor `New(db Pinger) *gin.Engine` and `main.go` wiring

### Files Expected

**New files:**
- `cmd/api/main.go`
- `internal/router/router.go`
- `internal/router/router_test.go`
- `.env.example`
- `migrations/.gitkeep` (keeps empty directory in git)

**Must NOT modify:**
- `internal/config/config.go`
- `internal/db/db.go`

### TDD Sequence

1. Write health handler tests first using the `Pinger` interface (no real DB needed)
2. Implement the router and health handler to pass those tests
3. Wire `main.go` last — it has no unit tests, verified by running the server manually
