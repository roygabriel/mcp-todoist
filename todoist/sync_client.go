package todoist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	syncBaseURL = "https://api.todoist.com/api/v1/sync"
)

// SyncClient wraps the HTTP client for Todoist Sync API v1
type SyncClient struct {
	httpClient *http.Client
	apiToken   string
	syncToken  string // For incremental sync if needed
	
	// Share rate limiting with REST client
	mu            sync.Mutex
	requestTimes  []time.Time
}

// Command represents a Sync API command
type Command struct {
	Type   string                 `json:"type"`
	UUID   string                 `json:"uuid"`
	TempID string                 `json:"temp_id,omitempty"`
	Args   map[string]interface{} `json:"args"`
}

// SyncResponse represents Sync API response
type SyncResponse struct {
	SyncToken     string                     `json:"sync_token"`
	SyncStatus    map[string]interface{}     `json:"sync_status"`
	TempIDMapping map[string]string          `json:"temp_id_mapping"`
	FullSync      bool                       `json:"full_sync"`
}

// NewSyncClient creates a new Todoist Sync API client
func NewSyncClient(apiToken string) *SyncClient {
	return &SyncClient{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
		apiToken:     apiToken,
		requestTimes: make([]time.Time, 0),
	}
}

// checkRateLimit checks if we're approaching the rate limit
func (sc *SyncClient) checkRateLimit() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)
	
	// Remove old request times
	validRequests := make([]time.Time, 0)
	for _, t := range sc.requestTimes {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	sc.requestTimes = validRequests
	
	// Check if we're at the limit
	if len(sc.requestTimes) >= maxRequests {
		return fmt.Errorf("rate limit reached: %d requests in the last 15 minutes (max: %d)", len(sc.requestTimes), maxRequests)
	}
	
	// Add current request
	sc.requestTimes = append(sc.requestTimes, now)
	
	return nil
}

// GetRemainingRequests returns how many requests are available in current window
func (sc *SyncClient) GetRemainingRequests() int {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)
	
	// Count valid requests in current window
	validCount := 0
	for _, t := range sc.requestTimes {
		if t.After(cutoff) {
			validCount++
		}
	}
	
	return maxRequests - validCount
}

// BatchCommands sends multiple commands in a single Sync API request
func (sc *SyncClient) BatchCommands(ctx context.Context, commands []Command) (*SyncResponse, error) {
	// Check rate limit
	if err := sc.checkRateLimit(); err != nil {
		return nil, err
	}
	
	// Marshal commands to JSON
	commandsJSON, err := json.Marshal(commands)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal commands: %w", err)
	}
	
	// Create form-encoded request body
	formData := url.Values{}
	formData.Set("commands", string(commandsJSON))
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, syncBaseURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers
	req.Header.Set("Authorization", "Bearer "+sc.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Execute request
	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Handle HTTP errors
	if resp.StatusCode >= 400 {
		return nil, sc.handleHTTPError(resp.StatusCode, respBody)
	}
	
	// Parse response
	var syncResp SyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse sync response: %w", err)
	}
	
	return &syncResp, nil
}

// handleHTTPError converts HTTP error responses to meaningful error messages
func (sc *SyncClient) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case 401:
		return fmt.Errorf("authentication failed: invalid API token (get a valid token from https://todoist.com/prefs/integrations)")
	case 403:
		return fmt.Errorf("access forbidden: you don't have permission to access this resource")
	case 404:
		return fmt.Errorf("resource not found: the requested item doesn't exist")
	case 429:
		return fmt.Errorf("rate limit exceeded: too many requests. Please wait and try again")
	case 500, 502, 503, 504:
		return fmt.Errorf("Todoist server error (status %d): please try again later", statusCode)
	default:
		if len(body) > 0 {
			return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
		}
		return fmt.Errorf("API error: unexpected status code %d", statusCode)
	}
}

// GenerateUUID generates a new UUID for command identification
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateTempID generates a temporary ID for new resources
func GenerateTempID() string {
	return uuid.New().String()
}
