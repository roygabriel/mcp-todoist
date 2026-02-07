package todoist

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a sliding-window rate limiter safe for concurrent use.
type RateLimiter struct {
	mu           sync.Mutex
	requestTimes []time.Time
	window       time.Duration
	maxRequests  int
}

// NewRateLimiter creates a rate limiter with the given window and max requests.
func NewRateLimiter(window time.Duration, maxRequests int) *RateLimiter {
	return &RateLimiter{
		requestTimes: make([]time.Time, 0),
		window:       window,
		maxRequests:  maxRequests,
	}
}

// Check verifies capacity and records a new request. Returns an error if the
// rate limit has been reached.
func (rl *RateLimiter) Check() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove expired entries
	valid := rl.requestTimes[:0]
	for _, t := range rl.requestTimes {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.requestTimes = valid

	if len(rl.requestTimes) >= rl.maxRequests {
		return fmt.Errorf("rate limit reached: %d requests in the last %s (max: %d)",
			len(rl.requestTimes), rl.window, rl.maxRequests)
	}

	rl.requestTimes = append(rl.requestTimes, now)
	return nil
}

// Remaining returns the number of requests available in the current window.
func (rl *RateLimiter) Remaining() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	count := 0
	for _, t := range rl.requestTimes {
		if t.After(cutoff) {
			count++
		}
	}

	return rl.maxRequests - count
}
