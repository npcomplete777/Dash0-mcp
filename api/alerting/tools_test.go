package alerting

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
		"dash0_alerting_check_rules_list":   false,
		"dash0_alerting_check_rules_get":    false,
		"dash0_alerting_check_rules_create": false,
		"dash0_alerting_check_rules_update": false,
		"dash0_alerting_check_rules_delete": false,
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
		"dash0_alerting_check_rules_list",
		"dash0_alerting_check_rules_get",
		"dash0_alerting_check_rules_create",
		"dash0_alerting_check_rules_update",
		"dash0_alerting_check_rules_delete",
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

func TestListCheckRulesToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.ListCheckRules()

	if tool.Name != "dash0_alerting_check_rules_list" {
		t.Errorf("ListCheckRules() name = %s, expected dash0_alerting_check_rules_list", tool.Name)
	}

	if tool.Description == "" {
		t.Error("ListCheckRules() has empty description")
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("ListCheckRules() schema type = %s, expected object", tool.InputSchema.Type)
	}

	// List has no required parameters
	if len(tool.InputSchema.Required) != 0 {
		t.Error("ListCheckRules() should have no required parameters")
	}
}

func TestListCheckRulesHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/alerting/check-rules" {
			t.Errorf("Expected /api/alerting/check-rules, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "HighErrorRate", "id": "rule-1"},
			{"name": "LowMemory", "id": "rule-2"},
		})
	}))
	defer server.Close()

	c := client.NewWithBaseURL(server.URL, "test-token")
	pkg := New(c)

	result := pkg.ListCheckRulesHandler(context.Background(), map[string]interface{}{})

	if !result.Success {
		t.Errorf("ListCheckRulesHandler failed: %v", result.Error)
	}
}

func TestGetCheckRuleToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.GetCheckRule()

	if tool.Name != "dash0_alerting_check_rules_get" {
		t.Errorf("GetCheckRule() name = %s, expected dash0_alerting_check_rules_get", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("GetCheckRule() should require 'origin_or_id'")
	}
}

func TestGetCheckRuleHandler(t *testing.T) {
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
				"origin_or_id": "rule-123",
			},
			expectSuccess: true,
			checkPath:     "/api/alerting/check-rules/rule-123",
		},
		{
			name: "origin_or_id with special characters",
			args: map[string]interface{}{
				"origin_or_id": "rule/with spaces",
			},
			expectSuccess: true,
			checkPath:     "/api/alerting/check-rules/rule%2Fwith%20spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				json.NewEncoder(w).Encode(map[string]interface{}{"name": "TestRule"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.GetCheckRuleHandler(context.Background(), tt.args)

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

func TestCreateCheckRuleToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateCheckRule()

	if tool.Name != "dash0_alerting_check_rules_create" {
		t.Errorf("CreateCheckRule() name = %s, expected dash0_alerting_check_rules_create", tool.Name)
	}

	// Description should mention plain JSON format (not CRD)
	if !strings.Contains(tool.Description, "plain JSON") {
		t.Error("CreateCheckRule() description should mention 'plain JSON format'")
	}

	// Should require body
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("CreateCheckRule() should require 'body'")
	}

	// Body should have properties for name, expression, interval, for
	bodyProps, ok := tool.InputSchema.Properties["body"].(map[string]interface{})
	if !ok {
		t.Fatal("body property not found in schema")
	}

	props, ok := bodyProps["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("body.properties not found in schema")
	}

	expectedProps := []string{"name", "expression", "interval", "for", "labels", "annotations", "keepFiringFor"}
	for _, prop := range expectedProps {
		if _, exists := props[prop]; !exists {
			t.Errorf("CreateCheckRule() body missing property: %s", prop)
		}
	}
}

func TestCreateCheckRuleHandler(t *testing.T) {
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
					"name":       "HighErrorRate",
					"expression": "rate(http_errors_total[5m]) > 0.05",
					"interval":   "1m",
					"for":        "5m",
					"labels": map[string]interface{}{
						"severity": "critical",
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
				if r.URL.Path != "/api/alerting/check-rules" {
					t.Errorf("Expected /api/alerting/check-rules, got %s", r.URL.Path)
				}
				json.NewDecoder(r.Body).Decode(&receivedBody)
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "new-rule"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.CreateCheckRuleHandler(context.Background(), tt.args)

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

func TestUpdateCheckRuleToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.UpdateCheckRule()

	if tool.Name != "dash0_alerting_check_rules_update" {
		t.Errorf("UpdateCheckRule() name = %s, expected dash0_alerting_check_rules_update", tool.Name)
	}

	// Should require origin_or_id and body
	if len(tool.InputSchema.Required) != 2 {
		t.Error("UpdateCheckRule() should require 2 parameters")
	}

	required := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}
	if !required["origin_or_id"] || !required["body"] {
		t.Error("UpdateCheckRule() should require origin_or_id and body")
	}
}

func TestUpdateCheckRuleHandler(t *testing.T) {
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
			args:        map[string]interface{}{"origin_or_id": "rule-123"},
			expectError: "body is required",
		},
		{
			name: "valid update",
			args: map[string]interface{}{
				"origin_or_id": "rule-123",
				"body": map[string]interface{}{
					"name":       "UpdatedRule",
					"expression": "rate(errors[5m]) > 0.1",
					"interval":   "1m",
					"for":        "5m",
				},
			},
			expectSuccess: true,
			checkPath:     "/api/alerting/check-rules/rule-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "rule-123"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.UpdateCheckRuleHandler(context.Background(), tt.args)

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

func TestDeleteCheckRuleToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.DeleteCheckRule()

	if tool.Name != "dash0_alerting_check_rules_delete" {
		t.Errorf("DeleteCheckRule() name = %s, expected dash0_alerting_check_rules_delete", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("DeleteCheckRule() should require 'origin_or_id'")
	}
}

func TestDeleteCheckRuleHandler(t *testing.T) {
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
				"origin_or_id": "rule-to-delete",
			},
			expectSuccess: true,
			checkPath:     "/api/alerting/check-rules/rule-to-delete",
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

			result := pkg.DeleteCheckRuleHandler(context.Background(), tt.args)

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
		// All alerting tools should start with dash0_alerting_check_rules_
		if !strings.HasPrefix(tool.Name, "dash0_alerting_check_rules_") {
			t.Errorf("Tool %s does not follow naming convention dash0_alerting_check_rules_*", tool.Name)
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
