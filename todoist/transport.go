package todoist

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// newHTTPTransport returns a hardened HTTP transport with granular timeouts.
func newHTTPTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout:  5 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    10,
		IdleConnTimeout:        90 * time.Second,
		DisableCompression:     false,
	}
}

// readResponseBody reads up to maxResponseBytes from r. Returns an error if
// the response exceeds the limit.
func readResponseBody(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxResponseBytes+1))
	if err != nil {
		return nil, &RetryableError{err: fmt.Errorf("failed to read response: %w", err)}
	}
	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes limit", maxResponseBytes)
	}
	return data, nil
}
