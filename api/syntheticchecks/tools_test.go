package syntheticchecks

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
		"dash0_synthetic_checks_list":   false,
		"dash0_synthetic_checks_get":    false,
		"dash0_synthetic_checks_create": false,
		"dash0_synthetic_checks_update": false,
		"dash0_synthetic_checks_delete": false,
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
		"dash0_synthetic_checks_list",
		"dash0_synthetic_checks_get",
		"dash0_synthetic_checks_create",
		"dash0_synthetic_checks_update",
		"dash0_synthetic_checks_delete",
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

func TestListSyntheticChecksToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.ListSyntheticChecks()

	if tool.Name != "dash0_synthetic_checks_list" {
		t.Errorf("ListSyntheticChecks() name = %s, expected dash0_synthetic_checks_list", tool.Name)
	}

	if tool.Description == "" {
		t.Error("ListSyntheticChecks() has empty description")
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("ListSyntheticChecks() schema type = %s, expected object", tool.InputSchema.Type)
	}

	// List has no required parameters
	if len(tool.InputSchema.Required) != 0 {
		t.Error("ListSyntheticChecks() should have no required parameters")
	}
}

func TestListSyntheticChecksHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/synthetic-checks" {
			t.Errorf("Expected /api/synthetic-checks, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "api-health-check", "id": "check-1"},
			{"name": "website-uptime", "id": "check-2"},
		})
	}))
	defer server.Close()

	c := client.NewWithBaseURL(server.URL, "test-token")
	pkg := New(c)

	result := pkg.ListSyntheticChecksHandler(context.Background(), map[string]interface{}{})

	if !result.Success {
		t.Errorf("ListSyntheticChecksHandler failed: %v", result.Error)
	}
}

func TestGetSyntheticCheckToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.GetSyntheticCheck()

	if tool.Name != "dash0_synthetic_checks_get" {
		t.Errorf("GetSyntheticCheck() name = %s, expected dash0_synthetic_checks_get", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("GetSyntheticCheck() should require 'origin_or_id'")
	}
}

func TestGetSyntheticCheckHandler(t *testing.T) {
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
				"origin_or_id": "my-check",
			},
			expectSuccess: true,
			checkPath:     "/api/synthetic-checks/my-check",
		},
		{
			name: "origin_or_id with special characters",
			args: map[string]interface{}{
				"origin_or_id": "check/with spaces",
			},
			expectSuccess: true,
			checkPath:     "/api/synthetic-checks/check%2Fwith%20spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				json.NewEncoder(w).Encode(map[string]interface{}{
					"kind": "Dash0SyntheticCheck",
					"metadata": map[string]interface{}{
						"name": "test-check",
					},
				})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.GetSyntheticCheckHandler(context.Background(), tt.args)

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

func TestCreateSyntheticCheckToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateSyntheticCheck()

	if tool.Name != "dash0_synthetic_checks_create" {
		t.Errorf("CreateSyntheticCheck() name = %s, expected dash0_synthetic_checks_create", tool.Name)
	}

	// Description should mention Dash0SyntheticCheck format
	if !strings.Contains(tool.Description, "Dash0SyntheticCheck") {
		t.Error("CreateSyntheticCheck() description should mention 'Dash0SyntheticCheck'")
	}

	// Description should mention nested plugin structure
	if !strings.Contains(tool.Description, "plugin.spec.request") || !strings.Contains(tool.Description, "NESTED") {
		t.Error("CreateSyntheticCheck() description should emphasize nested plugin structure")
	}

	// Should require body
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("CreateSyntheticCheck() should require 'body'")
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
			t.Errorf("CreateSyntheticCheck() body missing property: %s", prop)
		}
	}
}

func TestCreateSyntheticCheckHandler(t *testing.T) {
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
			name: "valid body with HTTP check",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"kind": "Dash0SyntheticCheck",
					"metadata": map[string]interface{}{
						"name": "api-health-check",
					},
					"spec": map[string]interface{}{
						"enabled": true,
						"plugin": map[string]interface{}{
							"kind": "http",
							"spec": map[string]interface{}{
								"request": map[string]interface{}{
									"method":    "get",
									"url":       "https://api.example.com/health",
									"redirects": "follow",
								},
							},
						},
						"schedule": map[string]interface{}{
							"interval":  "5m",
							"locations": []interface{}{"eu-west-1"},
							"strategy":  "all_locations",
						},
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
				if r.URL.Path != "/api/synthetic-checks" {
					t.Errorf("Expected /api/synthetic-checks, got %s", r.URL.Path)
				}
				json.NewDecoder(r.Body).Decode(&receivedBody)
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "new-check"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.CreateSyntheticCheckHandler(context.Background(), tt.args)

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

func TestUpdateSyntheticCheckToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.UpdateSyntheticCheck()

	if tool.Name != "dash0_synthetic_checks_update" {
		t.Errorf("UpdateSyntheticCheck() name = %s, expected dash0_synthetic_checks_update", tool.Name)
	}

	// Should require origin_or_id and body
	if len(tool.InputSchema.Required) != 2 {
		t.Error("UpdateSyntheticCheck() should require 2 parameters")
	}

	required := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}
	if !required["origin_or_id"] || !required["body"] {
		t.Error("UpdateSyntheticCheck() should require origin_or_id and body")
	}
}

