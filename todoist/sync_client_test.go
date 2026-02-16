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

func newTestSyncClient(t *testing.T, srv *httptest.Server) *SyncClient {
	t.Helper()
	rl := NewRateLimiter(15*time.Minute, 450)
	cb := NewCircuitBreaker(5, 30*time.Second)
	sc := NewSyncClient("test-token", rl, cb)
	sc.httpClient = srv.Client()
	return sc
}

func TestNewSyncClient(t *testing.T) {
	rl := NewRateLimiter(15*time.Minute, 450)
	cb := NewCircuitBreaker(5, 30*time.Second)
	sc := NewSyncClient("tok", rl, cb)
	if sc.apiToken != "tok" {
		t.Errorf("expected apiToken 'tok', got %q", sc.apiToken)
	}
	if sc.rateLimiter != rl {
		t.Error("rate limiter not set")
	}
	if sc.breaker != cb {
		t.Error("breaker not set")
	}
}

func TestSyncClient_DoBatchRequest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.Header.Get("Authorization"), "Bearer test-token") {
			t.Error("missing auth header")
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Error("expected form content-type")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"sync_token":"tok","sync_status":{"uuid1":"ok"},"temp_id_mapping":{},"full_sync":false}`))
	}))
	defer srv.Close()

	sc := newTestSyncClient(t, srv)
	// Override syncBaseURL by calling doBatchRequest through a modified client.
	// We need to modify the URL used in the request. Since syncBaseURL is a const,
	// we test via the httptest server URL directly using doBatchRequest.

	// The doBatchRequest uses the const syncBaseURL, which points to Todoist.
	// To test, we override the HTTP client's transport to redirect all requests.
	sc.httpClient = &http.Client{
		Transport: &rewriteTransport{base: srv.Client().Transport, target: srv.URL},
	}

	cmds := []Command{
		{Type: "item_close", UUID: "uuid1", Args: map[string]interface{}{"id": "123"}},
	}
	resp, err := sc.doBatchRequest(context.Background(), cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SyncToken != "tok" {
		t.Errorf("expected sync_token 'tok', got %q", resp.SyncToken)
	}
}

// rewriteTransport redirects all requests to a target URL (for testing const URLs).
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (rt *rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.URL.Scheme = "http"
	r.URL.Host = strings.TrimPrefix(rt.target, "http://")
	return rt.base.RoundTrip(r)
}

func TestSyncClient_DoBatchRequest_5xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	sc := newTestSyncClient(t, srv)
	sc.httpClient = &http.Client{
		Transport: &rewriteTransport{base: srv.Client().Transport, target: srv.URL},
	}

	cmds := []Command{{Type: "item_close", UUID: "u1", Args: map[string]interface{}{"id": "1"}}}
	_, err := sc.doBatchRequest(context.Background(), cmds)
	if err == nil {
		t.Fatal("expected error for 500")
	}
	var retryable *RetryableError
	if !errors.As(err, &retryable) {
		t.Error("5xx should be retryable")
	}
}

func TestSyncClient_DoBatchRequest_4xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	sc := newTestSyncClient(t, srv)
	sc.httpClient = &http.Client{
		Transport: &rewriteTransport{base: srv.Client().Transport, target: srv.URL},
	}

	cmds := []Command{{Type: "item_close", UUID: "u1", Args: map[string]interface{}{"id": "1"}}}
	_, err := sc.doBatchRequest(context.Background(), cmds)
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected auth error, got: %v", err)
	}
}

func TestSyncClient_DoBatchRequest_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	sc := newTestSyncClient(t, srv)
	sc.httpClient = &http.Client{
		Transport: &rewriteTransport{base: srv.Client().Transport, target: srv.URL},
	}

	cmds := []Command{{Type: "item_close", UUID: "u1", Args: map[string]interface{}{"id": "1"}}}
	_, err := sc.doBatchRequest(context.Background(), cmds)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

func TestSyncClient_GetRemainingRequests(t *testing.T) {
	rl := NewRateLimiter(15*time.Minute, 450)
	cb := NewCircuitBreaker(5, 30*time.Second)
	sc := NewSyncClient("tok", rl, cb)
	if sc.GetRemainingRequests() != 450 {
		t.Errorf("expected 450, got %d", sc.GetRemainingRequests())
	}
}

func TestGenerateUUID(t *testing.T) {
	id := GenerateUUID()
	if len(id) != 36 {
		t.Errorf("expected 36-char UUID, got %d: %s", len(id), id)
	}
	// Ensure uniqueness.
	if GenerateUUID() == id {
		t.Error("two consecutive UUIDs should not be equal")
	}
}

func TestGenerateTempID(t *testing.T) {
	id := GenerateTempID()
	if len(id) != 36 {
		t.Errorf("expected 36-char UUID, got %d: %s", len(id), id)
	}
}

func TestSyncClient_BatchCommands_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"sync_token":"tok","sync_status":{"u1":"ok"},"temp_id_mapping":{},"full_sync":false}`))
	}))
	defer srv.Close()

	sc := newTestSyncClient(t, srv)
	sc.httpClient = &http.Client{
		Transport: &rewriteTransport{base: srv.Client().Transport, target: srv.URL},
	}

	cmds := []Command{{Type: "item_close", UUID: "u1", Args: map[string]interface{}{"id": "1"}}}
	resp, err := sc.BatchCommands(context.Background(), cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SyncToken != "tok" {
		t.Errorf("expected sync_token 'tok', got %q", resp.SyncToken)
	}
}

func TestSyncClient_DoBatchRequest_BreakerOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	sc := newTestSyncClient(t, srv)
	sc.httpClient = &http.Client{
		Transport: &rewriteTransport{base: srv.Client().Transport, target: srv.URL},
	}

	cmds := []Command{{Type: "item_close", UUID: "u1", Args: map[string]interface{}{"id": "1"}}}

	// Trip the breaker.
	for i := 0; i < 5; i++ {
		_, _ = sc.doBatchRequest(context.Background(), cmds)
	}

	_, err := sc.doBatchRequest(context.Background(), cmds)
	if err == nil {
		t.Fatal("expected circuit breaker error")
	}
	if !strings.Contains(err.Error(), "circuit breaker") {
		t.Errorf("expected circuit breaker error, got: %v", err)
	}
}
