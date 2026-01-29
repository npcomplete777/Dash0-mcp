package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ajacobs/dash0-mcp-server/internal/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
		Debug:     true,
	}

	client := New(cfg)

	if client.baseURL != cfg.BaseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, cfg.BaseURL)
	}

	if client.authToken != cfg.AuthToken {
		t.Errorf("authToken = %q, want %q", client.authToken, cfg.AuthToken)
	}

	if client.debug != cfg.Debug {
		t.Errorf("debug = %v, want %v", client.debug, cfg.Debug)
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}

	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("httpClient.Timeout = %v, want %v", client.httpClient.Timeout, 60*time.Second)
	}
}

func TestErrorResult(t *testing.T) {
	result := ErrorResult(404, "not found")

	if result.Success {
		t.Error("Success should be false")
	}

	if result.Error == nil {
		t.Fatal("Error should not be nil")
	}

	if result.Error.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want %d", result.Error.StatusCode, 404)
	}

	if result.Error.Detail != "not found" {
		t.Errorf("Detail = %q, want %q", result.Error.Detail, "not found")
	}
}

func TestSuccessResult(t *testing.T) {
	data := map[string]string{"key": "value"}
	result := SuccessResult(data)

	if !result.Success {
		t.Error("Success should be true")
	}

	if result.Error != nil {
		t.Error("Error should be nil")
	}

	if result.Data == nil {
		t.Fatal("Data should not be nil")
	}
}

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodGet {
			t.Errorf("Method = %s, want %s", r.Method, http.MethodGet)
		}

		if r.URL.Path != "/api/test" {
			t.Errorf("Path = %s, want %s", r.URL.Path, "/api/test")
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
		}

		accept := r.Header.Get("Accept")
		if accept != "application/json" {
			t.Errorf("Accept = %q, want %q", accept, "application/json")
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}

	client := New(cfg)
	result := client.Get(context.Background(), "/api/test")

	if !result.Success {
		t.Errorf("Success = false, want true. Error: %+v", result.Error)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Data is not map[string]interface{}, got %T", result.Data)
	}

	if data["status"] != "ok" {
		t.Errorf("Data[status] = %v, want %v", data["status"], "ok")
	}
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want %s", r.Method, http.MethodPost)
		}

		// Verify body was sent
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode body: %v", err)
		}

		if body["name"] != "test" {
			t.Errorf("body[name] = %v, want %v", body["name"], "test")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}

	client := New(cfg)
	result := client.Post(context.Background(), "/api/test", map[string]string{"name": "test"})

	if !result.Success {
		t.Errorf("Success = false, want true. Error: %+v", result.Error)
	}
}

func TestClientPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Method = %s, want %s", r.Method, http.MethodPut)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"updated": "true"})
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}

	client := New(cfg)
	result := client.Put(context.Background(), "/api/test/123", map[string]string{"name": "updated"})

	if !result.Success {
		t.Errorf("Success = false, want true. Error: %+v", result.Error)
	}
}

func TestClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Method = %s, want %s", r.Method, http.MethodDelete)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}

	client := New(cfg)
	result := client.Delete(context.Background(), "/api/test/123")

	if !result.Success {
		t.Errorf("Success = false, want true. Error: %+v", result.Error)
	}
}

func TestClientErrorResponse(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   interface{}
		expectedDetail string
	}{
		{
			name:       "400 Bad Request with error field",
			statusCode: http.StatusBadRequest,
			responseBody: map[string]string{
				"error": "invalid request body",
			},
			expectedDetail: "invalid request body",
		},
		{
			name:       "404 Not Found with message field",
			statusCode: http.StatusNotFound,
			responseBody: map[string]string{
				"message": "resource not found",
			},
			expectedDetail: "resource not found",
		},
		{
			name:       "500 Internal Server Error with detail field",
			statusCode: http.StatusInternalServerError,
			responseBody: map[string]string{
				"detail": "internal error occurred",
			},
			expectedDetail: "internal error occurred",
		},
		{
			name:       "401 Unauthorized with nested error",
			statusCode: http.StatusUnauthorized,
			responseBody: map[string]interface{}{
				"error": map[string]string{
					"message": "invalid token",
				},
			},
			expectedDetail: "invalid token",
		},
		{
			name:       "422 with errors array",
			statusCode: http.StatusUnprocessableEntity,
			responseBody: map[string]interface{}{
				"errors": []map[string]string{
					{"detail": "field is required"},
				},
			},
			expectedDetail: "field is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			cfg := &config.Config{
				BaseURL:   server.URL,
				AuthToken: "test-token",
			}

			client := New(cfg)
			result := client.Get(context.Background(), "/api/test")

			if result.Success {
				t.Error("Success should be false for error response")
			}

			if result.Error == nil {
				t.Fatal("Error should not be nil")
			}

			if result.Error.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", result.Error.StatusCode, tt.statusCode)
			}

			if result.Error.Detail != tt.expectedDetail {
				t.Errorf("Detail = %q, want %q", result.Error.Detail, tt.expectedDetail)
			}
		})
	}
}

func TestClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}

	client := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := client.Get(ctx, "/api/test")

	if result.Success {
		t.Error("Success should be false when context is cancelled")
	}

	if result.Error == nil {
		t.Error("Error should not be nil when context is cancelled")
	}
}

func TestExtractErrorDetail(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string error field",
			input:    map[string]interface{}{"error": "error message"},
			expected: "error message",
		},
		{
			name:     "string message field",
			input:    map[string]interface{}{"message": "message text"},
			expected: "message text",
		},
		{
			name:     "string detail field",
			input:    map[string]interface{}{"detail": "detail text"},
			expected: "detail text",
		},
		{
			name: "nested error with detail",
			input: map[string]interface{}{
				"error": map[string]interface{}{
					"detail": "nested detail",
				},
			},
			expected: "nested detail",
		},
		{
			name: "nested error with message",
			input: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "nested message",
				},
			},
			expected: "nested message",
		},
		{
			name: "errors array with string",
			input: map[string]interface{}{
				"errors": []interface{}{"first error"},
			},
			expected: "first error",
		},
		{
			name: "errors array with object",
			input: map[string]interface{}{
				"errors": []interface{}{
					map[string]interface{}{"detail": "array error detail"},
				},
			},
			expected: "array error detail",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: "",
		},
		{
			name:     "non-map input",
			input:    "just a string",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractErrorDetail(tt.input)
			if result != tt.expected {
				t.Errorf("extractErrorDetail() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestClientNonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plain text response"))
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}

	client := New(cfg)
	result := client.Get(context.Background(), "/api/test")

	if !result.Success {
		t.Error("Success should be true even for non-JSON response")
	}

	// Non-JSON should be returned as string
	str, ok := result.Data.(string)
	if !ok {
		t.Fatalf("Data should be string for non-JSON, got %T", result.Data)
	}

	if str != "plain text response" {
		t.Errorf("Data = %q, want %q", str, "plain text response")
	}
}

func TestClientEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}

	client := New(cfg)
	result := client.Delete(context.Background(), "/api/test")

	if !result.Success {
		t.Error("Success should be true for 204 No Content")
	}

	if result.Data != nil {
		t.Errorf("Data should be nil for empty response, got %v", result.Data)
	}
}
