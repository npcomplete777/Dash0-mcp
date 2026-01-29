package imports

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
)

func TestNew(t *testing.T) {
	c := &client.Client{}
	pkg := New(c)
	if pkg == nil {
		t.Fatal("New() returned nil")
	}
	if pkg.client != c {
		t.Error("New() did not set client correctly")
	}
}

func TestTools(t *testing.T) {
	pkg := New(&client.Client{})
	tools := pkg.Tools()

	if len(tools) != 4 {
		t.Errorf("Tools() returned %d tools, expected 4", len(tools))
	}

	expectedNames := map[string]bool{
		"dash0_import_check_rule":     false,
		"dash0_import_dashboard":      false,
		"dash0_import_synthetic_check": false,
		"dash0_import_view":           false,
	}

	for _, tool := range tools {
		if _, exists := expectedNames[tool.Name]; !exists {
			t.Errorf("Unexpected tool name: %s", tool.Name)
		}
		expectedNames[tool.Name] = true
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Missing expected tool: %s", name)
		}
	}
}

func TestHandlers(t *testing.T) {
	pkg := New(&client.Client{})
	handlers := pkg.Handlers()

	expectedHandlers := []string{
		"dash0_import_check_rule",
		"dash0_import_dashboard",
		"dash0_import_synthetic_check",
		"dash0_import_view",
	}

	if len(handlers) != len(expectedHandlers) {
		t.Errorf("Handlers() returned %d handlers, expected %d", len(handlers), len(expectedHandlers))
	}

	for _, name := range expectedHandlers {
		if _, exists := handlers[name]; !exists {
			t.Errorf("Missing handler for: %s", name)
		}
	}
}

func TestImportCheckRuleToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.ImportCheckRule()

	if tool.Name != "dash0_import_check_rule" {
		t.Errorf("ImportCheckRule() name = %s, expected dash0_import_check_rule", tool.Name)
	}

	if tool.Description == "" {
		t.Error("ImportCheckRule() has empty description")
	}

	// Should mention Prometheus
	if !strings.Contains(tool.Description, "Prometheus") {
		t.Error("ImportCheckRule() description should mention Prometheus")
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("ImportCheckRule() schema type = %s, expected object", tool.InputSchema.Type)
	}

	// Should require body
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("ImportCheckRule() should require 'body'")
	}
}

func TestImportCheckRuleHandler(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectSuccess bool
		expectError   string
		checkPath     string
	}{
		{
			name:        "missing body",
			args:        map[string]interface{}{},
			expectError: "body is required",
		},
		{
			name: "valid prometheus alert rule",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"alert": "HighErrorRate",
					"expr":  "rate(http_errors_total[5m]) > 0.05",
					"for":   "5m",
					"labels": map[string]interface{}{
						"severity": "critical",
					},
				},
			},
			expectSuccess: true,
			checkPath:     "/api/import/check-rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "imported-rule"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.ImportCheckRuleHandler(context.Background(), tt.args)

			if tt.expectError != "" {
				if result.Success {
					t.Error("Expected error, got success")
				}
				return
			}

			if tt.expectSuccess {
				if !result.Success {
					t.Errorf("Expected success, got failure: %v", result.Error)
				}
				if receivedMethod != http.MethodPost {
					t.Errorf("Expected POST, got %s", receivedMethod)
				}
				if tt.checkPath != "" && receivedPath != tt.checkPath {
					t.Errorf("Path = %s, expected %s", receivedPath, tt.checkPath)
				}
			}
		})
	}
}

func TestImportDashboardToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.ImportDashboard()

	if tool.Name != "dash0_import_dashboard" {
		t.Errorf("ImportDashboard() name = %s, expected dash0_import_dashboard", tool.Name)
	}

	if tool.Description == "" {
		t.Error("ImportDashboard() has empty description")
	}

	// Should mention Grafana
	if !strings.Contains(tool.Description, "Grafana") {
		t.Error("ImportDashboard() description should mention Grafana")
	}

	// Should require body
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("ImportDashboard() should require 'body'")
	}
}