func TestUpdateSyntheticCheckHandler(t *testing.T) {
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
			args:        map[string]interface{}{"origin_or_id": "check-123"},
			expectError: "body is required",
		},
		{
			name: "valid update",
			args: map[string]interface{}{
				"origin_or_id": "my-check",
				"body": map[string]interface{}{
					"kind": "Dash0SyntheticCheck",
					"metadata": map[string]interface{}{
						"name": "updated-check",
					},
					"spec": map[string]interface{}{
						"enabled": true,
						"plugin": map[string]interface{}{
							"kind": "http",
							"spec": map[string]interface{}{
								"request": map[string]interface{}{
									"method":    "get",
									"url":       "https://api.example.com/v2/health",
									"redirects": "follow",
								},
							},
						},
						"schedule": map[string]interface{}{
							"interval":  "1m",
							"locations": []interface{}{"eu-west-1", "us-east-1"},
							"strategy":  "all_locations",
						},
					},
				},
			},
			expectSuccess: true,
			checkPath:     "/api/synthetic-checks/my-check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "my-check"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.UpdateSyntheticCheckHandler(context.Background(), tt.args)

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

func TestDeleteSyntheticCheckToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.DeleteSyntheticCheck()

	if tool.Name != "dash0_synthetic_checks_delete" {
		t.Errorf("DeleteSyntheticCheck() name = %s, expected dash0_synthetic_checks_delete", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("DeleteSyntheticCheck() should require 'origin_or_id'")
	}
}

func TestDeleteSyntheticCheckHandler(t *testing.T) {
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
				"origin_or_id": "check-to-delete",
			},
			expectSuccess: true,
			checkPath:     "/api/synthetic-checks/check-to-delete",
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

			result := pkg.DeleteSyntheticCheckHandler(context.Background(), tt.args)

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
		// All synthetic check tools should start with dash0_synthetic_checks_
		if !strings.HasPrefix(tool.Name, "dash0_synthetic_checks_") {
			t.Errorf("Tool %s does not follow naming convention dash0_synthetic_checks_*", tool.Name)
		}

		// Should use underscores, not hyphens (except for synthetic_checks which uses underscore)
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

func TestCreateSyntheticCheckDescription_ContainsExamples(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateSyntheticCheck()

	// Description should contain JSON examples
	if !strings.Contains(tool.Description, "Example body") {
		t.Error("CreateSyntheticCheck() description should contain example body")
	}

	// Should mention available locations
	if !strings.Contains(tool.Description, "eu-west-1") {
		t.Error("CreateSyntheticCheck() description should mention example locations")
	}

	// Should mention schedule configuration
	if !strings.Contains(tool.Description, "schedule") {
		t.Error("CreateSyntheticCheck() description should mention schedule")
	}

	// Should mention retries as optional
	if !strings.Contains(tool.Description, "retries") {
		t.Error("CreateSyntheticCheck() description should mention retries")
	}
}

func TestSyntheticCheckPluginStructure(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateSyntheticCheck()

	// Verify the schema shows the nested plugin structure
	bodyProps, ok := tool.InputSchema.Properties["body"].(map[string]interface{})
	if !ok {
		t.Fatal("body property not found")
	}

	props, ok := bodyProps["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("body.properties not found")
	}

	specProps, ok := props["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("spec property not found")
	}

	specInnerProps, ok := specProps["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("spec.properties not found")
	}

	// Should have plugin property
	if _, exists := specInnerProps["plugin"]; !exists {
		t.Error("spec should have 'plugin' property")
	}

	// Should have schedule property
	if _, exists := specInnerProps["schedule"]; !exists {
		t.Error("spec should have 'schedule' property")
	}

	// Should have enabled property
	if _, exists := specInnerProps["enabled"]; !exists {
		t.Error("spec should have 'enabled' property")
	}
}
