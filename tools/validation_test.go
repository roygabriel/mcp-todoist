package tools

import "testing"

func TestValidateID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		paramName string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid numeric ID",
			id:        "12345",
			paramName: "task_id",
		},
		{
			name:      "valid alphanumeric ID",
			id:        "abc123XYZ",
			paramName: "task_id",
		},
		{
			name:      "empty ID",
			id:        "",
			paramName: "task_id",
			wantErr:   true,
			errMsg:    "task_id is required",
		},
		{
			name:      "path traversal with ..",
			id:        "../etc/passwd",
			paramName: "project_id",
			wantErr:   true,
			errMsg:    "project_id contains invalid characters",
		},
		{
			name:      "ID with slash",
			id:        "abc/def",
			paramName: "label_id",
			wantErr:   true,
			errMsg:    "label_id contains invalid characters",
		},
		{
			name:      "ID with null byte",
			id:        "abc\x00def",
			paramName: "comment_id",
			wantErr:   true,
			errMsg:    "comment_id contains invalid characters",
		},
		{
			name:      "ID with tab",
			id:        "abc\tdef",
			paramName: "section_id",
			wantErr:   true,
			errMsg:    "section_id contains invalid characters",
		},
		{
			name:      "ID with DEL character",
			id:        "abc\x7fdef",
			paramName: "task_id",
			wantErr:   true,
			errMsg:    "task_id contains invalid characters",
		},
		{
			name:      "ID with double dots only",
			id:        "..",
			paramName: "task_id",
			wantErr:   true,
			errMsg:    "task_id contains invalid characters",
		},
		{
			name:      "valid ID with single dot",
			id:        "abc.def",
			paramName: "task_id",
		},
		{
			name:      "valid ID with hyphen",
			id:        "abc-def-123",
			paramName: "task_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateID(tt.id, tt.paramName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
