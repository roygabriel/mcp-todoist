package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantErr   bool
		errSubstr string
	}{
		{
			name:  "valid 40-char hex token",
			token: "abcdef1234567890abcdef1234567890abcdef12",
		},
		{
			name:  "valid long token",
			token: strings.Repeat("a", 100),
		},
		{
			name:      "empty token",
			token:     "",
			wantErr:   true,
			errSubstr: "is required",
		},
		{
			name:      "too short",
			token:     "abc123",
			wantErr:   true,
			errSubstr: "too short",
		},
		{
			name:      "too long",
			token:     strings.Repeat("a", 201),
			wantErr:   true,
			errSubstr: "too long",
		},
		{
			name:      "contains space",
			token:     "abcdef1234567890 bcdef1234567890abcdef12",
			wantErr:   true,
			errSubstr: "whitespace",
		},
		{
			name:      "contains tab",
			token:     "abcdef1234567890\tbcdef1234567890abcdef12",
			wantErr:   true,
			errSubstr: "whitespace",
		},
		{
			name:      "contains newline",
			token:     "abcdef1234567890\nbcdef1234567890abcdef12",
			wantErr:   true,
			errSubstr: "whitespace",
		},
		{
			name:      "contains null byte",
			token:     "abcdef1234567890\x00bcdef1234567890abcdef12",
			wantErr:   true,
			errSubstr: "control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{TodoistAPIToken: tt.token}
			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error = %q, want substring %q", err.Error(), tt.errSubstr)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoad_FilePrefix(t *testing.T) {
	// Create a temp file with a valid token
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	validToken := "abcdef1234567890abcdef1234567890abcdef12"
	if err := os.WriteFile(tokenFile, []byte(validToken+"\n"), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	t.Setenv("TODOIST_API_TOKEN", "file://"+tokenFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.TodoistAPIToken != validToken {
		t.Errorf("token = %q, want %q", cfg.TodoistAPIToken, validToken)
	}
}

func TestLoad_FilePrefix_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenFile, []byte("  \n"), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	t.Setenv("TODOIST_API_TOKEN", "file://"+tokenFile)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if !strings.Contains(err.Error(), "is empty") {
		t.Errorf("error = %q, want substring 'is empty'", err.Error())
	}
}

func TestLoad_FilePrefix_MissingFile(t *testing.T) {
	t.Setenv("TODOIST_API_TOKEN", "file:///nonexistent/path/token")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to read API token") {
		t.Errorf("error = %q, want substring 'failed to read API token'", err.Error())
	}
}

func TestLoad_MissingEnvVar(t *testing.T) {
	t.Setenv("TODOIST_API_TOKEN", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
	if !strings.Contains(err.Error(), "is required") {
		t.Errorf("error = %q, want substring 'is required'", err.Error())
	}
}
