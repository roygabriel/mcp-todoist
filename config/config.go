package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	TodoistAPIToken string
}

// Load reads configuration from environment variables and .env file.
func Load() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	apiToken := os.Getenv("TODOIST_API_TOKEN")

	if apiToken == "" {
		return nil, fmt.Errorf("TODOIST_API_TOKEN environment variable is required (get your token from https://todoist.com/prefs/integrations)")
	}

	// Support loading token from a file (useful for Kubernetes secrets / Docker secrets)
	if strings.HasPrefix(apiToken, "file://") {
		path := filepath.Clean(strings.TrimPrefix(apiToken, "file://"))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read API token from file %s: %w", path, err)
		}
		apiToken = strings.TrimSpace(string(data))
		if apiToken == "" {
			return nil, fmt.Errorf("API token file %s is empty", path)
		}
	}

	cfg := &Config{TodoistAPIToken: apiToken}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks that the configuration values are well-formed.
func (c *Config) Validate() error {
	if c.TodoistAPIToken == "" {
		return fmt.Errorf("API token is required")
	}
	if len(c.TodoistAPIToken) < 20 {
		return fmt.Errorf("API token appears too short (got %d characters)", len(c.TodoistAPIToken))
	}
	if len(c.TodoistAPIToken) > 200 {
		return fmt.Errorf("API token appears too long (got %d characters)", len(c.TodoistAPIToken))
	}
	for _, r := range c.TodoistAPIToken {
		if unicode.IsSpace(r) {
			return fmt.Errorf("API token contains whitespace characters")
		}
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("API token contains control characters")
		}
	}
	return nil
}
