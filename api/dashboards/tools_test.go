package dashboards

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

	if len(tools) != 5 {
		t.Errorf("Tools() returned %d tools, expected 5", len(tools))
	}

	expectedNames := map[string]bool{
		"dash0_dashboards_list":   false,
		"dash0_dashboards_get":    false,
		"dash0_dashboards_create": false,
		"dash0_dashboards_update": false,
		"dash0_dashboards_delete": false,
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
		"dash0_dashboards_list",
		"dash0_dashboards_get",
		"dash0_dashboards_create",
		"dash0_dashboards_update",
		"dash0_dashboards_delete",
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

func TestListDashboardsToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.ListDashboards()

	if tool.Name != "dash0_dashboards_list" {
		t.Errorf("ListDashboards() name = %s, expected dash0_dashboards_list", tool.Name)
	}

	if tool.Description == "" {
		t.Error("ListDashboards() has empty description")
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("ListDashboards() schema type = %s, expected object", tool.InputSchema.Type)
	}

	// List has no required parameters
	if len(tool.InputSchema.Required) != 0 {
		t.Error("ListDashboards() should have no required parameters")
	}
}

func TestListDashboardsHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/dashboards" {
			t.Errorf("Expected /api/dashboards, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "Dashboard1", "id": "dash-1"},
			{"name": "Dashboard2", "id": "dash-2"},
		})
	}))
	defer server.Close()

	c := client.NewWithBaseURL(server.URL, "test-token")
	pkg := New(c)

	result := pkg.ListDashboardsHandler(context.Background(), map[string]interface{}{})

	if !result.Success {
		t.Errorf("ListDashboardsHandler failed: %v", result.Error)
	}
}

func TestGetDashboardToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.GetDashboard()

	if tool.Name != "dash0_dashboards_get" {
		t.Errorf("GetDashboard() name = %s, expected dash0_dashboards_get", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("GetDashboard() should require 'origin_or_id'")
	}
}

func TestGetDashboardHandler(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectSuccess bool
		expectError   string
		checkPath     string
	}{
		{
			name:        "missing origin_or_id",
			args:        map[string]interface{}{},
			expectError: "origin_or_id is required",
		},
		{
			name: "empty origin_or_id",
			args: map[string]interface{}{
				"origin_or_id": "",
			},
			expectError: "origin_or_id is required",
		},
		{
			name: "valid origin_or_id",
			args: map[string]interface{}{
				"origin_or_id": "my-dashboard",
			},
			expectSuccess: true,
			checkPath:     "/api/dashboards/my-dashboard",
		},
		{
			name: "origin_or_id with special characters",
			args: map[string]interface{}{
				"origin_or_id": "dash/with spaces",
			},
			expectSuccess: true,
			checkPath:     "/api/dashboards/dash%2Fwith%20spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				json.NewEncoder(w).Encode(map[string]interface{}{
					"kind": "PersesDashboard",
					"metadata": map[string]interface{}{
						"name": "test-dashboard",
					},
				})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.GetDashboardHandler(context.Background(), tt.args)

			if tt.expectError != "" {
				if result.Success {
					t.Error("Expected error, got success")
				}
				return
			}

			if tt.expectSuccess && !result.Success {
				t.Errorf("Expected success, got failure: %v", result.Error)
			}

			if tt.checkPath != "" && receivedPath != tt.checkPath {
				t.Errorf("Path = %s, expected %s", receivedPath, tt.checkPath)
			}
		})
	}
}

func TestCreateDashboardToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateDashboard()

	if tool.Name != "dash0_dashboards_create" {
		t.Errorf("CreateDashboard() name = %s, expected dash0_dashboards_create", tool.Name)
	}

	// Description should mention Perses format
	if !strings.Contains(tool.Description, "PersesDashboard") {
		t.Error("CreateDashboard() description should mention 'PersesDashboard'")
	}

	// Should require body
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("CreateDashboard() should require 'body'")
	}

	// Body should have properties for kind, metadata, spec
	bodyProps, ok := tool.InputSchema.Properties["body"].(map[string]interface{})
	if !ok {
		t.Fatal("body property not found in schema")
	}

	props, ok := bodyProps["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("body.properties not found in schema")
	}

	expectedProps := []string{"kind", "metadata", "spec"}
	for _, prop := range expectedProps {
		if _, exists := props[prop]; !exists {
			t.Errorf("CreateDashboard() body missing property: %s", prop)
		}
	}
}

