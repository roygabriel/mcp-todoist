package todoist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL        = "https://api.todoist.com/rest/v2"
	timeout        = 30 * time.Second
	rateLimitWindow = 15 * time.Minute
	maxRequests    = 450
)

// Client wraps the HTTP client with Todoist-specific functionality
type Client struct {
	httpClient *http.Client
	apiToken   string
	
	// Rate limiting
	mu            sync.Mutex
	requestTimes  []time.Time
}

// NewClient creates a new Todoist API client
func NewClient(apiToken string) *Client {
	return &Client{
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
func (c *Client) checkRateLimit() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)
	
	// Remove old request times
	validRequests := make([]time.Time, 0)
	for _, t := range c.requestTimes {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}
	c.requestTimes = validRequests
	
	// Check if we're at the limit
	if len(c.requestTimes) >= maxRequests {
		return fmt.Errorf("rate limit reached: %d requests in the last 15 minutes (max: %d)", len(c.requestTimes), maxRequests)
	}
	
	// Add current request
	c.requestTimes = append(c.requestTimes, now)
	
	return nil
}

// doRequest performs an HTTP request with proper headers and error handling
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Check rate limit
	if err := c.checkRateLimit(); err != nil {
		return nil, err
	}
	
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}
	
	url := baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	// Execute request
	resp, err := c.httpClient.Do(req)
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
		return nil, c.handleHTTPError(resp.StatusCode, respBody)
	}
	
	return respBody, nil
}

// handleHTTPError converts HTTP error responses to meaningful error messages
func (c *Client) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case 401:
		return fmt.Errorf("authentication failed: invalid API token (get a valid token from https://todoist.com/prefs/integrations)")
	case 403:
		return fmt.Errorf("access forbidden: you don't have permission to access this resource")
	case 404:
		return fmt.Errorf("resource not found: the requested item doesn't exist")
	case 429:
		return fmt.Errorf("rate limit exceeded: too many requests (max 450 per 15 minutes). Please wait and try again")
	case 500, 502, 503, 504:
		return fmt.Errorf("Todoist server error (status %d): please try again later", statusCode)
	default:
		if len(body) > 0 {
			return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
		}
		return fmt.Errorf("API error: unexpected status code %d", statusCode)
	}
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// TestConnection tests the API connection by fetching projects
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.Get(ctx, "/projects")
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	return nil
}
