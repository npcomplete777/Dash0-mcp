// Package config provides configuration management for the Dash0 MCP server.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ToolDef defines a single tool's configuration in tools.yaml.
type ToolDef struct {
	Enabled     bool   `yaml:"enabled"`
	Description string `yaml:"description"`
	Dangerous   bool   `yaml:"dangerous"`
}

// ToolsConfig holds all tool definitions from tools.yaml.
type ToolsConfig struct {
	Version        string                        `yaml:"version"`
	DefaultProfile string                        `yaml:"default_profile"`
	Settings       ToolsSettings                 `yaml:"settings"`
	Tools          map[string]map[string]ToolDef `yaml:"tools"`
}

// ToolsSettings contains global settings for tool management.
type ToolsSettings struct {
	LogEnabledTools bool `yaml:"log_enabled_tools"`
	StrictMode      bool `yaml:"strict_mode"`
}

// Profile defines a tool enablement profile.
type Profile struct {
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	Enable          []string `yaml:"enable"`
	Disable         []string `yaml:"disable"`
	EnableAll       bool     `yaml:"enable_all"`
	DisableUnlisted bool     `yaml:"disable_unlisted"`
}

// LoadToolsConfig loads tools.yaml and the specified profile.
// If profileName is empty, it uses DASH0_MCP_PROFILE env var or default_profile from tools.yaml.
func LoadToolsConfig(configDir, profileName string) (*ToolsConfig, *Profile, error) {
	// Load master tools.yaml
	toolsPath := filepath.Join(configDir, "tools.yaml")
	toolsData, err := os.ReadFile(toolsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read tools.yaml: %w", err)
	}

	var toolsConfig ToolsConfig
	if err := yaml.Unmarshal(toolsData, &toolsConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to parse tools.yaml: %w", err)
	}

	// Determine profile name
	if profileName == "" {
		profileName = os.Getenv("DASH0_MCP_PROFILE")
	}
	if profileName == "" {
		profileName = toolsConfig.DefaultProfile
	}
	if profileName == "" {
		profileName = "full"
	}

	// Load profile
	profilePath := filepath.Join(configDir, "profiles", profileName+".yaml")
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		// If profile doesn't exist, return config with nil profile (will enable all)
		return &toolsConfig, nil, nil
	}

	var profile Profile
	if err := yaml.Unmarshal(profileData, &profile); err != nil {
		return nil, nil, fmt.Errorf("failed to parse profile %s: %w", profileName, err)
	}

	return &toolsConfig, &profile, nil
}

// GetEnabledTools returns a map of tool names that should be enabled based on config and profile.
func GetEnabledTools(tc *ToolsConfig, p *Profile) map[string]bool {
	enabled := make(map[string]bool)

	// If no profile, enable all tools based on their default enabled state
	if p == nil {
		for _, tools := range tc.Tools {
			for toolName, toolDef := range tools {
				if toolDef.Enabled {
					enabled[toolName] = true
				}
			}
		}
		return enabled
	}

	// Build profile override sets
	profileEnabled := make(map[string]bool)
	profileDisabled := make(map[string]bool)

	for _, t := range p.Enable {
		profileEnabled[t] = true
	}
	for _, t := range p.Disable {
		profileDisabled[t] = true
	}

	// Evaluate each tool
	for _, tools := range tc.Tools {
		for toolName, toolDef := range tools {
			shouldEnable := false

			if p.EnableAll {
				// Enable all, except those in disable list
				shouldEnable = !profileDisabled[toolName]
			} else if p.DisableUnlisted {
				// Only enable tools explicitly listed
				shouldEnable = profileEnabled[toolName]
			} else {
				// Use default enabled state, with overrides
				shouldEnable = toolDef.Enabled
				if profileEnabled[toolName] {
					shouldEnable = true
				}
				if profileDisabled[toolName] {
					shouldEnable = false
				}
			}

			if shouldEnable {
				enabled[toolName] = true
			}
		}
	}

	return enabled
}

// AllToolNames returns all tool names defined in the config.
func AllToolNames(tc *ToolsConfig) []string {
	var names []string
	for _, tools := range tc.Tools {
		for name := range tools {
			names = append(names, name)
		}
	}
	return names
}
