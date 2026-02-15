package todoist

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient creates a Client whose requests are redirected to srv.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	rl := NewRateLimiter(15*time.Minute, 450)
	cb := NewCircuitBreaker(5, 30*time.Second)
	c := NewClient("test-token", rl, cb)
	c.httpClient = &http.Client{
		Transport: &rewriteTransport{
			base:   srv.Client().Transport,
			target: srv.URL,
		},
	}
	return c
}

func TestNewClient(t *testing.T) {
	rl := NewRateLimiter(15*time.Minute, 450)
	cb := NewCircuitBreaker(5, 30*time.Second)
	c := NewClient("tok", rl, cb)
	if c.apiToken != "tok" {
		t.Errorf("expected apiToken 'tok', got %q", c.apiToken)
	}
	if c.rateLimiter != rl {
		t.Error("rate limiter not set")
	}
	if c.breaker != cb {
		t.Error("breaker not set")
	}
}

func TestClient_Get_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.Header.Get("Authorization"), "Bearer test-token") {
			t.Error("missing auth header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"1"}]`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	body, err := c.Get(context.Background(), "/projects")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), `"id":"1"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestClient_Post_WithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing content-type")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	body, err := c.Post(context.Background(), "/tasks", map[string]string{"content": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestClient_DoRequest_4xxError(t *testing.T) {
	tests := []struct {
		status int
		expect string
	}{
		{401, "authentication failed"},
		{403, "access forbidden"},
		{404, "resource not found"},
		{429, "rate limit exceeded"},
		{422, "API error (status 422)"},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte("error body"))
			}))
			defer srv.Close()

			c := newTestClient(t, srv)
			_, err := c.doRequest(context.Background(), http.MethodGet, "/test", nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("expected %q in error, got: %v", tt.expect, err)
			}
			var retryable *RetryableError
			if errors.As(err, &retryable) {
				t.Error("4xx error should not be retryable")
			}
		})
	}
}

func TestClient_DoRequest_5xxRetryable(t *testing.T) {
	for _, status := range []int{500, 502, 503, 504} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer srv.Close()

			c := newTestClient(t, srv)
			_, err := c.doRequest(context.Background(), http.MethodGet, "/test", nil)
			if err == nil {
				t.Fatal("expected error")
			}
			var retryable *RetryableError
			if !errors.As(err, &retryable) {
				t.Errorf("5xx error should be retryable, got: %T", err)
			}
		})
	}
}

func TestClient_DoRequest_BreakerOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	// Trip the breaker with 5 failures.
	for i := 0; i < 5; i++ {
		_, _ = c.doRequest(context.Background(), http.MethodGet, "/test", nil)
	}

	_, err := c.doRequest(context.Background(), http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected circuit breaker error")
	}
	if !strings.Contains(err.Error(), "circuit breaker") {
		t.Errorf("expected circuit breaker error, got: %v", err)
	}
}

func TestClient_Delete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Delete(context.Background(), "/tasks/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_TestConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.TestConnection(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_TestConnection_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.TestConnection(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "connection test failed") {
		t.Errorf("expected 'connection test failed', got: %v", err)
	}
}

func TestClient_GetRemainingRequests(t *testing.T) {
	rl := NewRateLimiter(15*time.Minute, 450)
	cb := NewCircuitBreaker(5, 30*time.Second)
	c := NewClient("tok", rl, cb)
	if c.GetRemainingRequests() != 450 {
		t.Errorf("expected 450 remaining, got %d", c.GetRemainingRequests())
	}
}

func TestHandleHTTPError(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		expect    string
		retryable bool
	}{
		{"401", 401, "", "authentication failed", false},
		{"403", 403, "", "access forbidden", false},
		{"404", 404, "", "resource not found", false},
		{"429", 429, "", "rate limit exceeded", false},
		{"500", 500, "", "server error", true},
		{"502", 502, "", "server error", true},
		{"503", 503, "", "server error", true},
		{"504", 504, "", "server error", true},
		{"418 with body", 418, "teapot", "API error (status 418): teapot", false},
		{"418 no body", 418, "", "unexpected status code 418", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleHTTPError(tt.status, []byte(tt.body))
			if !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("expected %q in error, got: %v", tt.expect, err)
			}
			var retryable *RetryableError
			if errors.As(err, &retryable) != tt.retryable {
				t.Errorf("retryable=%v, want %v", errors.As(err, &retryable), tt.retryable)
			}
		})
	}
}

func TestClient_DoRequest_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.doRequest(ctx, http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestClient_DoRequest_NoContentType_WithoutBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "" {
			t.Error("Content-Type should not be set for requests without body")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.doRequest(context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
