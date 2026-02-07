package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestListSectionsHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockGet   func(ctx context.Context, path string) ([]byte, error)
		wantErr   bool
		wantCount int
		errSubstr string
	}{
		{
			name: "no filter",
			args: map[string]interface{}{},
			mockGet: func(_ context.Context, path string) ([]byte, error) {
				if path != "/sections" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return json.Marshal([]map[string]interface{}{
					{"id": "1", "name": "Backlog"},
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
				return json.Marshal([]map[string]interface{}{
					{"id": "1", "name": "Backlog"},
					{"id": "2", "name": "In Progress"},
				})
			},
			wantCount: 2,
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
				return nil, fmt.Errorf("timeout")
			},
			wantErr:   true,
			errSubstr: "failed to list sections",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{GetFn: tt.mockGet}
			handler := ListSectionsHandler(client)
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

func TestCreateSectionHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"name": "Backlog", "project_id": "proj1"},
			mockPost: func(_ context.Context, path string, _ interface{}) ([]byte, error) {
				if path != "/sections" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return json.Marshal(map[string]interface{}{"id": "1", "name": "Backlog"})
			},
		},
		{
			name:      "missing name",
			args:      map[string]interface{}{"project_id": "proj1"},
			wantErr:   true,
			errSubstr: "name is required",
		},
		{
			name:      "missing project_id",
			args:      map[string]interface{}{"name": "Backlog"},
			wantErr:   true,
			errSubstr: "project_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := CreateSectionHandler(client)
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

func TestUpdateSectionHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"section_id": "123", "name": "Renamed"},
			mockPost: func(_ context.Context, path string, _ interface{}) ([]byte, error) {
				if path != "/sections/123" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return json.Marshal(map[string]interface{}{"id": "123", "name": "Renamed"})
			},
		},
		{
			name:      "missing section_id",
			args:      map[string]interface{}{"name": "x"},
			wantErr:   true,
			errSubstr: "section_id is required",
		},
		{
			name:      "missing name",
			args:      map[string]interface{}{"section_id": "123"},
			wantErr:   true,
			errSubstr: "name is required",
		},
		{
			name:      "invalid section_id",
			args:      map[string]interface{}{"section_id": "a/b", "name": "x"},
			wantErr:   true,
			errSubstr: "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := UpdateSectionHandler(client)
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

func TestDeleteSectionHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockDel   func(ctx context.Context, path string) error
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"section_id": "123"},
			mockDel: func(_ context.Context, path string) error {
				if path != "/sections/123" {
					return fmt.Errorf("unexpected path: %s", path)
				}
				return nil
			},
		},
		{
			name:      "missing section_id",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "section_id is required",
		},
		{
			name: "API error",
			args: map[string]interface{}{"section_id": "123"},
			mockDel: func(_ context.Context, _ string) error {
				return fmt.Errorf("forbidden")
			},
			wantErr:   true,
			errSubstr: "failed to delete section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{DeleteFn: tt.mockDel}
			handler := DeleteSectionHandler(client)
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
			if !strings.Contains(text, "Section deleted successfully") {
				t.Errorf("expected success message, got: %s", text)
			}
		})
	}
}
