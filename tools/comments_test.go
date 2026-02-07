package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestGetCommentsHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockGet   func(ctx context.Context, path string) ([]byte, error)
		wantErr   bool
		wantCount int
		errSubstr string
	}{
		{
			name: "with task_id",
			args: map[string]interface{}{"task_id": "123"},
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				if !strings.Contains(path, "task_id=123") {
					return nil, fmt.Errorf("expected task_id in path, got: %s", path)
				}
				return json.Marshal([]map[string]interface{}{
					{"id": "c1", "content": "Comment 1"},
				})
			},
			wantCount: 1,
		},
		{
			name: "with project_id",
			args: map[string]interface{}{"project_id": "proj1"},
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				if !strings.Contains(path, "project_id=proj1") {
					return nil, fmt.Errorf("expected project_id in path, got: %s", path)
				}
				return json.Marshal([]map[string]interface{}{})
			},
			wantCount: 0,
		},
		{
			name:      "no filter",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "either task_id or project_id is required",
		},
		{
			name:      "invalid task_id",
			args:      map[string]interface{}{"task_id": "../bad"},
			wantErr:   true,
			errSubstr: "contains invalid characters",
		},
		{
			name:      "invalid project_id",
			args:      map[string]interface{}{"project_id": "a/b"},
			wantErr:   true,
			errSubstr: "contains invalid characters",
		},
		{
			name: "API error",
			args: map[string]interface{}{"task_id": "123"},
			mockGet: func(_ context.Context, _ string) ([]byte, error) {
				return nil, fmt.Errorf("timeout")
			},
			wantErr:   true,
			errSubstr: "failed to get comments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{GetFn: tt.mockGet}
			handler := GetCommentsHandler(client)
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

func TestAddCommentHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "with task_id",
			args: map[string]interface{}{"content": "Great!", "task_id": "123"},
			mockPost: func(_ context.Context, path string, body interface{}) ([]byte, error) {
				if path != "/comments" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				b := body.(map[string]interface{})
				if b["task_id"] != "123" {
					return nil, fmt.Errorf("expected task_id 123")
				}
				return json.Marshal(map[string]interface{}{"id": "c1", "content": "Great!"})
			},
		},
		{
			name: "with project_id",
			args: map[string]interface{}{"content": "Note", "project_id": "proj1"},
			mockPost: func(_ context.Context, _ string, _ interface{}) ([]byte, error) {
				return json.Marshal(map[string]interface{}{"id": "c1", "content": "Note"})
			},
		},
		{
			name:      "missing content",
			args:      map[string]interface{}{"task_id": "123"},
			wantErr:   true,
			errSubstr: "content is required",
		},
		{
			name:      "no target",
			args:      map[string]interface{}{"content": "orphan"},
			wantErr:   true,
			errSubstr: "either task_id or project_id is required",
		},
		{
			name: "API error",
			args: map[string]interface{}{"content": "x", "task_id": "123"},
			mockPost: func(_ context.Context, _ string, _ interface{}) ([]byte, error) {
				return nil, fmt.Errorf("server error")
			},
			wantErr:   true,
			errSubstr: "failed to add comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := AddCommentHandler(client)
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

func TestUpdateCommentHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"comment_id": "c1", "content": "Updated"},
			mockPost: func(_ context.Context, path string, _ interface{}) ([]byte, error) {
				if path != "/comments/c1" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return json.Marshal(map[string]interface{}{"id": "c1", "content": "Updated"})
			},
		},
		{
			name:      "missing comment_id",
			args:      map[string]interface{}{"content": "x"},
			wantErr:   true,
			errSubstr: "comment_id is required",
		},
		{
			name:      "missing content",
			args:      map[string]interface{}{"comment_id": "c1"},
			wantErr:   true,
			errSubstr: "content is required",
		},
		{
			name:      "invalid comment_id",
			args:      map[string]interface{}{"comment_id": "../bad", "content": "x"},
			wantErr:   true,
			errSubstr: "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := UpdateCommentHandler(client)
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

func TestDeleteCommentHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockDel   func(ctx context.Context, path string) error
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"comment_id": "c1"},
			mockDel: func(_ context.Context, path string) error {
				if path != "/comments/c1" {
					return fmt.Errorf("unexpected path: %s", path)
				}
				return nil
			},
		},
		{
			name:      "missing comment_id",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "comment_id is required",
		},
		{
			name: "API error",
			args: map[string]interface{}{"comment_id": "c1"},
			mockDel: func(_ context.Context, _ string) error {
				return fmt.Errorf("not found")
			},
			wantErr:   true,
			errSubstr: "failed to delete comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{DeleteFn: tt.mockDel}
			handler := DeleteCommentHandler(client)
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
			if !strings.Contains(text, "Comment deleted successfully") {
				t.Errorf("expected success message, got: %s", text)
			}
		})
	}
}
