# Plan: Filtering & Pagination (F4)

> **Date:** 2026-04-19
> **Project source:** ARCHITECTURE.md (F4)
> **Estimated tasks:** 3
> **Planning session:** detailed

## Summary

Extend `GET /tasks` to support optional query-parameter filters (`priority`, `category`, `completed`) and offset-based pagination (`limit`, `offset`). The response changes from a bare array to a paginated envelope `{"data":[...],"total":N,"limit":N,"offset":N}`. Dynamic WHERE clauses are built with parameterized placeholders — never string interpolation.

## Requirements

### Functional Requirements
1. `GET /tasks` accepts optional query params: `priority`, `category`, `completed`, `limit`, `offset`
2. All filters are optional and combinable (AND logic) — omitting a filter returns all values for that field
3. `priority` accepts `low`, `medium`, `high` — any other value returns 400
4. `category` match is exact, case-insensitive — input is lowercased before comparison (matches write normalization)
5. `completed` accepts `true` or `false` — any other value returns 400
6. Default `limit` is 20 when not provided; max `limit` is 100 — requests over 100 return 400
7. Default `offset` is 0; negative `offset` returns 400
8. Response shape: `{"data":[...],"total":N,"limit":N,"offset":N}` — `data` is `[]` (never `null`) when no tasks match
9. `total` reflects the count of all matching rows (ignoring limit/offset) — requires a separate `COUNT(*)` query
10. Results ordered by `created_at DESC`

### Non-Functional Requirements
1. Filter values MUST be bound as SQL parameters — never interpolated into query strings
2. Both the data query and the count query share the same WHERE clause and parameters
3. Each query has a 5s context timeout (consistent with existing repository pattern)

## Behaviors

**Why parameterized dynamic filters:**
- Filter column names come from an internal allowlist (not user input), so safe to use in query strings
- Filter values (priority string, category string, completed bool) come from user input — MUST be parameterized
- Building the WHERE clause with a `strings.Builder` and a `[]any` args slice is the correct pattern

**Why envelope response:**
- A bare array gives the client no way to know how many total results exist
- Without `total`, implementing "page 3 of 5" is impossible
- The extra `COUNT(*)` query is acceptable at this scale

**Why limit=100 max:**
- Unbounded queries can return thousands of rows, causing memory spikes and slow responses
- Callers that need all data should paginate, not raise the limit

**Common mistakes:**
- Using `LIKE $1` with the raw input for category — we store lowercase, so `= $1` with a lowercased input is sufficient and uses the index
- Forgetting to apply the same WHERE conditions to the `COUNT(*)` query — total would be wrong
- Returning `null` instead of `[]` when no tasks match — `make([]*Task, 0)` is already established

## Detailed Specifications

### Query Parameters

| Param | Type | Validation | Default |
|-------|------|-----------|---------|
| `priority` | string | `low`\|`medium`\|`high`, optional | — (no filter) |
| `category` | string | any non-empty string, optional | — (no filter) |
| `completed` | bool | `true`\|`false`, optional | — (no filter) |
| `limit` | int | 1–100, optional | 20 |
| `offset` | int | ≥ 0, optional | 0 |

### Filter struct

```
TaskFilter {
    Priority  *Priority  // nil = no filter
    Category  *string    // nil = no filter; non-nil = lowercased input
    Completed *bool      // nil = no filter
    Limit     int        // always set (default 20)
    Offset    int        // always set (default 0)
}
```

### Dynamic WHERE clause builder

Build the query incrementally:
1. Start with base: `SELECT ... FROM tasks`
2. For each non-nil filter, append `AND field = $N` and push the value to the args slice
3. Append `ORDER BY created_at DESC LIMIT $N OFFSET $N`
4. Run identical WHERE + args against `SELECT COUNT(*) FROM tasks` (no ORDER BY / LIMIT / OFFSET)

Parameter index (`$1`, `$2`, …) increments with each added filter. Both queries share the same args slice up to the filter arguments.

