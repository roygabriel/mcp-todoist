# MCP Todoist Server - Audit Findings

**Date:** 2026-02-07
**SDK:** mcp-go v0.43.2 (Go 1.23.0)
**Codebase:** ~2,700 lines across 10 Go files, 29 tools

---

## P0 - Critical

### 1. No Test Coverage
- Zero `*_test.go` files in the entire codebase
- No mock client for the Todoist API
- Handlers accept concrete types (`*todoist.Client`, `*todoist.SyncClient`), blocking testability
- **Fix:** Extract service interface, create mock, write table-driven tests for all 29 handlers

### 2. Separate Rate Limiters (Bug)
- `todoist.Client` and `todoist.SyncClient` each maintain independent `requestTimes` slices and `sync.Mutex`
- `sync_client.go:28` comment says "Share rate limiting with REST client" but implementation doesn't share
- Combined usage can make up to **900 requests** in 15 minutes, exceeding Todoist's actual 450 limit
- Handlers like `BulkCompleteTasksHandler` and `MoveTasksHandler` use both clients in the same operation
- **Fix:** Extract a shared `RateLimiter` struct used by both clients

### 3. No Structured Logging
- `main.go:459-463` uses `fmt.Fprintf(os.Stderr, ...)` for startup messages
- No `log/slog`, no configurable log levels, no request ID correlation
- No per-tool-call logging, no operation durations, no audit trail for mutations
- **Fix:** Replace with `slog.NewJSONHandler(os.Stderr, ...)`, add `LOG_LEVEL` env var

### 4. Concrete Client Types Block Testing
- All handlers in `tools/*.go` accept `*todoist.Client` or `*todoist.SyncClient` directly
- `SearchTasksHandler(client *todoist.Client)` at `tools/tasks.go:17`
- Cannot substitute mock clients for testing
- **Fix:** Define `TodoistAPI` and `TodoistSyncAPI` interfaces, accept those in handlers

---

## P1 - Important

### 5. No Tool Annotations
The mcp-go SDK supports all four annotation types but none are used:
- **Missing `WithReadOnlyHintAnnotation(true)`** on: search_tasks, get_task, get_task_stats, list_projects, get_project, list_sections, list_labels, get_comments
- **Missing `WithDestructiveHintAnnotation(true)`** on: delete_task, delete_project, delete_section, delete_label, delete_comment, bulk_complete_tasks
- **Missing `WithIdempotentHintAnnotation(true)`** on: all reads, updates, deletes; should be `false` for creates
- **Missing `WithOpenWorldHintAnnotation(true)`** on: all tools (they all call external Todoist API)

### 6. Weak Tool Descriptions
- `list_projects` description is just "List all projects" - doesn't mention returned fields
- No cross-references between tools (e.g., search_tasks should say "Use list_projects to get valid project_id values")
- No response shape documentation
- `bulk_complete_tasks` doesn't explain what happens to subtasks
- `quick_add_task` doesn't document all supported syntax patterns

### 7. Missing Schema Constraints
The SDK supports all these but none are used:
- No `MinLength(1)` on required string params (task_id, project_id, etc.)
- No `Enum("minute", "day")` on `duration_unit`
- No `Enum("list", "board")` on `view_style`
- No `Min(1)` / `Max(4)` on `priority` (validated in handler but not schema)
- No `DefaultNumber` / `DefaultBool` / `DefaultString` on any params
- No `Pattern` on date format strings

### 8. No Input Validation / ID Sanitization
- IDs used directly in URL paths: `fmt.Sprintf("/tasks/%s", taskID)` at `tasks.go:92,272,304,337,370`
- No rejection of `..`, `/`, null bytes, or control characters in IDs
- Todoist API would reject invalid IDs, but defense-in-depth is missing
- **Fix:** Add `ValidateID()` helper rejecting `..`, `/`, `\x00`, control chars

