package api

import (
	"context"
	"testing"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/config"
)

func TestNewRegistry(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)

	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}

	if registry.handlers == nil {
		t.Error("handlers map is nil")
	}

	if registry.tools == nil {
		t.Error("tools slice is nil")
	}
}

func TestRegistryToolCount(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)

	count := registry.ToolCount()
	if count == 0 {
		t.Error("ToolCount should be greater than 0")
	}

	// Verify count matches length of tools
	if count != len(registry.AllTools()) {
		t.Errorf("ToolCount() = %d, but AllTools() has %d items", count, len(registry.AllTools()))
	}
}

func TestRegistryAllTools(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)
	tools := registry.AllTools()

	if len(tools) == 0 {
		t.Error("AllTools should return at least one tool")
	}

	// Verify all tools have names
	for i, tool := range tools {
		if tool.Name == "" {
			t.Errorf("Tool at index %d has empty name", i)
		}
	}
}

func TestRegistryHasTool(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)

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
		if !registry.HasTool(toolName) {
			t.Errorf("HasTool(%q) = false, want true", toolName)
		}
	}

	// Test unknown tool doesn't exist
	if registry.HasTool("nonexistent_tool") {
		t.Error("HasTool(nonexistent_tool) = true, want false")
	}
}

func TestRegistryGetHandler(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)

	// Test getting handler for known tool
	handler, ok := registry.GetHandler("dash0_dashboards_list")
	if !ok {
		t.Error("GetHandler(dash0_dashboards_list) returned false")
	}
	if handler == nil {
		t.Error("GetHandler(dash0_dashboards_list) returned nil handler")
	}

	// Test getting handler for unknown tool
	handler, ok = registry.GetHandler("nonexistent_tool")
	if ok {
		t.Error("GetHandler(nonexistent_tool) returned true")
	}
	if handler != nil {
		t.Error("GetHandler(nonexistent_tool) should return nil handler")
	}
}

func TestRegistryHandleTool(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)
	ctx := context.Background()

	// Test handling unknown tool
	result := registry.HandleTool(ctx, "nonexistent_tool", nil)
	if result.Success {
		t.Error("HandleTool for nonexistent tool should return error")
	}
	if result.Error == nil {
		t.Error("HandleTool for nonexistent tool should have error")
	}
	if result.Error.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", result.Error.StatusCode)
	}
}

func TestRegistryToolNames(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)
	names := registry.ToolNames()

	if len(names) == 0 {
		t.Error("ToolNames should return at least one name")
	}

	// Verify count matches
	if len(names) != registry.ToolCount() {
		t.Errorf("ToolNames() returned %d names, but ToolCount() = %d", len(names), registry.ToolCount())
	}

	// Verify all names are non-empty
	for i, name := range names {
		if name == "" {
			t.Errorf("ToolNames()[%d] is empty", i)
		}
	}

	// Verify each name has a handler
	for _, name := range names {
		if !registry.HasTool(name) {
			t.Errorf("ToolNames includes %q but HasTool returns false", name)
		}
	}
}

func TestRegistryToolsHaveDescriptions(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)
	tools := registry.AllTools()

	for _, tool := range tools {
		if tool.Description == "" {
			t.Errorf("Tool %q has empty description", tool.Name)
		}
	}
}

func TestRegistryToolsHaveInputSchema(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)
	tools := registry.AllTools()

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
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)

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

	actualCount := registry.ToolCount()
	if actualCount != expectedCount {
		t.Errorf("ToolCount() = %d, want %d", actualCount, expectedCount)

		// Print all tool names for debugging
		t.Log("Registered tools:")
		for _, name := range registry.ToolNames() {
			t.Logf("  - %s", name)
		}
	}
}

func TestRegistryToolNamingConvention(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	registry := NewRegistry(c)
	tools := registry.AllTools()

	for _, tool := range tools {
		// All tools should start with "dash0_"
		if len(tool.Name) < 6 || tool.Name[:6] != "dash0_" {
			t.Errorf("Tool %q does not start with 'dash0_'", tool.Name)
		}
	}
}
