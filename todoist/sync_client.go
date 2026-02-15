package todoist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

const (
	syncBaseURL = "https://api.todoist.com/api/v1/sync"
)

// SyncClient wraps the HTTP client for Todoist Sync API v1.
type SyncClient struct {
	httpClient  *http.Client
	apiToken    string
	rateLimiter *RateLimiter
	breaker     *CircuitBreaker
}

// Command represents a Sync API command.
type Command struct {
	Type   string                 `json:"type"`
	UUID   string                 `json:"uuid"`
	TempID string                 `json:"temp_id,omitempty"`
	Args   map[string]interface{} `json:"args"`
}

// SyncResponse represents a Sync API response.
type SyncResponse struct {
	SyncToken     string                 `json:"sync_token"`
	SyncStatus    map[string]interface{} `json:"sync_status"`
	TempIDMapping map[string]string      `json:"temp_id_mapping"`
	FullSync      bool                   `json:"full_sync"`
}

// NewSyncClient creates a new Todoist Sync API client with a shared rate limiter
// and circuit breaker.
func NewSyncClient(apiToken string, rl *RateLimiter, cb *CircuitBreaker) *SyncClient {
	return &SyncClient{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: newHTTPTransport(),
		},
		apiToken:    apiToken,
		rateLimiter: rl,
		breaker:     cb,
	}
}

// BatchCommands sends multiple commands in a single Sync API request.
// Retried automatically on transient failures because command UUIDs provide idempotency.
func (sc *SyncClient) BatchCommands(ctx context.Context, commands []Command) (*SyncResponse, error) {
	var result *SyncResponse
	err := retryWithBackoff(ctx, maxAttempts, func() error {
		var reqErr error
		result, reqErr = sc.doBatchRequest(ctx, commands)
		return reqErr
	})
	return result, err
}

func (sc *SyncClient) doBatchRequest(ctx context.Context, commands []Command) (*SyncResponse, error) {
	done, err := sc.breaker.Allow()
	if err != nil {
		return nil, err
	}

	if err := sc.rateLimiter.Check(); err != nil {
		done(true) // rate limit is local, not an upstream failure
		return nil, err
	}

	commandsJSON, err := json.Marshal(commands)
	if err != nil {
		done(true)
		return nil, fmt.Errorf("failed to marshal commands: %w", err)
	}

	formData := url.Values{}
	formData.Set("commands", string(commandsJSON))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, syncBaseURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		done(true)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+sc.apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		done(false) // connection failure counts as breaker failure
		return nil, &RetryableError{err: fmt.Errorf("request failed: %w", err)}
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := readResponseBody(resp.Body)
	if err != nil {
		done(false)
		return nil, err
	}

	if resp.StatusCode >= 500 {
		done(false) // 5xx counts as breaker failure
		return nil, handleHTTPError(resp.StatusCode, respBody)
	}

	done(true) // 4xx is not a breaker failure
	if resp.StatusCode >= 400 {
		return nil, handleHTTPError(resp.StatusCode, respBody)
	}

	var syncResp SyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse sync response: %w", err)
	}

	return &syncResp, nil
}

// GetRemainingRequests returns how many requests are available in the current window.
func (sc *SyncClient) GetRemainingRequests() int {
	return sc.rateLimiter.Remaining()
}

// GenerateUUID generates a new UUID for command identification.
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateTempID generates a temporary ID for new resources.
func GenerateTempID() string {
	return uuid.New().String()
}
