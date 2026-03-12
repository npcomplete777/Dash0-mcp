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
	// Total: 2 + 2 + 6 + 5 + 5 + 5 + 5 + 4 = 34
	expectedCount := 34

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

// setupRegistryWithProfile creates a registry filtered by the given profile.
// It loads the real tools.yaml and profile YAML from the config directory.
func setupRegistryWithProfile(t *testing.T, profileName string) *registry.Registry {
	t.Helper()
	configDir := "../config"

	tc, profile, err := config.LoadToolsConfig(configDir, profileName)
	if err != nil {
		t.Fatalf("failed to load tools config with profile %q: %v", profileName, err)
	}

	enabledTools := config.GetEnabledTools(tc, profile)

	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)
	reg := registry.New(enabledTools)
	RegisterAllTools(reg, c)
	return reg
}

func TestProfileFull_EnablesAllExceptDeletes(t *testing.T) {
	reg := setupRegistryWithProfile(t, "full")

	// Full profile should enable all tools except deletes
	shouldBeEnabled := []string{
		"dash0_dashboards_list",
		"dash0_dashboards_get",
		"dash0_dashboards_create",
		"dash0_dashboards_update",
		"dash0_alerting_check_rules_list",
		"dash0_alerting_check_rules_get",
		"dash0_alerting_check_rules_create",
		"dash0_alerting_check_rules_update",
		"dash0_alerting_active_alerts",
		"dash0_synthetic_checks_list",
		"dash0_synthetic_checks_get",
		"dash0_synthetic_checks_create",
		"dash0_synthetic_checks_update",
		"dash0_sampling_rules_list",
		"dash0_sampling_rules_get",
		"dash0_sampling_rules_create",
		"dash0_sampling_rules_update",
		"dash0_views_list",
		"dash0_views_get",
		"dash0_views_create",
		"dash0_views_update",
		"dash0_logs_query",
		"dash0_logs_send",
		"dash0_spans_query",
		"dash0_spans_send",
		"dash0_import_dashboard",
		"dash0_import_check_rule",
		"dash0_import_synthetic_check",
		"dash0_import_view",
	}

	shouldBeDisabled := []string{
		"dash0_dashboards_delete",
		"dash0_alerting_check_rules_delete",
		"dash0_synthetic_checks_delete",
		"dash0_sampling_rules_delete",
		"dash0_views_delete",
	}

	for _, name := range shouldBeEnabled {
		if !reg.IsEnabled(name) {
			t.Errorf("full profile: %s should be enabled", name)
		}
	}

	for _, name := range shouldBeDisabled {
		if reg.IsEnabled(name) {
			t.Errorf("full profile: %s should be disabled", name)
		}
	}

	// GetEnabledTools should not include disabled tools
	enabledTools := reg.GetEnabledTools()
	enabledNames := make(map[string]bool)
	for _, tool := range enabledTools {
		enabledNames[tool.Name] = true
	}

	for _, name := range shouldBeDisabled {
		if enabledNames[name] {
			t.Errorf("full profile: GetEnabledTools() should not include %s", name)
		}
	}
}

func TestProfileMinimal_OnlyEnablesCoreTools(t *testing.T) {
	reg := setupRegistryWithProfile(t, "minimal")

	shouldBeEnabled := []string{
		"dash0_logs_query",
		"dash0_spans_query",
		"dash0_dashboards_list",
		"dash0_dashboards_get",
		"dash0_alerting_check_rules_list",
		"dash0_alerting_check_rules_get",
		"dash0_synthetic_checks_list",
		"dash0_synthetic_checks_get",
	}

	shouldBeDisabled := []string{
		"dash0_dashboards_create",
		"dash0_dashboards_update",
		"dash0_dashboards_delete",
		"dash0_alerting_check_rules_create",
		"dash0_alerting_check_rules_update",
		"dash0_alerting_check_rules_delete",
		"dash0_alerting_active_alerts",
		"dash0_logs_send",
		"dash0_spans_send",
		"dash0_views_list",
		"dash0_views_get",
		"dash0_views_create",
		"dash0_views_update",
		"dash0_views_delete",
		"dash0_sampling_rules_list",
		"dash0_sampling_rules_create",
		"dash0_synthetic_checks_create",
		"dash0_synthetic_checks_delete",
		"dash0_import_dashboard",
		"dash0_import_check_rule",
	}

	for _, name := range shouldBeEnabled {
		if !reg.IsEnabled(name) {
			t.Errorf("minimal profile: %s should be enabled", name)
		}
	}

	for _, name := range shouldBeDisabled {
		if reg.IsEnabled(name) {
			t.Errorf("minimal profile: %s should be disabled", name)
		}
	}

	enabledCount := reg.EnabledCount()
	if enabledCount != 8 {
		t.Errorf("minimal profile: EnabledCount() = %d, want 8", enabledCount)
		t.Logf("Enabled tools: %v", reg.EnabledToolNames())
	}
}

