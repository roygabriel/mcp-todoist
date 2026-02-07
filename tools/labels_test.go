package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestListLabelsHandler(t *testing.T) {
	tests := []struct {
		name      string
		mockGet   func(ctx context.Context, path string) ([]byte, error)
		wantErr   bool
		wantCount int
		errSubstr string
	}{
		{
			name: "happy path",
			mockGet: func(_ context.Context, _ string) ([]byte, error) {
				return json.Marshal([]map[string]interface{}{
					{"id": "1", "name": "urgent"},
					{"id": "2", "name": "work"},
				})
			},
			wantCount: 2,
		},
		{
			name: "API error",
			mockGet: func(_ context.Context, _ string) ([]byte, error) {
				return nil, fmt.Errorf("unauthorized")
			},
			wantErr:   true,
			errSubstr: "failed to list labels",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{GetFn: tt.mockGet}
			handler := ListLabelsHandler(client)
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

func TestCreateLabelHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"name": "urgent"},
			mockPost: func(_ context.Context, path string, _ interface{}) ([]byte, error) {
				if path != "/labels" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return json.Marshal(map[string]interface{}{"id": "1", "name": "urgent"})
			},
		},
		{
			name: "with optional fields",
			args: map[string]interface{}{
				"name":        "urgent",
				"color":       "red",
				"order":       float64(1),
				"is_favorite": true,
			},
			mockPost: func(_ context.Context, _ string, body interface{}) ([]byte, error) {
				b := body.(map[string]interface{})
				if b["color"] != "red" {
					return nil, fmt.Errorf("missing color")
				}
				return json.Marshal(map[string]interface{}{"id": "1", "name": "urgent"})
			},
		},
		{
			name:      "missing name",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := CreateLabelHandler(client)
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

func TestUpdateLabelHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockPost  func(ctx context.Context, path string, body interface{}) ([]byte, error)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"label_id": "123", "name": "Renamed"},
			mockPost: func(_ context.Context, path string, _ interface{}) ([]byte, error) {
				if path != "/labels/123" {
					return nil, fmt.Errorf("unexpected path: %s", path)
				}
				return json.Marshal(map[string]interface{}{"id": "123", "name": "Renamed"})
			},
		},
		{
			name:      "missing label_id",
			args:      map[string]interface{}{"name": "x"},
			wantErr:   true,
			errSubstr: "label_id is required",
		},
		{
			name:      "no fields to update",
			args:      map[string]interface{}{"label_id": "123"},
			wantErr:   true,
			errSubstr: "at least one field to update",
		},
		{
			name:      "invalid label_id",
			args:      map[string]interface{}{"label_id": "../bad", "name": "x"},
			wantErr:   true,
			errSubstr: "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{PostFn: tt.mockPost}
			handler := UpdateLabelHandler(client)
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

func TestDeleteLabelHandler(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		mockDel   func(ctx context.Context, path string) error
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			args: map[string]interface{}{"label_id": "123"},
			mockDel: func(_ context.Context, path string) error {
				if path != "/labels/123" {
					return fmt.Errorf("unexpected path: %s", path)
				}
				return nil
			},
		},
		{
			name:      "missing label_id",
			args:      map[string]interface{}{},
			wantErr:   true,
			errSubstr: "label_id is required",
		},
		{
			name: "API error",
			args: map[string]interface{}{"label_id": "123"},
			mockDel: func(_ context.Context, _ string) error {
				return fmt.Errorf("not found")
			},
			wantErr:   true,
			errSubstr: "failed to delete label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &MockAPI{DeleteFn: tt.mockDel}
			handler := DeleteLabelHandler(client)
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
			if !strings.Contains(text, "Label deleted successfully") {
				t.Errorf("expected success message, got: %s", text)
			}
		})
	}
}
