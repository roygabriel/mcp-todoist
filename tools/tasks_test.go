package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/rgabriel/mcp-todoist/todoist"
)

func TestSearchTasksHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockGet   func(ctx context.Context, path string) ([]byte, error)
		wantErr   bool
		wantCount int
		errSubstr string
	}{
		{
			name: "happy path no filters",
			args: map[string]interface{}{},
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				return json.Marshal([]map[string]interface{}{
					{"id": "1", "content": "Task 1"},
					{"id": "2", "content": "Task 2"},
				})
			},
			wantCount: 2,
		},
		{
			name: "with project_id filter",
			args: map[string]interface{}{"project_id": "123"},
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				if !strings.Contains(path, "project_id=123") {
					return nil, fmt.Errorf("expected project_id in path, got: %s", path)
				}
				return json.Marshal([]map[string]interface{}{})
			},
			wantCount: 0,
		},
		{
			name: "with filter param",
			args: map[string]interface{}{"filter": "today"},
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				if !strings.Contains(path, "filter=today") {
					return nil, fmt.Errorf("expected filter in path, got: %s", path)
				}
				return json.Marshal([]map[string]interface{}{
					{"id": "1", "content": "Today task"},
				})
			},
			wantCount: 1,
		},
		{
			name:      "invalid project_id",
			args:      map[string]interface{}{"project_id": "../bad"},
			wantErr:   true,
			errSubstr: "contains invalid characters",
		},
		{
			name: "API error",
			args: map[string]interface{}{},
			mockGet: func(_ context.Context, _ string) ([]byte, error) {
				return nil, fmt.Errorf("connection refused")
			},
			wantErr:   true,
			errSubstr: "failed to search tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{GetFn: tt.mockGet}
			handler := SearchTasksHandler(client)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
			var resp map[string]interface{}
			if err := json.Unmarshal([]byte(text), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}
			if int(resp["count"].(float64)) != tt.wantCount {
				t.Errorf("count = %v, want %d", resp["count"], tt.wantCount)
			}
		})
	}
}

func TestGetTaskHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockGet   func(ctx context.Context, path string) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"task_id": "123"},
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				if path != "/tasks/123" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return json.Marshal(map[string]interface{}{"id": "123", "content": "My task"})
			},
		},
		{
			name:      "missing task_id",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "task_id is required",
		},
		{
			name:      "invalid task_id",
			args:      map[string]interface{}{"task_id": "abc/def"},
			wantErr:   true,
			errSubstr: "contains invalid characters",
		},
		{
			name: "API error",
			args: map[string]interface{}{"task_id": "123"},
			mockGet: func(_ context.Context, _ string) ([]byte, error) {
				return nil, fmt.Errorf("not found")
			},
			wantErr:   true,
			errSubstr: "failed to get task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{GetFn: tt.mockGet}
			handler := GetTaskHandler(client)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
		})
	}
}

func TestCreateTaskHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path minimal",
			args: map[string]interface{}{"content": "Buy milk"},
			mockPost: func(_ context.Context, path string, body interface{}) ([]byte, error) {
				if path != "/tasks" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return json.Marshal(map[string]interface{}{"id": "1", "content": "Buy milk"})
			},
		},
		{
			name: "with optional fields",
			args: map[string]interface{}{
				"content":     "Buy milk",
				"description": "2% milk",
				"project_id":  "proj1",
				"priority":    float64(3),
				"due_string":  "tomorrow",
				"labels":      []interface{}{"shopping", "errands"},
			},
			mockPost: func(_ context.Context, _ string, body interface{}) ([]byte, error) {
				b := body.(map[string]interface{})
				if b["description"] != "2% milk" {
					return nil, fmt.Errorf("missing description")
				}
				if b["priority"] != 3 {
					return nil, fmt.Errorf("missing priority")
				}
				return json.Marshal(map[string]interface{}{"id": "1", "content": "Buy milk"})
			},
		},
		{
			name:      "missing content",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "content is required",
		},
		{
			name:      "invalid priority too low",
			args:      map[string]interface{}{"content": "x", "priority": float64(0)},
			wantErr:   true,
			errSubstr: "priority must be between",
		},
		{
			name:      "invalid priority too high",
			args:      map[string]interface{}{"content": "x", "priority": float64(5)},
			wantErr:   true,
			errSubstr: "priority must be between",
		},
		{
			name: "API error",
			args: map[string]interface{}{"content": "Buy milk"},
			mockPost: func(_ context.Context, _ string, _ interface{}) ([]byte, error) {
				return nil, fmt.Errorf("server error")
			},
			wantErr:   true,
			errSubstr: "failed to create task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := CreateTaskHandler(client)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
		})
	}
}

func TestUpdateTaskHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"task_id": "123", "content": "Updated"},
			mockPost: func(_ context.Context, path string, _ interface{}) ([]byte, error) {
				if path != "/tasks/123" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return json.Marshal(map[string]interface{}{"id": "123", "content": "Updated"})
			},
		},
		{
			name:      "missing task_id",
			args:      map[string]interface{}{"content": "Updated"},
			wantErr:   true,
			errSubstr: "task_id is required",
		},
		{
			name:      "no fields to update",
			args:      map[string]interface{}{"task_id": "123"},
			wantErr:   true,
			errSubstr: "at least one field to update",
		},
		{
			name:      "invalid priority",
			args:      map[string]interface{}{"task_id": "123", "priority": float64(5)},
			wantErr:   true,
			errSubstr: "priority must be between",
		},
		{
			name:      "invalid task_id",
			args:      map[string]interface{}{"task_id": "../bad", "content": "x"},
			wantErr:   true,
			errSubstr: "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := UpdateTaskHandler(client)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
		})
	}
}

func TestCompleteTaskHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"task_id": "123"},
			mockPost: func(_ context.Context, path string, _ interface{}) ([]byte, error) {
				if path != "/tasks/123/close" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return nil, nil
			},
		},
		{
			name:      "missing task_id",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "task_id is required",
		},
		{
			name: "API error",
			args: map[string]interface{}{"task_id": "123"},
			mockPost: func(_ context.Context, _ string, _ interface{}) ([]byte, error) {
				return nil, fmt.Errorf("forbidden")
			},
			wantErr:   true,
			errSubstr: "failed to complete task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := CompleteTaskHandler(client)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
			if !strings.Contains(text, "Task completed successfully") {
				t.Errorf("expected success message, got: %s", text)
			}
		})
	}
}

func TestUncompleteTaskHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"task_id": "123"},
			mockPost: func(_ context.Context, path string, _ interface{}) ([]byte, error) {
				if path != "/tasks/123/reopen" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return nil, nil
			},
		},
		{
			name:      "missing task_id",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "task_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := UncompleteTaskHandler(client)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
			if !strings.Contains(text, "Task reopened successfully") {
				t.Errorf("expected success message, got: %s", text)
			}
		})
	}
}

func TestDeleteTaskHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockDel   func(ctx context.Context, path string) error
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"task_id": "123"},
			mockDel: func(_ context.Context, path string) error {
				if path != "/tasks/123" {
					return fmt.Errorf("unexpected path: %s", path)
				}
				return nil
			},
		},
		{
			name:      "missing task_id",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "task_id is required",
		},
		{
			name: "API error",
			args: map[string]interface{}{"task_id": "123"},
			mockDel: func(_ context.Context, _ string) error {
				return fmt.Errorf("not found")
			},
			wantErr:   true,
			errSubstr: "failed to delete task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{DeleteFn: tt.mockDel}
			handler := DeleteTaskHandler(client)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
			if !strings.Contains(text, "Task deleted successfully") {
				t.Errorf("expected success message, got: %s", text)
			}
		})
	}
}

func TestQuickAddTaskHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockGet   func(ctx context.Context, path string) ([]byte, error)
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "simple task",
			args: map[string]interface{}{"content": "Buy groceries"},
			mockPost: func(_ context.Context, _ string, body interface{}) ([]byte, error) {
				return json.Marshal(map[string]interface{}{"id": "1", "content": "Buy groceries"})
			},
		},
		{
			name: "with priority p1",
			args: map[string]interface{}{"content": "Fix bug p1"},
			mockPost: func(_ context.Context, _ string, body interface{}) ([]byte, error) {
				b := body.(map[string]interface{})
				if b["priority"] != 4 {
					return nil, fmt.Errorf("expected priority 4 for p1, got %v", b["priority"])
				}
				return json.Marshal(map[string]interface{}{"id": "1", "content": "Fix bug"})
			},
		},
		{
			name: "with label",
			args: map[string]interface{}{"content": "Review PR @work"},
			mockPost: func(_ context.Context, _ string, body interface{}) ([]byte, error) {
				b := body.(map[string]interface{})
				labels := b["labels"].([]string)
				if len(labels) != 1 || labels[0] != "work" {
					return nil, fmt.Errorf("expected labels [work], got %v", labels)
				}
				return json.Marshal(map[string]interface{}{"id": "1", "content": "Review PR"})
			},
		},
		{
			name: "with project match",
			args: map[string]interface{}{"content": "Task #MyProject"},
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				return json.Marshal([]map[string]interface{}{
					{"id": "proj1", "name": "MyProject"},
				})
			},
			mockPost: func(_ context.Context, _ string, body interface{}) ([]byte, error) {
				b := body.(map[string]interface{})
				if b["project_id"] != "proj1" {
					return nil, fmt.Errorf("expected project_id proj1, got %v", b["project_id"])
				}
				return json.Marshal(map[string]interface{}{"id": "1", "content": "Task"})
			},
		},
		{
			name:      "missing content",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{GetFn: tt.mockGet, PostFn: tt.mockPost}
			handler := QuickAddTaskHandler(client)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
		})
	}
}

func TestGetTaskStatsHandler(t *testing.T) {
	tests := []struct {
		name      string
		mockGet   func(ctx context.Context, path string) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				if path == "/tasks" {
					return json.Marshal([]map[string]interface{}{
						{"id": "1", "content": "Task 1", "priority": float64(4), "project_id": "p1"},
						{"id": "2", "content": "Task 2", "priority": float64(1), "project_id": "p1"},
					})
				}
				if path == "/projects" {
					return json.Marshal([]map[string]interface{}{
						{"id": "p1", "name": "Inbox"},
					})
				}
				return nil, fmt.Errorf("unexpected path: %s", path)
			},
		},
		{
			name: "tasks API error",
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				if path == "/tasks" {
					return nil, fmt.Errorf("timeout")
				}
				return json.Marshal([]map[string]interface{}{})
			},
			wantErr:   true,
			errSubstr: "failed to fetch tasks",
		},
		{
			name: "projects API error",
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				if path == "/tasks" {
					return json.Marshal([]map[string]interface{}{})
				}
				return nil, fmt.Errorf("timeout")
			},
			wantErr:   true,
			errSubstr: "failed to fetch projects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{GetFn: tt.mockGet}
			handler := GetTaskStatsHandler(client)
			result, err := handler(context.Background(), makeReq(nil))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
			var stats map[string]interface{}
			if err := json.Unmarshal([]byte(text), &stats); err != nil {
				t.Fatalf("failed to parse stats: %v", err)
			}
			if int(stats["total_active"].(float64)) != 2 {
				t.Errorf("total_active = %v, want 2", stats["total_active"])
			}
		})
	}
}

func TestBulkCompleteTasksHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockGet   func(ctx context.Context, path string) ([]byte, error)
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		mockBatch func(ctx context.Context, commands []todoist.Command) (*todoist.SyncResponse, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "REST path with few task_ids",
			args: map[string]interface{}{
				"task_ids": []interface{}{"1", "2", "3"},
			},
			mockPost: func(_ context.Context, _ string, _ interface{}) ([]byte, error) {
				return nil, nil
			},
		},
		{
			name: "batch path with many task_ids",
			args: map[string]interface{}{
				"task_ids": []interface{}{"1", "2", "3", "4", "5", "6"},
			},
			mockBatch: func(_ context.Context, commands []todoist.Command) (*todoist.SyncResponse, error) {
				status := make(map[string]interface{})
				for _, cmd := range commands {
					status[cmd.UUID] = "ok"
				}
				return &todoist.SyncResponse{SyncStatus: status}, nil
			},
		},
		{
			name: "with filter",
			args: map[string]interface{}{"filter": "today"},
			mockGet: func(_ context.Context, _ string) ([]byte, error) {
				return json.Marshal([]map[string]interface{}{
					{"id": "1"},
					{"id": "2"},
				})
			},
			mockPost: func(_ context.Context, _ string, _ interface{}) ([]byte, error) {
				return nil, nil
			},
		},
		{
			name:      "no task_ids or filter",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "either task_ids or filter must be provided",
		},
		{
			name: "batch API error",
			args: map[string]interface{}{
				"task_ids": []interface{}{"1", "2", "3", "4", "5", "6"},
			},
			mockBatch: func(_ context.Context, _ []todoist.Command) (*todoist.SyncResponse, error) {
				return nil, fmt.Errorf("sync error")
			},
			wantErr:   true,
			errSubstr: "failed to batch complete tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{GetFn: tt.mockGet, PostFn: tt.mockPost}
			syncClient := &MockSyncAPI{BatchCommandsFn: tt.mockBatch}
			handler := BulkCompleteTasksHandler(client, syncClient)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
		})
	}
}

func TestBatchCreateTasksHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockBatch func(ctx context.Context, commands []todoist.Command) (*todoist.SyncResponse, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{
				"tasks": []interface{}{
					map[string]interface{}{"content": "Task 1"},
					map[string]interface{}{"content": "Task 2", "priority": float64(3)},
				},
			},
			mockBatch: func(_ context.Context, commands []todoist.Command) (*todoist.SyncResponse, error) {
				status := make(map[string]interface{})
				mapping := make(map[string]string)
				for _, cmd := range commands {
					status[cmd.UUID] = "ok"
					mapping[cmd.TempID] = "real-" + cmd.TempID[:8]
				}
				return &todoist.SyncResponse{SyncStatus: status, TempIDMapping: mapping}, nil
			},
		},
		{
			name:      "empty tasks array",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "tasks array is required",
		},
		{
			name: "task missing content",
			args: map[string]interface{}{
				"tasks": []interface{}{
					map[string]interface{}{"description": "no content"},
				},
			},
			wantErr:   true,
			errSubstr: "missing required 'content' field",
		},
		{
			name: "batch API error",
			args: map[string]interface{}{
				"tasks": []interface{}{
					map[string]interface{}{"content": "Task 1"},
				},
			},
			mockBatch: func(_ context.Context, _ []todoist.Command) (*todoist.SyncResponse, error) {
				return nil, fmt.Errorf("sync error")
			},
			wantErr:   true,
			errSubstr: "failed to batch create tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncClient := &MockSyncAPI{BatchCommandsFn: tt.mockBatch}
			handler := BatchCreateTasksHandler(syncClient)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
		})
	}
}

func TestMoveTasksHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockGet   func(ctx context.Context, path string) ([]byte, error)
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		mockBatch func(ctx context.Context, commands []todoist.Command) (*todoist.SyncResponse, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "REST path with few tasks",
			args: map[string]interface{}{
				"task_ids":      []interface{}{"1", "2"},
				"to_project_id": "proj1",
			},
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				return json.Marshal(map[string]interface{}{"id": "proj1", "name": "Destination"})
			},
			mockPost: func(_ context.Context, _ string, _ interface{}) ([]byte, error) {
				return json.Marshal(map[string]interface{}{"id": "1"})
			},
		},
		{
			name:      "missing to_project_id",
			args:      map[string]interface{}{"task_ids": []interface{}{"1"}},
			wantErr:   true,
			errSubstr: "to_project_id is required",
		},
		{
			name:      "invalid to_project_id",
			args:      map[string]interface{}{"to_project_id": "../bad", "task_ids": []interface{}{"1"}},
			wantErr:   true,
			errSubstr: "contains invalid characters",
		},
		{
			name:      "no tasks provided",
			args:      map[string]interface{}{"to_project_id": "proj1"},
			wantErr:   true,
			errSubstr: "either task_ids or filter must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{GetFn: tt.mockGet, PostFn: tt.mockPost}
			syncClient := &MockSyncAPI{BatchCommandsFn: tt.mockBatch}
			handler := MoveTasksHandler(client, syncClient)
			result, err := handler(context.Background(), makeReq(tt.args))
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			text := resultText(result)
			if tt.wantErr {
				if !result.IsError {
					t.Fatal("expected tool error")
				}
				if !strings.Contains(text, tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", text, tt.errSubstr)
				}
				return
			}
			if result.IsError {
				t.Fatalf("unexpected tool error: %s", text)
			}
		})
	}
}
