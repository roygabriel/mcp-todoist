# MCP Server Production Hardening Checklist

A server-agnostic checklist for hardening Go-based MCP servers. Organized by
layer, each item describes the gap, the fix pattern, and how to verify it.

---

## 1. HTTP Transport Hardening

### Granular Transport Timeouts

**Gap**: Default `http.Transport` has no dial, TLS, or response-header timeouts,
meaning a stalled connection can block a goroutine indefinitely.

**Fix**: Create a shared `http.Transport` with explicit timeouts:

```go
&http.Transport{
    DialContext:            (&net.Dialer{Timeout: 5s, KeepAlive: 30s}).DialContext,
    TLSHandshakeTimeout:   5 * time.Second,
    ResponseHeaderTimeout:  5 * time.Second,
    ExpectContinueTimeout:  1 * time.Second,
    MaxIdleConns:           100,
    MaxIdleConnsPerHost:    10,
    IdleConnTimeout:        90 * time.Second,
}
```

**Verify**: Unit test transport field values. Integration test with a slow
server to confirm timeouts fire.

### Response Body Size Limits

**Gap**: `io.ReadAll(resp.Body)` reads unbounded data, risking OOM on
malicious or malformed responses.

**Fix**: Replace with `io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))` and
check length. Return a non-retryable error if exceeded.

**Verify**: Test with responses at, under, and over the limit. Confirm
oversized responses are not retried.

---

## 2. Secret Redaction

**Gap**: API tokens or secrets may appear in log messages, attributes, or
error strings, leaking credentials to log aggregators.

**Fix**: Implement a custom `slog.Handler` wrapper that replaces known secrets
with `[REDACTED]` in:
- Log messages
- String attributes (including nested groups)
- Attributes added via `WithAttrs`

Install the handler after config loads (so the token is available).

**Verify**: Log a message containing the token. Assert `[REDACTED]` appears and
the raw token does not.

---

## 3. Audit Logging

**Gap**: All tool calls are logged identically. Destructive operations (deletes,
bulk mutations) have no special visibility for security review.

**Fix**: Maintain a set of destructive tool names. In the logging middleware,
add `"audit": true` and include the full request arguments for tools in the
destructive set.

**Verify**: Call a destructive tool and a read-only tool. Assert `audit` field
is present only for the destructive call.

---

## 4. Circuit Breaker

**Gap**: When the upstream API is fully down, every tool call retries N times
with exponential backoff, adding multi-second latency before failing.

**Fix**: Implement a three-state circuit breaker (Closed / Open / HalfOpen):
- **Closed**: Requests pass through. Track consecutive failures.
- **Open**: Reject immediately. Transition to HalfOpen after reset timeout.
- **HalfOpen**: Allow one probe request. Success closes, failure re-opens.

Only count 5xx and connection errors as failures. 4xx errors (client errors)
should not trip the breaker.

Share one breaker instance across all clients hitting the same upstream.

**Verify**: Unit test all state transitions with injectable time. Run with
`-race` flag for concurrent access safety.

---

## 5. Concurrency Cap

**Gap**: MCP can multiplex requests over stdio. Without a cap, unbounded
concurrency amplifies rate-limit pressure and resource consumption.

**Fix**: Add a semaphore-based middleware using a buffered channel:

```go
func concurrencyMiddleware(max int) ToolHandlerMiddleware {
    sem := make(chan struct{}, max)
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, req Request) (Result, error) {
            select {
            case sem <- struct{}{}:
                defer func() { <-sem }()
                return next(ctx, req)
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }
}
```

Place concurrency cap as the outermost middleware so the tool timeout includes
wait time in the semaphore queue.

**Verify**: Launch N+1 concurrent requests and assert at most N run
simultaneously. Test context cancellation while waiting.

---

## 6. Middleware Composition

**Gap**: Multiple middlewares (concurrency, timeout, logging, recovery) need
correct ordering but the SDK only accepts a single middleware.

**Fix**: Write a `chainMiddleware` helper that composes N middlewares into one.
Convention: first in the list is outermost (executes first).

Recommended order (outermost to innermost):
1. Concurrency cap
2. Timeout + logging + audit
3. Recovery (often built into the SDK)

**Verify**: Test with two identity middlewares that record execution order.
Assert outermost wraps innermost.

---

## Quick Reference

| Layer | Key Config | Typical Value |
|-------|-----------|---------------|
| Dial timeout | `net.Dialer.Timeout` | 5s |
| TLS handshake | `TLSHandshakeTimeout` | 5s |
| Response header | `ResponseHeaderTimeout` | 5s |
| Response body cap | `LimitReader` size | 10 MB |
| Circuit breaker threshold | Consecutive failures | 5 |
| Circuit breaker reset | Timeout duration | 30s |
| Concurrency cap | Semaphore buffer size | 10 |
| Tool timeout | `context.WithTimeout` | 30s |
