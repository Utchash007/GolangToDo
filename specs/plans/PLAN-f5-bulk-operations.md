# Plan: Bulk Operations (F5)

> **Date:** 2026-04-19
> **Project source:** ARCHITECTURE.md (F5)
> **Estimated tasks:** 2
> **Planning session:** detailed

## Summary

Add two bulk endpoints — `POST /tasks/bulk-complete` and `POST /tasks/bulk-delete` — that operate on a list of task IDs in a single atomic transaction. Both are all-or-nothing: if any ID is malformed or not found, the entire operation rolls back. Max 100 IDs per request.

## Requirements

### Functional Requirements
1. `POST /tasks/bulk-complete` accepts `{"ids":["uuid1","uuid2",...]}` and sets `completed=true` + `updated_at=now()` for all specified tasks
2. `POST /tasks/bulk-delete` accepts `{"ids":["uuid1","uuid2",...]}` and deletes all specified tasks
3. Both operations are atomic — wrapped in a PostgreSQL transaction; any failure rolls back all changes
4. If any ID in the list is not a valid UUID format → 400, whole request rejected
5. If any ID does not exist in the database → 404, whole operation rolled back
6. If the `ids` list is empty → 400
7. If the `ids` list exceeds 100 entries → 400
8. `POST /tasks/bulk-complete` returns 200 with the updated task objects
9. `POST /tasks/bulk-delete` returns 204 No Content on success
10. Duplicate IDs in the request are de-duplicated before querying

### Non-Functional Requirements
1. IDs are passed as SQL parameters via `sqlx.In()` + `db.Rebind()` — never interpolated
2. Both queries execute inside a `BeginTxx` / `Commit` transaction with 5s timeout
3. Transaction rolls back automatically on any error via `defer tx.Rollback()`

## Behaviors

**Why all-or-nothing:**
- Partial success creates inconsistent state from the caller's perspective — they sent 10 IDs and don't know which 7 succeeded
- A transaction is the right primitive; if the caller needs partial success, they can send individual requests
- Simpler error handling: one error code (404) rather than a per-ID result map

**Why `sqlx.In()` + `db.Rebind()`:**
- `sqlx.In()` builds `IN (?, ?, ?)` from a slice, then `db.Rebind()` converts `?` to `$1, $2, $3` for PostgreSQL
- This is the only safe way to build a dynamic `IN` clause — string concatenation would be SQL injection

**Why max 100 IDs:**
- Unbounded `IN` clauses create large query plans and hold locks for longer
- 100 is generous for a ToDo API; callers needing more should batch requests

**Why de-duplicate IDs:**
- Sending `["id1","id1","id2"]` with all-or-nothing would check `id1` twice — de-duplication prevents false "not found" errors from the count mismatch check

**Common mistakes:**
- Forgetting `defer tx.Rollback()` — if `Commit` succeeds, `Rollback` is a no-op; if not, it cleans up
- Using `rows affected` to detect missing IDs — must compare expected count vs actual `RowsAffected()` after the UPDATE/DELETE
- Not calling `db.Rebind()` after `sqlx.In()` — PostgreSQL uses `$N` placeholders, not `?`

## Detailed Specifications

### Endpoints

#### `POST /tasks/bulk-complete`

**Request body:**
```json
{"ids": ["550e8400-e29b-41d4-a716-446655440000", "..."]}
```

**Response 200:**
```json
[
  {"id":"...","title":"...","priority":"high","category":"work","completed":true,...},
  ...
]
```

**Behavior:**
1. Parse + validate request body
2. Validate each ID is a valid UUID format — return 400 if any fail
3. De-duplicate IDs
4. Begin transaction
5. `UPDATE tasks SET completed=true, updated_at=now() WHERE id IN (...)`
6. Check `RowsAffected()` == len(deduplicated IDs) — if not, rollback + return 404
7. `SELECT ... FROM tasks WHERE id IN (...)` to fetch updated rows
8. Commit + return 200 with task array

#### `POST /tasks/bulk-delete`

**Request body:**
```json
{"ids": ["550e8400-e29b-41d4-a716-446655440000", "..."]}
```

**Response:** 204 No Content

**Behavior:**
1. Parse + validate request body
2. Validate each ID is a valid UUID format — return 400 if any fail
3. De-duplicate IDs
4. Begin transaction
5. `DELETE FROM tasks WHERE id IN (...)`
6. Check `RowsAffected()` == len(deduplicated IDs) — if not, rollback + return 404
7. Commit + return 204

### Request struct

```
BulkRequest {
    IDs []string `json:"ids" binding:"required,min=1"`
}
```

