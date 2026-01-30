package samplingrules

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
		"dash0_sampling_rules_list":   false,
		"dash0_sampling_rules_get":    false,
		"dash0_sampling_rules_create": false,
		"dash0_sampling_rules_update": false,
		"dash0_sampling_rules_delete": false,
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
		"dash0_sampling_rules_list",
		"dash0_sampling_rules_get",
		"dash0_sampling_rules_create",
		"dash0_sampling_rules_update",
		"dash0_sampling_rules_delete",
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

func TestListSamplingRulesToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.ListSamplingRules()

	if tool.Name != "dash0_sampling_rules_list" {
		t.Errorf("ListSamplingRules() name = %s, expected dash0_sampling_rules_list", tool.Name)
	}

	if tool.Description == "" {
		t.Error("ListSamplingRules() has empty description")
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("ListSamplingRules() schema type = %s, expected object", tool.InputSchema.Type)
	}

	// List has no required parameters
	if len(tool.InputSchema.Required) != 0 {
		t.Error("ListSamplingRules() should have no required parameters")
	}
}

func TestListSamplingRulesHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/sampling-rules" {
			t.Errorf("Expected /api/sampling-rules, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"name": "capture-errors", "id": "rule-1"},
			{"name": "sample-10-percent", "id": "rule-2"},
		})
	}))
	defer server.Close()

	c := client.NewWithBaseURL(server.URL, "test-token")
	pkg := New(c)

	result := pkg.ListSamplingRulesHandler(context.Background(), map[string]interface{}{})

	if !result.Success {
		t.Errorf("ListSamplingRulesHandler failed: %v", result.Error)
	}
}

func TestGetSamplingRuleToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.GetSamplingRule()

	if tool.Name != "dash0_sampling_rules_get" {
		t.Errorf("GetSamplingRule() name = %s, expected dash0_sampling_rules_get", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("GetSamplingRule() should require 'origin_or_id'")
	}
}

func TestGetSamplingRuleHandler(t *testing.T) {
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
				"origin_or_id": "my-rule",
			},
			expectSuccess: true,
			checkPath:     "/api/sampling-rules/my-rule",
		},
		{
			name: "origin_or_id with special characters",
			args: map[string]interface{}{
				"origin_or_id": "rule/with spaces",
			},
			expectSuccess: true,
			checkPath:     "/api/sampling-rules/rule%2Fwith%20spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				json.NewEncoder(w).Encode(map[string]interface{}{
					"kind": "Dash0Sampling",
					"metadata": map[string]interface{}{
						"name": "test-rule",
					},
				})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.GetSamplingRuleHandler(context.Background(), tt.args)

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

func TestCreateSamplingRuleToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateSamplingRule()

	if tool.Name != "dash0_sampling_rules_create" {
		t.Errorf("CreateSamplingRule() name = %s, expected dash0_sampling_rules_create", tool.Name)
	}

	// Description should mention Dash0Sampling format
	if !strings.Contains(tool.Description, "Dash0Sampling") {
		t.Error("CreateSamplingRule() description should mention 'Dash0Sampling'")
	}

	// Description should mention rate (not probability)
	if !strings.Contains(tool.Description, "rate") {
		t.Error("CreateSamplingRule() description should mention 'rate' for probabilistic sampling")
	}

	// Should require body
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("CreateSamplingRule() should require 'body'")
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
			t.Errorf("CreateSamplingRule() body missing property: %s", prop)
		}
	}
}

