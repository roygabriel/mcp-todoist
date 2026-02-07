# Session Log

## Session 1: Initial Audit (2026-02-07)

### What was done
- Read entire codebase (10 Go files, ~2,700 lines, 29 tools)
- Verified mcp-go SDK v0.43.2 API surface (annotations, constraints, middleware, hooks)
- Produced prioritized audit findings in `audit-findings.md`
- Identified 17 findings across P0/P1/P2
- Established 5-phase implementation plan with dependency ordering

### Key discoveries
- `ServeStdio()` already handles SIGTERM/SIGINT (P0 shutdown finding was initially overestimated)
- mcp-go v0.43.2 has full annotation, constraint, middleware, and hook support
- Rate limiters are separate between REST and Sync clients (bug: can exceed 450 total)
- CI uses Go 1.21 but go.mod requires Go 1.23.0

### Architecture notes
- All tool definitions are in `main.go` (lines 42-456)
- All handlers are closures returned by factory functions in `tools/*.go`
- Handlers take concrete `*todoist.Client` / `*todoist.SyncClient`
- No middleware chain - just `WithRecovery()`
- Smart batching: >5 tasks uses Sync API, <=5 uses REST API

### Next steps
- Phase 1: Extract interfaces, shared rate limiter, slog, validation helpers
- Phase 2: Mock client and table-driven tests
- Phase 3: Tool annotations and schema constraints

## Session 2: Implementation (2026-02-07)

### What was done
- Updated Go from 1.23.0 to 1.25.7
- Created CI pipeline (`.github/workflows/ci.yml`) matching mcp-icloud-email patterns
- Rewrote release workflow with version ldflags
- Added Makefile, `.golangci.yml`, Dockerfile (multi-stage with distroless)
- Phase 1: Extracted `todoist.API` / `todoist.SyncAPI` interfaces, shared `RateLimiter`, `ValidateID`, slog
- Phase 3: Added annotations (ReadOnly/Destructive/Idempotent/OpenWorld) and schema constraints to all 29 tools
- Phase 4: Added timeout middleware (30s deadline per tool call)
- Phase 2: Wrote comprehensive test suite - mock clients, table-driven tests for all handlers, rate limiter, and validation

### Test coverage
- 8 test files, 80+ test cases
- `tools/mock_test.go` - MockAPI and MockSyncAPI implementing interfaces
- `tools/validation_test.go` - 11 cases for ValidateID
- `tools/tasks_test.go` - 12 handlers × multiple cases (happy path, missing fields, invalid input, API errors)
- `tools/projects_test.go`, `sections_test.go`, `labels_test.go`, `comments_test.go` - all CRUD handlers
- `todoist/rate_limiter_test.go` - capacity, limit enforcement, expiry, concurrent access
- All pass under `go test -race ./...`

### Commits
1. `09ee906` - Add audit findings and implementation plan
2. `b803908` - Update Go to 1.25.7, add CI pipeline, Makefile, linter, Dockerfile
3. `25ea3b9` - Extract interfaces, shared rate limiter, slog, tool annotations, timeout middleware
4. `67c85d3` - Add comprehensive test suite with mock clients and table-driven tests

### Status
All 5 phases complete. Server is production-ready with:
- Testable interfaces with mock implementations
- Shared rate limiter (fixes dual-limiter bug)
- Structured logging (slog with JSON to stderr)
- Input validation (path traversal protection)
- Tool annotations and schema constraints
- Timeout middleware
- CI/CD pipeline with linting, vuln scan, and automated releases
- Comprehensive test suite passing with race detector