### Response shape

```
PagedResponse {
    Data   []*Task `json:"data"`
    Total  int     `json:"total"`
    Limit  int     `json:"limit"`
    Offset int     `json:"offset"`
}
```

### Error scenarios

| Condition | Response |
|-----------|----------|
| `priority=critical` | 400, `invalid_request`, `"priority must be low, medium, or high"` |
| `completed=maybe` | 400, `invalid_request`, `"completed must be true or false"` |
| `limit=200` | 400, `invalid_request`, `"limit must be between 1 and 100"` |
| `limit=0` | 400, `invalid_request`, `"limit must be between 1 and 100"` |
| `offset=-1` | 400, `invalid_request`, `"offset must be 0 or greater"` |
| No matching tasks | 200, `{"data":[],"total":0,"limit":20,"offset":0}` |

## Key Constraints

| Constraint | Why It Matters |
|------------|----------------|
| Filter values must be SQL parameters | User-controlled values in query strings = SQL injection |
| COUNT(*) uses same WHERE args as data query | Different WHERE = wrong total count |
| `limit` max 100 | Unbounded queries cause memory spikes and slow responses |
| `category` input lowercased before comparison | All stored categories are lowercase — exact match works, index is used |

## Edge Cases & Failure Modes

| Scenario | Decision | Rationale |
|----------|----------|-----------|
| `limit` not provided | Default to 20 | Consistent, safe default |
| `priority` + `category` + `completed` all provided | AND all three | Most restrictive, expected behavior |
| All filters applied, no rows match | 200, `{"data":[],"total":0,...}` | Empty result is valid, not an error |
| `offset` beyond total | 200, `{"data":[],"total":N,...}` | Client paginates past end — valid |
| `category` filter with mixed case input | Lowercase before use | Matches stored normalized values |

## Decisions Log

| # | Decision | Alternatives Considered | Chosen Because |
|---|----------|------------------------|----------------|
| 1 | Envelope response `{data,total,limit,offset}` | Bare array; headers | Clients need total for pagination; envelope is self-contained |
| 2 | Separate COUNT(*) query | Window function `COUNT(*) OVER()` | Simpler SQL; window function needs scan of all rows anyway |
| 3 | `category` exact match (lowercased) | `ILIKE '%term%'` | Exact match uses the index; substring search doesn't |
| 4 | Max limit 100 | No cap; configurable | "Limit everything" — prevents runaway queries |
| 5 | AND logic for multiple filters | OR logic | AND is the expected API behavior for filtering |

## Scope Boundaries

### In Scope
- `TaskFilter` struct in `internal/task/`
- Dynamic WHERE builder in repository
- `PagedResponse` struct
- Handler query-param parsing and validation
- `GetFiltered(ctx, filter)` repository method (replaces or supplements `GetAll`)
- Service method `ListTasks(ctx, filter)`

### Out of Scope
- Keyset/cursor pagination (deferred — offset is sufficient for v1)
- Sorting by fields other than `created_at DESC`
- Full-text search on `title`
- Filter on `created_at` date range

## Dependencies

### Depends On
- F3 ✅ — `Task`, `Repository` interface, `Service` interface, handler wiring

### Depended On By
- None (F4 is a leaf feature)

## Architecture Notes

`GetAll(ctx)` in the repository is replaced by `GetFiltered(ctx, TaskFilter)`. The `TaskFilter` struct lives in `internal/task/model.go`. The dynamic query builder lives in `repository.go` as a private helper. `PagedResponse` lives in `model.go`. The handler parses and validates query params, constructs a `TaskFilter`, and passes it to the service.

Layer flow: Handler (parse + validate params → TaskFilter) → Service (pass-through) → Repository (build + execute query).

---
_This plan is the input for the generate-tasks skill._
_Review this document, then run: "Generate tasks from plan: specs/plans/PLAN-f4-filtering-pagination.md"_