func TestImportDashboardHandler(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectSuccess bool
		expectError   string
		checkPath     string
	}{
		{
			name:        "missing body",
			args:        map[string]interface{}{},
			expectError: "body is required",
		},
		{
			name: "valid grafana dashboard",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"title": "My Grafana Dashboard",
					"panels": []interface{}{
						map[string]interface{}{
							"title": "Panel 1",
							"type":  "graph",
						},
					},
				},
			},
			expectSuccess: true,
			checkPath:     "/api/import/dashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "imported-dashboard"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.ImportDashboardHandler(context.Background(), tt.args)

			if tt.expectError != "" {
				if result.Success {
					t.Error("Expected error, got success")
				}
				return
			}

			if tt.expectSuccess {
				if !result.Success {
					t.Errorf("Expected success, got failure: %v", result.Error)
				}
				if receivedMethod != http.MethodPost {
					t.Errorf("Expected POST, got %s", receivedMethod)
				}
				if tt.checkPath != "" && receivedPath != tt.checkPath {
					t.Errorf("Path = %s, expected %s", receivedPath, tt.checkPath)
				}
			}
		})
	}
}

func TestImportSyntheticCheckToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.ImportSyntheticCheck()

	if tool.Name != "dash0_import_synthetic_check" {
		t.Errorf("ImportSyntheticCheck() name = %s, expected dash0_import_synthetic_check", tool.Name)
	}

	if tool.Description == "" {
		t.Error("ImportSyntheticCheck() has empty description")
	}

	// Should require body
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("ImportSyntheticCheck() should require 'body'")
	}
}

func TestImportSyntheticCheckHandler(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectSuccess bool
		expectError   string
		checkPath     string
	}{
		{
			name:        "missing body",
			args:        map[string]interface{}{},
			expectError: "body is required",
		},
		{
			name: "valid synthetic check",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"name": "API Health Check",
					"url":  "https://api.example.com/health",
					"type": "http",
				},
			},
			expectSuccess: true,
			checkPath:     "/api/import/synthetic-check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "imported-check"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.ImportSyntheticCheckHandler(context.Background(), tt.args)

			if tt.expectError != "" {
				if result.Success {
					t.Error("Expected error, got success")
				}
				return
			}

			if tt.expectSuccess {
				if !result.Success {
					t.Errorf("Expected success, got failure: %v", result.Error)
				}
				if receivedMethod != http.MethodPost {
					t.Errorf("Expected POST, got %s", receivedMethod)
				}
				if tt.checkPath != "" && receivedPath != tt.checkPath {
					t.Errorf("Path = %s, expected %s", receivedPath, tt.checkPath)
				}
			}
		})
	}
}

func TestImportViewToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.ImportView()

	if tool.Name != "dash0_import_view" {
		t.Errorf("ImportView() name = %s, expected dash0_import_view", tool.Name)
	}

	if tool.Description == "" {
		t.Error("ImportView() has empty description")
	}

	// Should require body
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("ImportView() should require 'body'")
	}
}

func TestImportViewHandler(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectSuccess bool
		expectError   string
		checkPath     string
	}{
		{
			name:        "missing body",
			args:        map[string]interface{}{},
			expectError: "body is required",
		},
		{
			name: "valid view",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"name":   "Production Errors",
					"query":  "level:error",
					"filter": "service:production",
				},
			},
			expectSuccess: true,
			checkPath:     "/api/import/view",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "imported-view"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.ImportViewHandler(context.Background(), tt.args)

			if tt.expectError != "" {
				if result.Success {
					t.Error("Expected error, got success")
				}
				return
			}

			if tt.expectSuccess {
				if !result.Success {
					t.Errorf("Expected success, got failure: %v", result.Error)
				}
				if receivedMethod != http.MethodPost {
					t.Errorf("Expected POST, got %s", receivedMethod)
				}
				if tt.checkPath != "" && receivedPath != tt.checkPath {
					t.Errorf("Path = %s, expected %s", receivedPath, tt.checkPath)
				}
			}
		})
	}
}

