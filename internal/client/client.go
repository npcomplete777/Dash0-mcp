// Package client provides the HTTP client for Dash0 API requests.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/npcomplete777/dash0-mcp/internal/config"
)

// Client handles authenticated HTTP requests to the Dash0 API.
type Client struct {
	baseURL    string
	authToken  string
	dataset    string
	httpClient *http.Client
	debug      bool
	maxRetries int
}

// New creates a new Dash0 API client from configuration.
func New(cfg *config.Config) *Client {
	return &Client{
		baseURL:    cfg.BaseURL,
		authToken:  cfg.AuthToken,
		dataset:    cfg.Dataset,
		debug:      cfg.Debug,
		maxRetries: 3,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewWithBaseURL creates a new Dash0 API client with a custom base URL.
// This is primarily used for testing with mock servers.
func NewWithBaseURL(baseURL, authToken string) *Client {
	return &Client{
		baseURL:    baseURL,
		authToken:  authToken,
		debug:      false,
		maxRetries: 3,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ToolResult represents the result of an MCP tool call.
type ToolResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// APIError represents a Dash0 API error.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Title      string `json:"title,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

// ErrorResult creates an error ToolResult.
func ErrorResult(statusCode int, message string) *ToolResult {
	return &ToolResult{
		Success: false,
		Error: &APIError{
			StatusCode: statusCode,
			Detail:     message,
		},
	}
}

// GetDataset returns the configured dataset name.
func (c *Client) GetDataset() string {
	return c.dataset
}

// PostWithDataset performs a POST request with a specific dataset override.
// If dataset is non-empty, it overrides the global dataset for this request.
func (c *Client) PostWithDataset(ctx context.Context, path string, body interface{}, dataset string) *ToolResult {
	if dataset != "" {
		return c.requestWithDataset(ctx, http.MethodPost, path, body, dataset)
	}
	return c.Request(ctx, http.MethodPost, path, body)
}

// requestWithDataset performs an HTTP request with a specific dataset, overriding the global one.
func (c *Client) requestWithDataset(ctx context.Context, method, path string, body interface{}, dataset string) *ToolResult {
	requestURL := c.baseURL + path

	if strings.Contains(requestURL, "?") {
		requestURL = requestURL + "&dataset=" + url.QueryEscape(dataset)
	} else {
		requestURL = requestURL + "?dataset=" + url.QueryEscape(dataset)
	}

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return ErrorResult(http.StatusBadRequest, fmt.Sprintf("failed to marshal request body: %v", err))
		}
	}

	var resp *http.Response
	var respBody []byte

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
		if err != nil {
			return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("failed to create request: %v", err))
		}

		req.Header.Set("Authorization", "Bearer "+c.authToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("request failed: %v", err))
		}

		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && attempt < c.maxRetries {
			var waitDuration time.Duration
			if resp.StatusCode == http.StatusTooManyRequests {
				if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
					if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil && seconds > 0 {
						waitDuration = time.Duration(seconds) * time.Second
					} else {
						waitDuration = time.Second * (1 << uint(attempt))
					}
				} else {
					waitDuration = time.Second * (1 << uint(attempt))
				}
			} else {
				waitDuration = time.Second * (1 << uint(attempt))
			}

			resp.Body.Close()

			select {
			case <-ctx.Done():
				return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("request cancelled during retry: %v", ctx.Err()))
			case <-time.After(waitDuration):
			}
			continue
		}

		break
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
	}

	var result interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			result = string(respBody)
		}
	}

	if resp.StatusCode >= 400 {
		return &ToolResult{
			Success: false,
			Error: &APIError{
				StatusCode: resp.StatusCode,
				Title:      resp.Status,
				Detail:     extractErrorDetail(result),
			},
			Data: result,
		}
	}

	return SuccessResult(result)
}