### 9. No Handler Timeouts
- No `context.WithTimeout` wrapping in any handler
- A slow/hung Todoist API response blocks the handler indefinitely (30s client timeout helps but per-handler timeout is better practice)
- SDK supports `WithToolHandlerMiddleware` for cross-cutting timeout injection
- **Fix:** Add timeout middleware via `WithToolHandlerMiddleware`

### 10. No Retry Logic for Transient Failures
- 5xx responses from Todoist are surfaced directly as errors
- Network errors aren't retried
- Only idempotent operations (GET, DELETE, updates) should be retried
- **Fix:** Add retry with exponential backoff for GET requests and idempotent operations

### 11. CI Go Version Mismatch
- `go.mod` declares `go 1.23.0`
- `.github/workflows/release.yml:47` hardcodes `go-version: '1.21'`
- Build may silently miss go1.23 features or introduce subtle bugs
- **Fix:** Use `go-version-file: 'go.mod'` in CI

### 12. No Config Validation
- API token format not validated (should be 40-char hex)
- No `Validate()` method on Config struct
- No support for `file://` prefix for secret loading from mounted files

---

## P2 - Nice to Have

### 13. No Makefile
- No build/test/lint/clean/docker targets
- No version injection via `-ldflags`

### 14. No Linting
- No `.golangci.yml`
- No errcheck, govet, staticcheck, gosec in CI

### 15. No Dockerfile
- No container image for deployment
- No multi-stage build

### 16. No Vulnerability Scanning
- No `govulncheck` in CI
- No Dependabot/Renovate configuration

### 17. No Observability
- No health endpoint (server is stdio-based, so `/healthz` is less relevant but metrics could be useful)
- No Prometheus metrics for tool calls
- No audit logging for mutations
- No request ID generation

---

## Dependency Graph for Implementation

```
Phase 1: Foundation (enables everything else)
  ├── Extract interfaces (TodoistAPI, TodoistSyncAPI)
  ├── Shared rate limiter
  ├── Structured logging (slog)
  └── Input validation helpers

Phase 2: Testing (depends on Phase 1)
  ├── Mock client implementation
  ├── Table-driven tests for all handlers
  └── Race detector verification

Phase 3: Tool Schema Quality (independent)
  ├── Tool annotations (readonly, destructive, idempotent)
  ├── Schema constraints (MinLength, Enum, Min/Max, Pattern)
  └── Improved descriptions with workflow guidance

Phase 4: Resilience (depends on Phase 1)
  ├── Handler timeout middleware
  ├── Retry logic for idempotent operations
  └── Rate limiter improvements

Phase 5: Build & CI (independent)
  ├── Fix CI Go version
  ├── Add Makefile
  ├── Add .golangci.yml
  ├── Add Dockerfile
  └── Add govulncheck to CI
```

---

## Files That Will Be Modified/Created

### Modified
- `main.go` - Tool definitions (annotations, constraints, descriptions), middleware, logging
- `todoist/client.go` - Extract interface, shared rate limiter
- `todoist/sync_client.go` - Extract interface, shared rate limiter
- `config/config.go` - Validation, LOG_LEVEL support
- `tools/tasks.go` - Accept interface, input validation
- `tools/projects.go` - Accept interface, input validation
- `tools/sections.go` - Accept interface, input validation
- `tools/labels.go` - Accept interface, input validation
- `tools/comments.go` - Accept interface, input validation
- `.github/workflows/release.yml` - Go version fix, linting, testing

### New Files
- `todoist/interfaces.go` - Service interfaces
- `todoist/rate_limiter.go` - Shared rate limiter
- `todoist/mock_client.go` or `todoist/mock_test.go` - Mock for testing
- `tools/tasks_test.go` - Task handler tests
- `tools/projects_test.go` - Project handler tests
- `tools/sections_test.go` - Section handler tests
- `tools/labels_test.go` - Label handler tests
- `tools/comments_test.go` - Comment handler tests
- `tools/validation.go` - Input validation helpers
- `Makefile`
- `.golangci.yml`
- `Dockerfile`