func TestToolNamingConvention(t *testing.T) {
	pkg := New(&client.Client{})
	tools := pkg.Tools()

	for _, tool := range tools {
		// All import tools should start with dash0_import_
		if !strings.HasPrefix(tool.Name, "dash0_import_") {
			t.Errorf("Tool %s does not follow naming convention dash0_import_*", tool.Name)
		}

		// Should use underscores, not hyphens
		parts := strings.Split(tool.Name, "_")
		for _, part := range parts {
			if strings.Contains(part, "-") {
				t.Errorf("Tool %s should use underscores, not hyphens within parts", tool.Name)
			}
		}
	}
}

func TestToolDescriptionsNotEmpty(t *testing.T) {
	pkg := New(&client.Client{})
	tools := pkg.Tools()

	for _, tool := range tools {
		if tool.Description == "" {
			t.Errorf("Tool %s has empty description", tool.Name)
		}
	}
}

func TestAllImportToolsUsePost(t *testing.T) {
	// Verify that all import handlers send POST requests to the correct endpoints
	testCases := []struct {
		toolName     string
		expectedPath string
		handler      func(*Package, context.Context, map[string]interface{}) *client.ToolResult
	}{
		{
			toolName:     "ImportCheckRule",
			expectedPath: "/api/import/check-rule",
			handler: func(p *Package, ctx context.Context, args map[string]interface{}) *client.ToolResult {
				return p.ImportCheckRuleHandler(ctx, args)
			},
		},
		{
			toolName:     "ImportDashboard",
			expectedPath: "/api/import/dashboard",
			handler: func(p *Package, ctx context.Context, args map[string]interface{}) *client.ToolResult {
				return p.ImportDashboardHandler(ctx, args)
			},
		},
		{
			toolName:     "ImportSyntheticCheck",
			expectedPath: "/api/import/synthetic-check",
			handler: func(p *Package, ctx context.Context, args map[string]interface{}) *client.ToolResult {
				return p.ImportSyntheticCheckHandler(ctx, args)
			},
		},
		{
			toolName:     "ImportView",
			expectedPath: "/api/import/view",
			handler: func(p *Package, ctx context.Context, args map[string]interface{}) *client.ToolResult {
				return p.ImportViewHandler(ctx, args)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.toolName, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			args := map[string]interface{}{
				"body": map[string]interface{}{
					"test": "data",
				},
			}

			result := tc.handler(pkg, context.Background(), args)

			if !result.Success {
				t.Errorf("%s failed: %v", tc.toolName, result.Error)
			}

			if receivedMethod != http.MethodPost {
				t.Errorf("%s expected POST, got %s", tc.toolName, receivedMethod)
			}

			if receivedPath != tc.expectedPath {
				t.Errorf("%s path = %s, expected %s", tc.toolName, receivedPath, tc.expectedPath)
			}
		})
	}
}

func TestImportToolsOnlySupportPost(t *testing.T) {
	// Import tools should only support POST (create), not GET/PUT/DELETE
	// This test verifies the design choice that imports are one-way operations
	pkg := New(&client.Client{})
	tools := pkg.Tools()

	// Should have exactly 4 import tools
	if len(tools) != 4 {
		t.Errorf("Expected 4 import tools, got %d", len(tools))
	}

	// All tools should have the same structure (body required)
	for _, tool := range tools {
		if len(tool.InputSchema.Required) != 1 {
			t.Errorf("Tool %s should have exactly 1 required field", tool.Name)
		}
		if tool.InputSchema.Required[0] != "body" {
			t.Errorf("Tool %s should require 'body', got %s", tool.Name, tool.InputSchema.Required[0])
		}
	}
}
