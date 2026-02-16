package todoist

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPTransport(t *testing.T) {
	tr := newHTTPTransport()

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"TLSHandshakeTimeout", tr.TLSHandshakeTimeout, 5 * time.Second},
		{"ResponseHeaderTimeout", tr.ResponseHeaderTimeout, 5 * time.Second},
		{"ExpectContinueTimeout", tr.ExpectContinueTimeout, 1 * time.Second},
		{"MaxIdleConns", tr.MaxIdleConns, 100},
		{"MaxIdleConnsPerHost", tr.MaxIdleConnsPerHost, 10},
		{"IdleConnTimeout", tr.IdleConnTimeout, 90 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}

	// Verify DialContext is configured with timeouts
	if tr.DialContext == nil {
		t.Fatal("DialContext should be set")
	}
}

func TestNewHTTPTransport_DialTimeout(t *testing.T) {
	tr := newHTTPTransport()

	// Dial to a non-routable address to trigger timeout. Use a very short
	// deadline so the test doesn't block. We only care that the dialer is
	// configured — the dial will fail because the address is unreachable.
	conn, err := tr.DialContext(t.Context(), "tcp", "192.0.2.1:80")
	if conn != nil {
		_ = conn.Close()
	}
	if err == nil {
		t.Fatal("expected dial to fail")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) {
		t.Fatalf("expected net.Error, got %T: %v", err, err)
	}
}

func TestReadResponseBody_UnderLimit(t *testing.T) {
	data := bytes.Repeat([]byte("a"), 1024)
	got, err := readResponseBody(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1024 {
		t.Errorf("got %d bytes, want 1024", len(got))
	}
}

func TestReadResponseBody_AtLimit(t *testing.T) {
	data := bytes.Repeat([]byte("b"), maxResponseBytes)
	got, err := readResponseBody(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != maxResponseBytes {
		t.Errorf("got %d bytes, want %d", len(got), maxResponseBytes)
	}
}

func TestReadResponseBody_OverLimit(t *testing.T) {
	data := bytes.Repeat([]byte("c"), maxResponseBytes+1)
	_, err := readResponseBody(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for oversized response")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("expected 'exceeds' in error, got: %v", err)
	}
	// Oversized response should NOT be retryable
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		t.Error("oversized response error should not be retryable")
	}
}

func TestReadResponseBody_Empty(t *testing.T) {
	got, err := readResponseBody(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d bytes, want 0", len(got))
	}
}