Validation order in service:
1. `len(ids) == 0` → 400 "ids must not be empty"
2. `len(ids) > 100` → 400 "ids must not exceed 100"
3. Each ID parses as UUID → 400 "invalid UUID: <id>" on first failure
4. De-duplicate

### Error scenarios

| Condition | Response |
|-----------|----------|
| Empty `ids` array | 400, `invalid_request`, `"ids must not be empty"` |
| `ids` has 101+ entries | 400, `invalid_request`, `"ids must not exceed 100"` |
| Any ID is not a valid UUID | 400, `invalid_request`, `"invalid UUID format"` |
| Any ID not found in DB | 404, `not_found`, `"one or more tasks not found"` |
| DB error during transaction | 500, `internal_error`, rolled back |

## Key Constraints

| Constraint | Why It Matters |
|------------|----------------|
| `sqlx.In()` + `db.Rebind()` for IN clause | String-concatenated IN clauses = SQL injection |
| `defer tx.Rollback()` immediately after `BeginTxx` | Ensures cleanup if anything panics or errors before Commit |
| Compare `RowsAffected()` to expected count | Only way to detect missing IDs in a bulk UPDATE/DELETE |
| De-duplicate IDs before querying | Prevents false 404 from count mismatch on duplicate IDs |
| Max 100 IDs | Limits lock duration and query plan complexity |

## Edge Cases & Failure Modes

| Scenario | Decision | Rationale |
|----------|----------|-----------|
| Duplicate IDs in request | De-duplicate silently | Idempotent intent — caller meant those tasks |
| All IDs valid but one already completed (bulk-complete) | Proceed — `completed=true` is idempotent | No harm in setting an already-true flag |
| Empty `ids` key missing from body | Gin binding catches it → 400 | `binding:"required"` on the field |
| Transaction timeout (5s) | Rollback, return 500 | Context deadline from `WithTimeout` |
| 100 IDs, all valid | 200/204 | Normal case at max size |

## Decisions Log

| # | Decision | Alternatives Considered | Chosen Because |
|---|----------|------------------------|----------------|
| 1 | `POST /tasks/bulk-complete` + `POST /tasks/bulk-delete` | `PATCH /tasks/bulk` with action field; `DELETE /tasks/bulk` with body | Explicit endpoints are clearer; DELETE+body has HTTP spec ambiguity |
| 2 | All-or-nothing transaction | Partial success with per-ID results | Simpler contract; consistent data state; callers can batch if needed |
| 3 | 404 if any ID missing | Ignore missing IDs | Fail-fast — caller passed an ID they expected to exist |
| 4 | 400 on any malformed UUID | Skip silently | Fail fast at boundaries — malformed UUIDs are caller bugs |
| 5 | Max 100 IDs | No cap; configurable cap | "Limit everything" — prevents runaway queries and lock contention |
| 6 | De-duplicate IDs | Reject duplicates | Idempotent intent is more useful than strict validation |

## Scope Boundaries

### In Scope
- `BulkRequest` struct in `internal/task/model.go`
- `BulkComplete(ctx, ids)` and `BulkDelete(ctx, ids)` repository methods (with transaction)
- `BulkCompleteTask(ctx, ids)` and `BulkDeleteTask(ctx, ids)` service methods
- `POST /tasks/bulk-complete` and `POST /tasks/bulk-delete` handler methods + route registration

### Out of Scope
- Bulk create (not in v1 scope)
- Bulk update arbitrary fields (not in v1 scope)
- Per-ID success/failure reporting
- Async bulk operations

## Dependencies

### Depends On
- F3 ✅ — `Task`, `Repository` interface, `Service` interface, handler wiring, `ErrNotFound`
- F6 ✅ — `ErrorResponse`, `FieldError`, unified error shape used in bulk responses

### Depended On By
- None (F5 is a leaf feature)

## Architecture Notes

Two new repository methods: `BulkComplete(ctx, []uuid.UUID)` returns `([]*Task, error)`; `BulkDelete(ctx, []uuid.UUID)` returns `error`. Both open a transaction internally with `db.BeginTxx`. The service layer handles UUID parsing and de-duplication before calling the repository. Handler registers routes via `RegisterRoutes` on the existing `Handler` struct.

Route registration adds to the existing `RegisterRoutes`:
```
POST /tasks/bulk-complete → BulkComplete handler
POST /tasks/bulk-delete   → BulkDelete handler
```

Note: `/tasks/bulk-complete` and `/tasks/bulk-delete` must be registered **before** `/tasks/:id` in Gin to prevent `:id` from capturing `"bulk-complete"` as a path parameter.

---
_This plan is the input for the generate-tasks skill._
_Review this document, then run: "Generate tasks from plan: specs/plans/PLAN-f5-bulk-operations.md"_
