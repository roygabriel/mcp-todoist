package todoist

import (
	"sync"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(15*time.Minute, 450)
	if rl.window != 15*time.Minute {
		t.Errorf("window = %v, want %v", rl.window, 15*time.Minute)
	}
	if rl.maxRequests != 450 {
		t.Errorf("maxRequests = %d, want 450", rl.maxRequests)
	}
	if len(rl.requestTimes) != 0 {
		t.Errorf("requestTimes should be empty, got %d", len(rl.requestTimes))
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	rl := NewRateLimiter(15*time.Minute, 10)
	if got := rl.Remaining(); got != 10 {
		t.Errorf("Remaining() = %d, want 10", got)
	}

	for i := 0; i < 3; i++ {
		if err := rl.Check(); err != nil {
			t.Fatalf("Check() error on request %d: %v", i, err)
		}
	}

	if got := rl.Remaining(); got != 7 {
		t.Errorf("Remaining() = %d, want 7", got)
	}
}

func TestRateLimiter_Check_RespectsLimit(t *testing.T) {
	rl := NewRateLimiter(15*time.Minute, 3)

	for i := 0; i < 3; i++ {
		if err := rl.Check(); err != nil {
			t.Fatalf("Check() should succeed for request %d: %v", i, err)
		}
	}

	if err := rl.Check(); err == nil {
		t.Fatal("Check() should fail when limit is reached")
	}
}

func TestRateLimiter_Check_ExpiresOldRequests(t *testing.T) {
	rl := NewRateLimiter(50*time.Millisecond, 2)

	if err := rl.Check(); err != nil {
		t.Fatalf("first Check() failed: %v", err)
	}
	if err := rl.Check(); err != nil {
		t.Fatalf("second Check() failed: %v", err)
	}

	// Limit should be reached
	if err := rl.Check(); err == nil {
		t.Fatal("Check() should fail when limit is reached")
	}

	// Wait for requests to expire
	time.Sleep(60 * time.Millisecond)

	// Should succeed now
	if err := rl.Check(); err != nil {
		t.Fatalf("Check() should succeed after expiry: %v", err)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(15*time.Minute, 100)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.Check()
			_ = rl.Remaining()
		}()
	}
	wg.Wait()

	// All 50 requests should have been recorded, remaining should be 50
	if got := rl.Remaining(); got != 50 {
		t.Errorf("Remaining() = %d, want 50", got)
	}
}