func TestCreateDashboardHandler(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectSuccess bool
		expectError   string
	}{
		{
			name:        "missing body",
			args:        map[string]interface{}{},
			expectError: "body is required",
		},
		{
			name: "valid body",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"kind": "PersesDashboard",
					"metadata": map[string]interface{}{
						"name": "my-dashboard",
					},
					"spec": map[string]interface{}{
						"display": map[string]interface{}{
							"name": "My Dashboard",
						},
						"panels": []interface{}{},
					},
				},
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedBody map[string]interface{}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/dashboards" {
					t.Errorf("Expected /api/dashboards, got %s", r.URL.Path)
				}
				json.NewDecoder(r.Body).Decode(&receivedBody)
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "new-dashboard"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.CreateDashboardHandler(context.Background(), tt.args)

			if tt.expectError != "" {
				if result.Success {
					t.Error("Expected error, got success")
				}
				return
			}

			if tt.expectSuccess && !result.Success {
				t.Errorf("Expected success, got failure: %v", result.Error)
			}
		})
	}
}

func TestUpdateDashboardToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.UpdateDashboard()

	if tool.Name != "dash0_dashboards_update" {
		t.Errorf("UpdateDashboard() name = %s, expected dash0_dashboards_update", tool.Name)
	}

	// Should require origin_or_id and body
	if len(tool.InputSchema.Required) != 2 {
		t.Error("UpdateDashboard() should require 2 parameters")
	}

	required := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}
	if !required["origin_or_id"] || !required["body"] {
		t.Error("UpdateDashboard() should require origin_or_id and body")
	}
}

func TestUpdateDashboardHandler(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectSuccess bool
		expectError   string
		checkPath     string
	}{
		{
			name:        "missing origin_or_id",
			args:        map[string]interface{}{"body": map[string]interface{}{}},
			expectError: "origin_or_id is required",
		},
		{
			name:        "missing body",
			args:        map[string]interface{}{"origin_or_id": "dash-123"},
			expectError: "body is required",
		},
		{
			name: "valid update",
			args: map[string]interface{}{
				"origin_or_id": "my-dashboard",
				"body": map[string]interface{}{
					"kind": "PersesDashboard",
					"metadata": map[string]interface{}{
						"name": "updated-dashboard",
					},
					"spec": map[string]interface{}{
						"display": map[string]interface{}{
							"name": "Updated Dashboard",
						},
						"panels": []interface{}{},
					},
				},
			},
			expectSuccess: true,
			checkPath:     "/api/dashboards/my-dashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "my-dashboard"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.UpdateDashboardHandler(context.Background(), tt.args)

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
				if receivedMethod != http.MethodPut {
					t.Errorf("Expected PUT, got %s", receivedMethod)
				}
				if tt.checkPath != "" && receivedPath != tt.checkPath {
					t.Errorf("Path = %s, expected %s", receivedPath, tt.checkPath)
				}
			}
		})
	}
}

func TestDeleteDashboardToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.DeleteDashboard()

	if tool.Name != "dash0_dashboards_delete" {
		t.Errorf("DeleteDashboard() name = %s, expected dash0_dashboards_delete", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("DeleteDashboard() should require 'origin_or_id'")
	}
}

func TestDeleteDashboardHandler(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectSuccess bool
		expectError   string
		checkPath     string
	}{
		{
			name:        "missing origin_or_id",
			args:        map[string]interface{}{},
			expectError: "origin_or_id is required",
		},
		{
			name: "valid delete",
			args: map[string]interface{}{
				"origin_or_id": "dashboard-to-delete",
			},
			expectSuccess: true,
			checkPath:     "/api/dashboards/dashboard-to-delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				receivedMethod = r.Method
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.DeleteDashboardHandler(context.Background(), tt.args)

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
				if receivedMethod != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", receivedMethod)
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
		// All dashboard tools should start with dash0_dashboards_
		if !strings.HasPrefix(tool.Name, "dash0_dashboards_") {
			t.Errorf("Tool %s does not follow naming convention dash0_dashboards_*", tool.Name)
		}

		// Should use underscores, not hyphens
		if strings.Contains(tool.Name, "-") {
			t.Errorf("Tool %s should use underscores, not hyphens", tool.Name)
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

func TestCreateDashboardDescription_ContainsExamples(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateDashboard()

	// Description should contain JSON examples
	if !strings.Contains(tool.Description, "Example body") {
		t.Error("CreateDashboard() description should contain example body")
	}

	// Should mention panels
	if !strings.Contains(tool.Description, "panels") {
		t.Error("CreateDashboard() description should mention panels")
	}
}
