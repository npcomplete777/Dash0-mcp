package api

import (
	"context"
	"testing"

	"github.com/npcomplete777/dash0-mcp/internal/client"
	"github.com/npcomplete777/dash0-mcp/internal/config"
	"github.com/npcomplete777/dash0-mcp/internal/registry"
)

func setupRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)
	reg := registry.New(nil)
	RegisterAllTools(reg, c)
	return reg
}

func TestRegistryToolCount(t *testing.T) {
	reg := setupRegistry(t)

	count := reg.ToolCount()
	if count == 0 {
		t.Error("ToolCount should be greater than 0")
	}

	// Verify count matches length of enabled tools
	if count != len(reg.GetEnabledTools()) {
		t.Errorf("ToolCount() = %d, but GetEnabledTools() has %d items", count, len(reg.GetEnabledTools()))
	}
}

func TestRegistryAllTools(t *testing.T) {
	reg := setupRegistry(t)
	tools := reg.GetEnabledTools()

	if len(tools) == 0 {
		t.Error("GetEnabledTools should return at least one tool")
	}

	// Verify all tools have names
	for i, tool := range tools {
		if tool.Name == "" {
			t.Errorf("Tool at index %d has empty name", i)
		}
	}
}

func TestRegistryHasTool(t *testing.T) {
	reg := setupRegistry(t)

	// Test known tools exist
	knownTools := []string{
		"dash0_dashboards_list",
		"dash0_dashboards_get",
		"dash0_dashboards_create",
		"dash0_dashboards_update",
		"dash0_dashboards_delete",
		"dash0_alerting_check_rules_list",
		"dash0_alerting_check_rules_get",
		"dash0_alerting_check_rules_create",
		"dash0_views_list",
		"dash0_logs_send",
		"dash0_logs_query",
		"dash0_spans_send",
		"dash0_spans_query",
		"dash0_synthetic_checks_list",
		"dash0_sampling_rules_list",
		"dash0_import_dashboard",
	}

	for _, toolName := range knownTools {
		if reg.GetHandler(toolName) == nil {
			t.Errorf("GetHandler(%q) = nil, want non-nil", toolName)
		}
	}

	// Test unknown tool doesn't exist
	if reg.GetHandler("nonexistent_tool") != nil {
		t.Error("GetHandler(nonexistent_tool) should return nil")
	}
}

func TestRegistryCallUnknownTool(t *testing.T) {
	reg := setupRegistry(t)
	ctx := context.Background()

	// Test calling unknown tool
	result := reg.Call(ctx, "nonexistent_tool", nil)
	if result.Success {
		t.Error("Call for nonexistent tool should return error")
	}
	if result.Error == nil {
		t.Error("Call for nonexistent tool should have error")
	}
	if result.Error.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", result.Error.StatusCode)
	}
}

func TestRegistryToolNames(t *testing.T) {
	reg := setupRegistry(t)
	names := reg.AllToolNames()

	if len(names) == 0 {
		t.Error("AllToolNames should return at least one name")
	}

	// Verify count matches
	if len(names) != reg.ToolCount() {
		t.Errorf("AllToolNames() returned %d names, but ToolCount() = %d", len(names), reg.ToolCount())
	}

	// Verify all names are non-empty
	for i, name := range names {
		if name == "" {
			t.Errorf("AllToolNames()[%d] is empty", i)
		}
	}

	// Verify each name has a handler
	for _, name := range names {
		if reg.GetHandler(name) == nil {
			t.Errorf("AllToolNames includes %q but GetHandler returns nil", name)
		}
	}
}

func TestRegistryToolsHaveDescriptions(t *testing.T) {
	reg := setupRegistry(t)
	tools := reg.GetEnabledTools()

	for _, tool := range tools {
		if tool.Description == "" {
			t.Errorf("Tool %q has empty description", tool.Name)
		}
	}
}

func TestRegistryToolsHaveInputSchema(t *testing.T) {
	reg := setupRegistry(t)
	tools := reg.GetEnabledTools()

	for _, tool := range tools {
		if tool.InputSchema.Type == "" {
			t.Errorf("Tool %q has empty InputSchema.Type", tool.Name)
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("Tool %q has InputSchema.Type = %q, want \"object\"", tool.Name, tool.InputSchema.Type)
		}
	}
}

func TestRegistryExpectedToolCount(t *testing.T) {
	reg := setupRegistry(t)

	// Count expected tools:
	// logs: 2 (send, query)
	// spans: 2 (send, query)
	// alerting: 5 (list, get, create, update, delete)
	// dashboards: 5 (list, get, create, update, delete)
	// views: 5 (list, get, create, update, delete)
	// syntheticchecks: 5 (list, get, create, update, delete)
	// samplingrules: 5 (list, get, create, update, delete)
	// imports: 4 (check_rule, dashboard, synthetic_check, view)
	// Total: 2 + 2 + 5 + 5 + 5 + 5 + 5 + 4 = 33
	expectedCount := 33

	actualCount := reg.ToolCount()
	if actualCount != expectedCount {
		t.Errorf("ToolCount() = %d, want %d", actualCount, expectedCount)

		// Print all tool names for debugging
		t.Log("Registered tools:")
		for _, name := range reg.AllToolNames() {
			t.Logf("  - %s", name)
		}
	}
}

func TestRegistryToolNamingConvention(t *testing.T) {
	reg := setupRegistry(t)
	tools := reg.GetEnabledTools()

	for _, tool := range tools {
		// All tools should start with "dash0_"
		if len(tool.Name) < 6 || tool.Name[:6] != "dash0_" {
			t.Errorf("Tool %q does not start with 'dash0_'", tool.Name)
		}
	}
}
