package views

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
		"dash0_views_list":   false,
		"dash0_views_get":    false,
		"dash0_views_create": false,
		"dash0_views_update": false,
		"dash0_views_delete": false,
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
		"dash0_views_list",
		"dash0_views_get",
		"dash0_views_create",
		"dash0_views_update",
		"dash0_views_delete",
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

func TestListViewsToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.ListViews()

	if tool.Name != "dash0_views_list" {
		t.Errorf("ListViews() name = %s, expected dash0_views_list", tool.Name)
	}

	if tool.Description == "" {
		t.Error("ListViews() has empty description")
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("ListViews() schema type = %s, expected object", tool.InputSchema.Type)
	}

	// List has no required parameters
	if len(tool.InputSchema.Required) != 0 {
		t.Error("ListViews() should have no required parameters")
	}
}

func TestListViewsHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/views" {
			t.Errorf("Expected /api/views, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "production-errors", "id": "view-1"},
			{"name": "staging-logs", "id": "view-2"},
		})
	}))
	defer server.Close()

	c := client.NewWithBaseURL(server.URL, "test-token")
	pkg := New(c)

	result := pkg.ListViewsHandler(context.Background(), map[string]interface{}{})

	if !result.Success {
		t.Errorf("ListViewsHandler failed: %v", result.Error)
	}
}

func TestGetViewToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.GetView()

	if tool.Name != "dash0_views_get" {
		t.Errorf("GetView() name = %s, expected dash0_views_get", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("GetView() should require 'origin_or_id'")
	}
}

func TestGetViewHandler(t *testing.T) {
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
				"origin_or_id": "my-view",
			},
			expectSuccess: true,
			checkPath:     "/api/views/my-view",
		},
		{
			name: "origin_or_id with special characters",
			args: map[string]interface{}{
				"origin_or_id": "view/with spaces",
			},
			expectSuccess: true,
			checkPath:     "/api/views/view%2Fwith%20spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				json.NewEncoder(w).Encode(map[string]interface{}{
					"kind": "Dash0View",
					"metadata": map[string]interface{}{
						"name": "test-view",
					},
				})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.GetViewHandler(context.Background(), tt.args)

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

func TestCreateViewToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateView()

	if tool.Name != "dash0_views_create" {
		t.Errorf("CreateView() name = %s, expected dash0_views_create", tool.Name)
	}

	// Description should mention Dash0View format
	if !strings.Contains(tool.Description, "Dash0View") {
		t.Error("CreateView() description should mention 'Dash0View'")
	}

	// Should require body
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("CreateView() should require 'body'")
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
			t.Errorf("CreateView() body missing property: %s", prop)
		}
	}
}

func TestCreateViewHandler(t *testing.T) {
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
					"kind": "Dash0View",
					"metadata": map[string]interface{}{
						"name": "my-view",
					},
					"spec": map[string]interface{}{
						"type": "resources",
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
				if r.URL.Path != "/api/views" {
					t.Errorf("Expected /api/views, got %s", r.URL.Path)
				}
				json.NewDecoder(r.Body).Decode(&receivedBody)
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "new-view"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.CreateViewHandler(context.Background(), tt.args)

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

func TestUpdateViewToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.UpdateView()

	if tool.Name != "dash0_views_update" {
		t.Errorf("UpdateView() name = %s, expected dash0_views_update", tool.Name)
	}

	// Should require origin_or_id and body
	if len(tool.InputSchema.Required) != 2 {
		t.Error("UpdateView() should require 2 parameters")
	}

	required := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}
	if !required["origin_or_id"] || !required["body"] {
		t.Error("UpdateView() should require origin_or_id and body")
	}
}

func TestUpdateViewHandler(t *testing.T) {
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
			args:        map[string]interface{}{"origin_or_id": "view-123"},
			expectError: "body is required",
		},
		{
			name: "valid update",
			args: map[string]interface{}{
				"origin_or_id": "my-view",
				"body": map[string]interface{}{
					"kind": "Dash0View",
					"metadata": map[string]interface{}{
						"name": "updated-view",
					},
					"spec": map[string]interface{}{
						"type": "resources",
					},
				},
			},
			expectSuccess: true,
			checkPath:     "/api/views/my-view",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "my-view"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.UpdateViewHandler(context.Background(), tt.args)

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

func TestDeleteViewToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.DeleteView()

	if tool.Name != "dash0_views_delete" {
		t.Errorf("DeleteView() name = %s, expected dash0_views_delete", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("DeleteView() should require 'origin_or_id'")
	}
}

func TestDeleteViewHandler(t *testing.T) {
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
				"origin_or_id": "view-to-delete",
			},
			expectSuccess: true,
			checkPath:     "/api/views/view-to-delete",
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

			result := pkg.DeleteViewHandler(context.Background(), tt.args)

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
		// All view tools should start with dash0_views_
		if !strings.HasPrefix(tool.Name, "dash0_views_") {
			t.Errorf("Tool %s does not follow naming convention dash0_views_*", tool.Name)
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

func TestViewSpecType(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateView()

	// The description should mention "resources" as the supported type
	if !strings.Contains(tool.Description, "resources") {
		t.Error("CreateView() description should mention 'resources' type")
	}
}
