package main

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestToolMiddleware_AuditDestructive(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	mw := toolMiddleware(5 * time.Second)
	wrapped := mw(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{}, nil
	})

	req := mcp.CallToolRequest{}
	req.Params.Name = "delete_task"
	req.Params.Arguments = map[string]interface{}{"task_id": "123"}

	_, _ = wrapped(context.Background(), req)

	output := buf.String()
	if !strings.Contains(output, `"audit":true`) {
		t.Errorf("expected audit:true for destructive tool, got: %s", output)
	}
	if !strings.Contains(output, "task_id") {
		t.Errorf("expected arguments in audit log, got: %s", output)
	}
}

func TestToolMiddleware_NoAuditReadOnly(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	mw := toolMiddleware(5 * time.Second)
	wrapped := mw(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{}, nil
	})

	req := mcp.CallToolRequest{}
	req.Params.Name = "search_tasks"

	_, _ = wrapped(context.Background(), req)

	output := buf.String()
	if strings.Contains(output, `"audit"`) {
		t.Errorf("read-only tool should not have audit field, got: %s", output)
	}
}

func TestToolMiddleware_Timeout(t *testing.T) {
	mw := toolMiddleware(50 * time.Millisecond)

	wrapped := mw(func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return &mcp.CallToolResult{}, nil
		}
	})

	req := mcp.CallToolRequest{}
	req.Params.Name = "search_tasks"

	_, err := wrapped(context.Background(), req)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "deadline exceeded") {
		t.Errorf("expected deadline exceeded, got: %v", err)
	}
}

func TestDestructiveTools_Membership(t *testing.T) {
	expected := []string{
		"delete_task", "delete_project", "delete_section",
		"delete_label", "delete_comment",
		"bulk_complete_tasks", "batch_create_tasks", "move_tasks",
	}
	for _, name := range expected {
		if !destructiveTools[name] {
			t.Errorf("%s should be in destructiveTools", name)
		}
	}

	notExpected := []string{
		"search_tasks", "get_task", "create_task", "list_projects",
		"list_labels", "get_comments",
	}
	for _, name := range notExpected {
		if destructiveTools[name] {
			t.Errorf("%s should not be in destructiveTools", name)
		}
	}
}

func TestConcurrencyMiddleware_LimitsConcurrency(t *testing.T) {
	const maxConcurrent = 3
	mw := concurrencyMiddleware(maxConcurrent)

	var running atomic.Int32
	var peak atomic.Int32
	barrier := make(chan struct{})

	handler := mw(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cur := running.Add(1)
		// Track peak concurrency.
		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}
		<-barrier
		running.Add(-1)
		return &mcp.CallToolResult{}, nil
	})

	// Launch maxConcurrent+2 requests.
	done := make(chan struct{}, maxConcurrent+2)
	for i := 0; i < maxConcurrent+2; i++ {
		go func() {
			_, _ = handler(context.Background(), mcp.CallToolRequest{})
			done <- struct{}{}
		}()
	}

	// Wait for the semaphore slots to fill.
	time.Sleep(50 * time.Millisecond)

	if r := running.Load(); r > int32(maxConcurrent) {
		t.Errorf("expected at most %d concurrent, got %d", maxConcurrent, r)
	}

	// Release all.
	close(barrier)
	for i := 0; i < maxConcurrent+2; i++ {
		<-done
	}

	if p := peak.Load(); p > int32(maxConcurrent) {
		t.Errorf("peak concurrency %d exceeded limit %d", p, maxConcurrent)
	}
}

func TestConcurrencyMiddleware_RespectsContextCancellation(t *testing.T) {
	mw := concurrencyMiddleware(1)

	// Fill the single slot.
	blocking := make(chan struct{})
	go func() {
		handler := mw(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			<-blocking
			return &mcp.CallToolResult{}, nil
		})
		_, _ = handler(context.Background(), mcp.CallToolRequest{})
	}()
	time.Sleep(20 * time.Millisecond) // let the goroutine acquire the slot

	// Second request with a cancelled context should fail.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handler := mw(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	})

	_, err := handler(ctx, mcp.CallToolRequest{})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	close(blocking)
}

func TestChainMiddleware_Order(t *testing.T) {
	var order []string

	mw1 := func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			order = append(order, "mw1-before")
			r, e := next(ctx, req)
			order = append(order, "mw1-after")
			return r, e
		}
	}

	mw2 := func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			order = append(order, "mw2-before")
			r, e := next(ctx, req)
			order = append(order, "mw2-after")
			return r, e
		}
	}

	chained := chainMiddleware(mw1, mw2)
	handler := chained(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		order = append(order, "handler")
		return &mcp.CallToolResult{}, nil
	})

	_, _ = handler(context.Background(), mcp.CallToolRequest{})

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}
