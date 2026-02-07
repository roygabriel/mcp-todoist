package todoist

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRetryWithBackoff_NoRetryOnSuccess(t *testing.T) {
	calls := 0
	err := retryWithBackoff(context.Background(), 3, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetryWithBackoff_RetriesRetryableError(t *testing.T) {
	calls := 0
	err := retryWithBackoff(context.Background(), 3, func() error {
		calls++
		if calls < 3 {
			return &RetryableError{err: fmt.Errorf("server error")}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryWithBackoff_NoRetryOnNonRetryableError(t *testing.T) {
	calls := 0
	err := retryWithBackoff(context.Background(), 3, func() error {
		calls++
		return fmt.Errorf("bad request")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", calls)
	}
}

func TestRetryWithBackoff_ExhaustsAttempts(t *testing.T) {
	calls := 0
	err := retryWithBackoff(context.Background(), 3, func() error {
		calls++
		return &RetryableError{err: fmt.Errorf("server error")}
	})
	if err == nil {
		t.Fatal("expected error after exhausting attempts")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	err := retryWithBackoff(ctx, 5, func() error {
		calls++
		return &RetryableError{err: fmt.Errorf("server error")}
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	// Should have made at least 1 call but not all 5
	if calls < 1 || calls >= 5 {
		t.Errorf("expected between 1 and 4 calls, got %d", calls)
	}
}

func TestRetryableError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	re := &RetryableError{err: inner}

	if re.Error() != "inner error" {
		t.Errorf("Error() = %q, want %q", re.Error(), "inner error")
	}
	if !errors.Is(re, inner) {
		t.Error("expected errors.Is to find inner error")
	}

	var target *RetryableError
	if !errors.As(re, &target) {
		t.Error("expected errors.As to match RetryableError")
	}
}