// SuccessResult creates a success ToolResult.
func SuccessResult(data interface{}) *ToolResult {
	return &ToolResult{
		Success: true,
		Data:    data,
	}
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string) *ToolResult {
	return c.Request(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}) *ToolResult {
	return c.Request(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}) *ToolResult {
	return c.Request(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) *ToolResult {
	return c.Request(ctx, http.MethodDelete, path, nil)
}

// Request performs an HTTP request to the Dash0 API.
func (c *Client) Request(ctx context.Context, method, path string, body interface{}) *ToolResult {
	requestURL := c.baseURL + path

	// Add dataset as query parameter for all request methods
	if c.dataset != "" {
		requestURL = c.addDatasetQueryParam(requestURL)
	}

	// Marshal the body once so we can re-use it across retries.
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return ErrorResult(http.StatusBadRequest, fmt.Sprintf("failed to marshal request body: %v", err))
		}
	}

	var resp *http.Response
	var respBody []byte

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Build a fresh body reader for each attempt.
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
		if err != nil {
			return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("failed to create request: %v", err))
		}

		// Set headers
		req.Header.Set("Authorization", "Bearer "+c.authToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		// Execute request
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("request failed: %v", err))
		}

		// Check if we should retry (429 or 503) and we have attempts left.
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && attempt < c.maxRetries {
			// Determine how long to wait before retrying.
			var waitDuration time.Duration
			if resp.StatusCode == http.StatusTooManyRequests {
				if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
					if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil && seconds > 0 {
						waitDuration = time.Duration(seconds) * time.Second
					} else {
						// Fallback to exponential backoff if header is not a valid integer.
						waitDuration = time.Second * (1 << uint(attempt))
					}
				} else {
					// No Retry-After header; use exponential backoff.
					waitDuration = time.Second * (1 << uint(attempt))
				}
			} else {
				// 503: use exponential backoff (1s, 2s, 4s).
				waitDuration = time.Second * (1 << uint(attempt))
			}

			// Close the body before retrying to free resources.
			resp.Body.Close()

			// Wait, but respect context cancellation.
			select {
			case <-ctx.Done():
				return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("request cancelled during retry: %v", ctx.Err()))
			case <-time.After(waitDuration):
			}
			continue
		}

		// No retry needed; break out of the loop.
		break
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
	}

	// Parse response
	var result interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			// If not JSON, return raw string
			result = string(respBody)
		}
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return &ToolResult{
			Success: false,
			Error: &APIError{
				StatusCode: resp.StatusCode,
				Title:      resp.Status,
				Detail:     extractErrorDetail(result),
			},
			Data: result,
		}
	}

	return SuccessResult(result)
}

// extractErrorDetail attempts to extract error details from the response.
func extractErrorDetail(result interface{}) string {
	if m, ok := result.(map[string]interface{}); ok {
		// Try common error field names
		for _, key := range []string{"error", "message", "detail", "errors"} {
			if v, exists := m[key]; exists {
				switch val := v.(type) {
				case string:
					return val
				case []interface{}:
					if len(val) > 0 {
						if s, ok := val[0].(string); ok {
							return s
						}
						if em, ok := val[0].(map[string]interface{}); ok {
							if detail, ok := em["detail"].(string); ok {
								return detail
							}
							if msg, ok := em["message"].(string); ok {
								return msg
							}
						}
					}
				case map[string]interface{}:
					if detail, ok := val["detail"].(string); ok {
						return detail
					}
					if msg, ok := val["message"].(string); ok {
						return msg
					}
				}
			}
		}
	}
	return ""
}

// addDatasetQueryParam adds the dataset query parameter to a URL.
func (c *Client) addDatasetQueryParam(requestURL string) string {
	if c.dataset == "" {
		return requestURL
	}

	// Parse the URL to handle existing query parameters
	if strings.Contains(requestURL, "?") {
		return requestURL + "&dataset=" + url.QueryEscape(c.dataset)
	}
	return requestURL + "?dataset=" + url.QueryEscape(c.dataset)
}


