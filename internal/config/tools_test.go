package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadToolsConfig(t *testing.T) {
	// Create temp directory with test config
	tmpDir := t.TempDir()

	// Create tools.yaml
	toolsYAML := `
version: "1.0"
default_profile: full
settings:
  log_enabled_tools: true
  strict_mode: false
tools:
  dashboards:
    dash0_dashboards_list:
      enabled: true
      description: "List dashboards"
      dangerous: false
    dash0_dashboards_delete:
      enabled: false
      description: "Delete dashboard"
      dangerous: true
  logs:
    dash0_logs_query:
      enabled: true
      description: "Query logs"
      dangerous: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "tools.yaml"), []byte(toolsYAML), 0644); err != nil {
		t.Fatalf("failed to write tools.yaml: %v", err)
	}

	// Create profiles directory
	profilesDir := filepath.Join(tmpDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	// Create full.yaml profile
	fullProfile := `
name: full
description: "Full profile"
enable_all: true
disable:
  - dash0_dashboards_delete
`
	if err := os.WriteFile(filepath.Join(profilesDir, "full.yaml"), []byte(fullProfile), 0644); err != nil {
		t.Fatalf("failed to write full.yaml: %v", err)
	}

	// Create minimal.yaml profile
	minimalProfile := `
name: minimal
description: "Minimal profile"
enable:
  - dash0_logs_query
disable_unlisted: true
`
	if err := os.WriteFile(filepath.Join(profilesDir, "minimal.yaml"), []byte(minimalProfile), 0644); err != nil {
		t.Fatalf("failed to write minimal.yaml: %v", err)
	}

	t.Run("LoadDefaultProfile", func(t *testing.T) {
		tc, profile, err := LoadToolsConfig(tmpDir, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tc.Version != "1.0" {
			t.Errorf("expected version 1.0, got %s", tc.Version)
		}
		if profile.Name != "full" {
			t.Errorf("expected profile 'full', got '%s'", profile.Name)
		}
	})

	t.Run("LoadSpecificProfile", func(t *testing.T) {
		tc, profile, err := LoadToolsConfig(tmpDir, "minimal")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tc == nil {
			t.Fatal("expected tools config, got nil")
		}
		if profile.Name != "minimal" {
			t.Errorf("expected profile 'minimal', got '%s'", profile.Name)
		}
	})

	t.Run("LoadFromEnvVar", func(t *testing.T) {
		os.Setenv("DASH0_MCP_PROFILE", "minimal")
		defer os.Unsetenv("DASH0_MCP_PROFILE")

		tc, profile, err := LoadToolsConfig(tmpDir, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tc == nil {
			t.Fatal("expected tools config, got nil")
		}
		if profile.Name != "minimal" {
			t.Errorf("expected profile 'minimal', got '%s'", profile.Name)
		}
	})

	t.Run("NonexistentProfile", func(t *testing.T) {
		tc, profile, err := LoadToolsConfig(tmpDir, "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should return config but nil profile
		if tc == nil {
			t.Fatal("expected tools config, got nil")
		}
		if profile != nil {
			t.Errorf("expected nil profile for nonexistent, got %v", profile)
		}
	})

	t.Run("NonexistentConfigDir", func(t *testing.T) {
		_, _, err := LoadToolsConfig("/nonexistent/path", "")
		if err == nil {
			t.Error("expected error for nonexistent config dir")
		}
	})
}

func TestGetEnabledTools(t *testing.T) {
	tc := &ToolsConfig{
		Tools: map[string]map[string]ToolDef{
			"dashboards": {
				"dash0_dashboards_list":   {Enabled: true},
				"dash0_dashboards_get":    {Enabled: true},
				"dash0_dashboards_delete": {Enabled: false},
			},
			"logs": {
				"dash0_logs_query": {Enabled: true},
				"dash0_logs_send":  {Enabled: false},
			},
		},
	}

	t.Run("NilProfile", func(t *testing.T) {
		enabled := GetEnabledTools(tc, nil)
		// Should use default enabled states
		if !enabled["dash0_dashboards_list"] {
			t.Error("expected dash0_dashboards_list to be enabled")
		}
		if !enabled["dash0_dashboards_get"] {
			t.Error("expected dash0_dashboards_get to be enabled")
		}
		if enabled["dash0_dashboards_delete"] {
			t.Error("expected dash0_dashboards_delete to be disabled")
		}
		if !enabled["dash0_logs_query"] {
			t.Error("expected dash0_logs_query to be enabled")
		}
		if enabled["dash0_logs_send"] {
			t.Error("expected dash0_logs_send to be disabled")
		}
	})

	t.Run("EnableAllProfile", func(t *testing.T) {
		profile := &Profile{
			EnableAll: true,
			Disable:   []string{"dash0_dashboards_delete"},
		}
		enabled := GetEnabledTools(tc, profile)

		if !enabled["dash0_dashboards_list"] {
			t.Error("expected dash0_dashboards_list to be enabled")
		}
		if !enabled["dash0_logs_send"] {
			t.Error("expected dash0_logs_send to be enabled (enable_all)")
		}
		if enabled["dash0_dashboards_delete"] {
			t.Error("expected dash0_dashboards_delete to be disabled (in disable list)")
		}
	})

	t.Run("DisableUnlistedProfile", func(t *testing.T) {
		profile := &Profile{
			Enable:          []string{"dash0_logs_query"},
			DisableUnlisted: true,
		}
		enabled := GetEnabledTools(tc, profile)

		if enabled["dash0_dashboards_list"] {
			t.Error("expected dash0_dashboards_list to be disabled (not in enable list)")
		}
		if !enabled["dash0_logs_query"] {
			t.Error("expected dash0_logs_query to be enabled (in enable list)")
		}
	})

	t.Run("OverrideProfile", func(t *testing.T) {
		profile := &Profile{
			Enable:  []string{"dash0_logs_send"},
			Disable: []string{"dash0_dashboards_list"},
		}
		enabled := GetEnabledTools(tc, profile)

		if enabled["dash0_dashboards_list"] {
			t.Error("expected dash0_dashboards_list to be disabled (override)")
		}
		if !enabled["dash0_logs_send"] {
			t.Error("expected dash0_logs_send to be enabled (override)")
		}
		if !enabled["dash0_dashboards_get"] {
			t.Error("expected dash0_dashboards_get to be enabled (default)")
		}
	})
}

func TestAllToolNames(t *testing.T) {
	tc := &ToolsConfig{
		Tools: map[string]map[string]ToolDef{
			"dashboards": {
				"dash0_dashboards_list": {Enabled: true},
				"dash0_dashboards_get":  {Enabled: true},
			},
			"logs": {
				"dash0_logs_query": {Enabled: true},
			},
		},
	}

	names := AllToolNames(tc)
	if len(names) != 3 {
		t.Errorf("expected 3 tool names, got %d", len(names))
	}

	expected := map[string]bool{
		"dash0_dashboards_list": true,
		"dash0_dashboards_get":  true,
		"dash0_logs_query":      true,
	}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected tool name: %s", name)
		}
	}
}
