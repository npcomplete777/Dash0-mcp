// Package client provides the HTTP client for Dash0 API requests.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ajacobs/dash0-mcp-server/internal/config"
)

// Client handles authenticated HTTP requests to the Dash0 API.
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
	debug      bool
}

// New creates a new Dash0 API client from configuration.
func New(cfg *config.Config) *Client {
	return &Client{
		baseURL:   cfg.BaseURL,
		authToken: cfg.AuthToken,
		debug:     cfg.Debug,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewWithBaseURL creates a new Dash0 API client with a custom base URL.
// This is primarily used for testing with mock servers.
func NewWithBaseURL(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		debug:     false,
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
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return ErrorResult(http.StatusBadRequest, fmt.Sprintf("failed to marshal request body: %v", err))
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("failed to create request: %v", err))
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ErrorResult(http.StatusInternalServerError, fmt.Sprintf("request failed: %v", err))
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