func TestProfileReadonly_NoWriteTools(t *testing.T) {
	reg := setupRegistryWithProfile(t, "readonly")

	shouldBeEnabled := []string{
		"dash0_dashboards_list",
		"dash0_dashboards_get",
		"dash0_alerting_check_rules_list",
		"dash0_alerting_check_rules_get",
		"dash0_synthetic_checks_list",
		"dash0_synthetic_checks_get",
		"dash0_sampling_rules_list",
		"dash0_sampling_rules_get",
		"dash0_views_list",
		"dash0_views_get",
		"dash0_logs_query",
		"dash0_spans_query",
	}

	// All write operations should be disabled
	shouldBeDisabled := []string{
		"dash0_dashboards_create",
		"dash0_dashboards_update",
		"dash0_dashboards_delete",
		"dash0_alerting_check_rules_create",
		"dash0_alerting_check_rules_update",
		"dash0_alerting_check_rules_delete",
		"dash0_alerting_active_alerts",
		"dash0_synthetic_checks_create",
		"dash0_synthetic_checks_update",
		"dash0_synthetic_checks_delete",
		"dash0_sampling_rules_create",
		"dash0_sampling_rules_update",
		"dash0_sampling_rules_delete",
		"dash0_views_create",
		"dash0_views_update",
		"dash0_views_delete",
		"dash0_logs_send",
		"dash0_spans_send",
		"dash0_import_dashboard",
		"dash0_import_check_rule",
		"dash0_import_synthetic_check",
		"dash0_import_view",
	}

	for _, name := range shouldBeEnabled {
		if !reg.IsEnabled(name) {
			t.Errorf("readonly profile: %s should be enabled", name)
		}
	}

	for _, name := range shouldBeDisabled {
		if reg.IsEnabled(name) {
			t.Errorf("readonly profile: %s should be disabled", name)
		}
	}

	enabledCount := reg.EnabledCount()
	if enabledCount != 12 {
		t.Errorf("readonly profile: EnabledCount() = %d, want 12", enabledCount)
		t.Logf("Enabled tools: %v", reg.EnabledToolNames())
	}
}

func TestProfileDemo_WorkflowTools(t *testing.T) {
	reg := setupRegistryWithProfile(t, "demo")

	shouldBeEnabled := []string{
		"dash0_dashboards_list",
		"dash0_dashboards_get",
		"dash0_dashboards_create",
		"dash0_dashboards_update",
		"dash0_alerting_check_rules_list",
		"dash0_alerting_check_rules_get",
		"dash0_alerting_check_rules_create",
		"dash0_alerting_check_rules_update",
		"dash0_synthetic_checks_list",
		"dash0_synthetic_checks_get",
		"dash0_synthetic_checks_create",
		"dash0_synthetic_checks_update",
		"dash0_logs_query",
		"dash0_spans_query",
		"dash0_import_dashboard",
		"dash0_import_check_rule",
		"dash0_views_list",
		"dash0_views_get",
		"dash0_views_create",
	}

	shouldBeDisabled := []string{
		"dash0_dashboards_delete",
		"dash0_alerting_check_rules_delete",
		"dash0_alerting_active_alerts",
		"dash0_synthetic_checks_delete",
		"dash0_sampling_rules_list",
		"dash0_sampling_rules_create",
		"dash0_sampling_rules_delete",
		"dash0_views_delete",
		"dash0_views_update",
		"dash0_logs_send",
		"dash0_spans_send",
		"dash0_import_synthetic_check",
		"dash0_import_view",
	}

	for _, name := range shouldBeEnabled {
		if !reg.IsEnabled(name) {
			t.Errorf("demo profile: %s should be enabled", name)
		}
	}

	for _, name := range shouldBeDisabled {
		if reg.IsEnabled(name) {
			t.Errorf("demo profile: %s should be disabled", name)
		}
	}

	enabledCount := reg.EnabledCount()
	if enabledCount != 19 {
		t.Errorf("demo profile: EnabledCount() = %d, want 19", enabledCount)
		t.Logf("Enabled tools: %v", reg.EnabledToolNames())
	}
}

func TestProfileDisabledToolCannotBeCalled(t *testing.T) {
	reg := setupRegistryWithProfile(t, "minimal")
	ctx := context.Background()

	// dash0_dashboards_create should be disabled in minimal profile
	result := reg.Call(ctx, "dash0_dashboards_create", map[string]interface{}{})
	if result.Success {
		t.Error("calling disabled tool should fail")
	}
	if result.Error == nil || result.Error.StatusCode != 403 {
		t.Errorf("expected 403 error for disabled tool, got: %v", result.Error)
	}
}

func TestProfileEnabledToolCanBeCalled(t *testing.T) {
	reg := setupRegistryWithProfile(t, "minimal")

	// dash0_logs_query is enabled in minimal - handler should exist
	handler := reg.GetHandler("dash0_logs_query")
	if handler == nil {
		t.Error("handler for enabled tool dash0_logs_query should not be nil")
	}
}

func TestProfileGetEnabledToolsExcludesDisabled(t *testing.T) {
	reg := setupRegistryWithProfile(t, "readonly")

	enabledTools := reg.GetEnabledTools()
	enabledNames := make(map[string]bool)
	for _, tool := range enabledTools {
		enabledNames[tool.Name] = true
	}

	// Verify no write tools are in the enabled list
	writeTools := []string{
		"dash0_dashboards_create",
		"dash0_dashboards_update",
		"dash0_dashboards_delete",
		"dash0_logs_send",
		"dash0_spans_send",
	}

	for _, name := range writeTools {
		if enabledNames[name] {
			t.Errorf("readonly profile: GetEnabledTools() should not include %s", name)
		}
	}
}
