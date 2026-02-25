package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajacobs/dash0-mcp-server/internal/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
		Debug:     true,
	}

	client := New(cfg)

	if client == nil {
		t.Fatal("New() returned nil")
	}
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
}

func TestNewWithBaseURL(t *testing.T) {
	baseURL := "https://test.api.com"
	authToken := "test-token-123"

	client := NewWithBaseURL(baseURL, authToken)

	if client == nil {
		t.Fatal("NewWithBaseURL() returned nil")
	}
	if client.baseURL != baseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, baseURL)
	}
	if client.authToken != authToken {
		t.Errorf("authToken = %q, want %q", client.authToken, authToken)
	}
	if client.debug != false {
		t.Errorf("debug = %v, want false", client.debug)
	}
	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestClient_Get(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   interface{}
		wantSuccess    bool
		wantStatusCode int
	}{
		{
			name:         "successful GET",
			statusCode:   http.StatusOK,
			responseBody: map[string]interface{}{"data": "test"},
			wantSuccess:  true,
		},
		{
			name:           "404 error",
			statusCode:     http.StatusNotFound,
			responseBody:   map[string]interface{}{"error": "not found"},
			wantSuccess:    false,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "500 error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   map[string]interface{}{"error": "internal server error"},
			wantSuccess:    false,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("expected Bearer token, got %s", r.Header.Get("Authorization"))
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
				}

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewWithBaseURL(server.URL, "test-token")
			result := client.Get(context.Background(), "/test")

			if result.Success != tt.wantSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.wantSuccess)
			}
			if !tt.wantSuccess && result.Error != nil && result.Error.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %d, want %d", result.Error.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestClient_Post(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		statusCode     int
		responseBody   interface{}
		wantSuccess    bool
	}{
		{
			name:         "successful POST with body",
			body:         map[string]interface{}{"key": "value"},
			statusCode:   http.StatusOK,
			responseBody: map[string]interface{}{"result": "success"},
			wantSuccess:  true,
		},
		{
			name:         "successful POST with nil body",
			body:         nil,
			statusCode:   http.StatusOK,
			responseBody: map[string]interface{}{"result": "success"},
			wantSuccess:  true,
		},
		{
			name:         "POST with 400 error",
			body:         map[string]interface{}{"invalid": "data"},
			statusCode:   http.StatusBadRequest,
			responseBody: map[string]interface{}{"error": "bad request"},
			wantSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewWithBaseURL(server.URL, "test-token")
			result := client.Post(context.Background(), "/test", tt.body)

			if result.Success != tt.wantSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.wantSuccess)
			}
		})
	}
}

func TestClient_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"updated": true})
	}))
	defer server.Close()

	client := NewWithBaseURL(server.URL, "test-token")
	result := client.Put(context.Background(), "/test", map[string]interface{}{"data": "update"})

	if !result.Success {
		t.Errorf("expected success, got failure")
	}
}

func TestClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewWithBaseURL(server.URL, "test-token")
	result := client.Delete(context.Background(), "/test")

	if !result.Success {
		t.Errorf("expected success, got failure")
	}
}

func TestErrorResult(t *testing.T) {
	result := ErrorResult(404, "not found")

	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.Error == nil {
		t.Fatal("expected Error to be set")
	}
	if result.Error.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", result.Error.StatusCode)
	}
	if result.Error.Detail != "not found" {
		t.Errorf("Detail = %q, want %q", result.Error.Detail, "not found")
	}
}

func TestSuccessResult(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	result := SuccessResult(data)

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Error != nil {
		t.Error("expected Error to be nil")
	}
	if result.Data == nil {
		t.Error("expected Data to be set")
	}
}

func TestExtractErrorDetail(t *testing.T) {
	tests := []struct {
		name   string
		result interface{}
		want   string
	}{
		{
			name:   "error field",
			result: map[string]interface{}{"error": "test error"},
			want:   "test error",
		},
		{
			name:   "message field",
			result: map[string]interface{}{"message": "test message"},
			want:   "test message",
		},
		{
			name:   "detail field",
			result: map[string]interface{}{"detail": "test detail"},
			want:   "test detail",
		},
		{
			name:   "errors array with string",
			result: map[string]interface{}{"errors": []interface{}{"first error"}},
			want:   "first error",
		},
		{
			name:   "errors array with detail map",
			result: map[string]interface{}{"errors": []interface{}{map[string]interface{}{"detail": "nested detail"}}},
			want:   "nested detail",
		},
		{
			name:   "errors array with message map",
			result: map[string]interface{}{"errors": []interface{}{map[string]interface{}{"message": "nested message"}}},
			want:   "nested message",
		},
		{
			name:   "nested error detail",
			result: map[string]interface{}{"error": map[string]interface{}{"detail": "nested error detail"}},
			want:   "nested error detail",
		},
		{
			name:   "nested error message",
			result: map[string]interface{}{"error": map[string]interface{}{"message": "nested error message"}},
			want:   "nested error message",
		},
		{
			name:   "non-map result",
			result: "string result",
			want:   "",
		},
		{
			name:   "nil result",
			result: nil,
			want:   "",
		},
		{
			name:   "empty map",
			result: map[string]interface{}{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractErrorDetail(tt.result)
			if got != tt.want {
				t.Errorf("extractErrorDetail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClient_Request_NonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plain text response"))
	}))
	defer server.Close()

	client := NewWithBaseURL(server.URL, "test-token")
	result := client.Get(context.Background(), "/test")

	if !result.Success {
		t.Error("expected success")
	}
	if result.Data != "plain text response" {
		t.Errorf("Data = %v, want %q", result.Data, "plain text response")
	}
}

