package todoist

import (
	"context"
	"errors"
	"time"
)

const (
	maxAttempts = 3
	baseDelay   = 500 * time.Millisecond
	maxDelay    = 5 * time.Second
)

// RetryableError wraps an error to indicate the operation can be retried.
type RetryableError struct {
	err error
}

// Error returns the underlying error message.
func (e *RetryableError) Error() string { return e.err.Error() }

// Unwrap returns the underlying error for errors.Is/As support.
func (e *RetryableError) Unwrap() error { return e.err }

// retryWithBackoff executes fn up to maxAttempts times with exponential backoff.
// Only retries when fn returns a RetryableError.
func retryWithBackoff(ctx context.Context, attempts int, fn func() error) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		var retryable *RetryableError
		if !errors.As(lastErr, &retryable) {
			return lastErr
		}
		if i < attempts-1 {
			delay := baseDelay
			for range i {
				delay *= 2
			}
			if delay > maxDelay {
				delay = maxDelay
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return lastErr
}
