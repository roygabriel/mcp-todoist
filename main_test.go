package main

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
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