func TestCreateSamplingRuleHandler(t *testing.T) {
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
			name: "valid error condition",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"kind": "Dash0Sampling",
					"metadata": map[string]interface{}{
						"name": "capture-all-errors",
					},
					"spec": map[string]interface{}{
						"enabled": true,
						"conditions": map[string]interface{}{
							"kind": "error",
							"spec": map[string]interface{}{},
						},
					},
				},
			},
			expectSuccess: true,
		},
		{
			name: "valid probabilistic condition",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"kind": "Dash0Sampling",
					"metadata": map[string]interface{}{
						"name": "sample-10-percent",
					},
					"spec": map[string]interface{}{
						"enabled": true,
						"conditions": map[string]interface{}{
							"kind": "probabilistic",
							"spec": map[string]interface{}{
								"rate": 0.1,
							},
						},
					},
				},
			},
			expectSuccess: true,
		},
		{
			name: "valid OTTL condition",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"kind": "Dash0Sampling",
					"metadata": map[string]interface{}{
						"name": "slow-requests",
					},
					"spec": map[string]interface{}{
						"enabled": true,
						"conditions": map[string]interface{}{
							"kind": "ottl",
							"spec": map[string]interface{}{
								"ottl": "duration > 1000",
							},
						},
					},
				},
			},
			expectSuccess: true,
		},
		{
			name: "valid AND condition",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"kind": "Dash0Sampling",
					"metadata": map[string]interface{}{
						"name": "sampled-errors",
					},
					"spec": map[string]interface{}{
						"enabled": true,
						"conditions": map[string]interface{}{
							"kind": "and",
							"spec": map[string]interface{}{
								"conditions": []interface{}{
									map[string]interface{}{"kind": "error", "spec": map[string]interface{}{}},
									map[string]interface{}{"kind": "probabilistic", "spec": map[string]interface{}{"rate": 0.5}},
								},
							},
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
				if r.URL.Path != "/api/sampling-rules" {
					t.Errorf("Expected /api/sampling-rules, got %s", r.URL.Path)
				}
				json.NewDecoder(r.Body).Decode(&receivedBody)
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "new-rule"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.CreateSamplingRuleHandler(context.Background(), tt.args)

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

func TestUpdateSamplingRuleToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.UpdateSamplingRule()

	if tool.Name != "dash0_sampling_rules_update" {
		t.Errorf("UpdateSamplingRule() name = %s, expected dash0_sampling_rules_update", tool.Name)
	}

	// Description should mention rate (not probability)
	if !strings.Contains(tool.Description, "rate") {
		t.Error("UpdateSamplingRule() description should mention 'rate'")
	}

	// Should require origin_or_id and body
	if len(tool.InputSchema.Required) != 2 {
		t.Error("UpdateSamplingRule() should require 2 parameters")
	}

	required := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}
	if !required["origin_or_id"] || !required["body"] {
		t.Error("UpdateSamplingRule() should require origin_or_id and body")
	}
}

func TestUpdateSamplingRuleHandler(t *testing.T) {
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
				"origin_or_id": "my-rule",
				"body": map[string]interface{}{
					"kind": "Dash0Sampling",
					"metadata": map[string]interface{}{
						"name": "updated-rule",
					},
					"spec": map[string]interface{}{
						"enabled": true,
						"conditions": map[string]interface{}{
							"kind": "probabilistic",
							"spec": map[string]interface{}{
								"rate": 0.2,
							},
						},
					},
				},
			},
			expectSuccess: true,
			checkPath:     "/api/sampling-rules/my-rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.EscapedPath()
				receivedMethod = r.Method
				json.NewEncoder(w).Encode(map[string]interface{}{"id": "my-rule"})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.UpdateSamplingRuleHandler(context.Background(), tt.args)

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

func TestDeleteSamplingRuleToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.DeleteSamplingRule()

	if tool.Name != "dash0_sampling_rules_delete" {
		t.Errorf("DeleteSamplingRule() name = %s, expected dash0_sampling_rules_delete", tool.Name)
	}

	// Should require origin_or_id
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "origin_or_id" {
		t.Error("DeleteSamplingRule() should require 'origin_or_id'")
	}
}

func TestDeleteSamplingRuleHandler(t *testing.T) {
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
			checkPath:     "/api/sampling-rules/rule-to-delete",
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

			result := pkg.DeleteSamplingRuleHandler(context.Background(), tt.args)

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
		// All sampling rule tools should start with dash0_sampling_rules_
		if !strings.HasPrefix(tool.Name, "dash0_sampling_rules_") {
			t.Errorf("Tool %s does not follow naming convention dash0_sampling_rules_*", tool.Name)
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

func TestCreateSamplingRuleDescription_ContainsConditionTypes(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateSamplingRule()

	conditionTypes := []string{"error", "probabilistic", "ottl", "and"}

	for _, condType := range conditionTypes {
		if !strings.Contains(tool.Description, condType) {
			t.Errorf("CreateSamplingRule() description should mention '%s' condition type", condType)
		}
	}
}

func TestSamplingRuleConditionsSchema(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.CreateSamplingRule()

	// Verify the schema shows the conditions structure
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

	// Should have conditions property
	conditionsProps, exists := specInnerProps["conditions"]
	if !exists {
		t.Error("spec should have 'conditions' property")
	}

	conditionsMap, ok := conditionsProps.(map[string]interface{})
	if !ok {
		t.Fatal("conditions is not a map")
	}

	conditionsInnerProps, ok := conditionsMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("conditions.properties not found")
	}

	// Conditions should have kind and spec
	if _, exists := conditionsInnerProps["kind"]; !exists {
		t.Error("conditions should have 'kind' property")
	}

	if _, exists := conditionsInnerProps["spec"]; !exists {
		t.Error("conditions should have 'spec' property")
	}
}