func TestClient_Request_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewWithBaseURL(server.URL, "test-token")
	result := client.Get(context.Background(), "/test")

	if !result.Success {
		t.Error("expected success")
	}
}

func TestClient_Request_MarshalError(t *testing.T) {
	client := NewWithBaseURL("http://example.com", "test-token")

	// Create a value that cannot be marshaled to JSON
	badBody := make(chan int)

	result := client.Post(context.Background(), "/test", badBody)

	if result.Success {
		t.Error("expected failure due to marshal error")
	}
	if result.Error == nil {
		t.Fatal("expected Error to be set")
	}
	if result.Error.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", result.Error.StatusCode, http.StatusBadRequest)
	}
}

func TestClient_Request_NetworkError(t *testing.T) {
	// Use a URL that will fail to connect
	client := NewWithBaseURL("http://localhost:1", "test-token")

	result := client.Get(context.Background(), "/test")

	if result.Success {
		t.Error("expected failure due to network error")
	}
	if result.Error == nil {
		t.Fatal("expected Error to be set")
	}
	if result.Error.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want %d", result.Error.StatusCode, http.StatusInternalServerError)
	}
}

func TestClient_Request_PathConcatenation(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWithBaseURL(server.URL, "test-token")
	client.Get(context.Background(), "/api/v1/test")

	if capturedPath != "/api/v1/test" {
		t.Errorf("path = %q, want %q", capturedPath, "/api/v1/test")
	}
}

func TestClient_Request_AuthorizationHeader(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWithBaseURL(server.URL, "my-secret-token")
	client.Get(context.Background(), "/test")

	if capturedAuth != "Bearer my-secret-token" {
		t.Errorf("Authorization = %q, want %q", capturedAuth, "Bearer my-secret-token")
	}
}

func TestClient_DatasetQueryParam(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
		Dataset:   "my-dataset",
	}
	client := New(cfg)
	client.Get(context.Background(), "/api/views")

	if capturedURL != "/api/views?dataset=my-dataset" {
		t.Errorf("URL = %q, want %q", capturedURL, "/api/views?dataset=my-dataset")
	}
}

func TestClient_DatasetQueryParamWithExistingParams(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
		Dataset:   "my-dataset",
	}
	client := New(cfg)
	client.Get(context.Background(), "/api/views?other=param")

	if capturedURL != "/api/views?other=param&dataset=my-dataset" {
		t.Errorf("URL = %q, want %q", capturedURL, "/api/views?other=param&dataset=my-dataset")
	}
}

func TestClient_DatasetInPostBody(t *testing.T) {
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
		Dataset:   "my-dataset",
	}
	client := New(cfg)
	client.Post(context.Background(), "/api/spans", map[string]interface{}{
		"sampling": map[string]string{"mode": "adaptive"},
	})

	if capturedBody["dataset"] != "my-dataset" {
		t.Errorf("dataset = %v, want %q", capturedBody["dataset"], "my-dataset")
	}
	if capturedBody["sampling"] == nil {
		t.Error("expected sampling to be preserved in body")
	}
}

func TestClient_DatasetDoesNotOverrideExisting(t *testing.T) {
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
		Dataset:   "config-dataset",
	}
	client := New(cfg)
	client.Post(context.Background(), "/api/spans", map[string]interface{}{
		"dataset": "explicit-dataset",
	})

	if capturedBody["dataset"] != "explicit-dataset" {
		t.Errorf("dataset = %v, want %q (should not override)", capturedBody["dataset"], "explicit-dataset")
	}
}

func TestClient_NoDatasetWhenNotConfigured(t *testing.T) {
	var capturedURL string
	var capturedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		if r.Method == http.MethodPost {
			json.NewDecoder(r.Body).Decode(&capturedBody)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
		// No Dataset configured
	}
	client := New(cfg)

	// Test GET - should not have dataset param
	client.Get(context.Background(), "/api/views")
	if capturedURL != "/api/views" {
		t.Errorf("URL = %q, want %q (no dataset param)", capturedURL, "/api/views")
	}

	// Test POST - should not have dataset in body
	client.Post(context.Background(), "/api/spans", map[string]interface{}{"data": "test"})
	if _, exists := capturedBody["dataset"]; exists {
		t.Errorf("dataset should not be in body when not configured")
	}
}

func TestClient_DatasetDeleteQueryParam(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
		Dataset:   "my-dataset",
	}
	client := New(cfg)
	client.Delete(context.Background(), "/api/views/123")

	if capturedURL != "/api/views/123?dataset=my-dataset" {
		t.Errorf("URL = %q, want %q", capturedURL, "/api/views/123?dataset=my-dataset")
	}
}
