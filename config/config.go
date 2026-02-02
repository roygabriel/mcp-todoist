package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	TodoistAPIToken string
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	apiToken := os.Getenv("TODOIST_API_TOKEN")

	// Validate required fields
	if apiToken == "" {
		return nil, fmt.Errorf("TODOIST_API_TOKEN environment variable is required (get your token from https://todoist.com/prefs/integrations)")
	}

	return &Config{
		TodoistAPIToken: apiToken,
	}, nil
}
