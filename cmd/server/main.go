// Package main provides the entry point for the Dash0 MCP server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ajacobs/dash0-mcp-server/api"
	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/config"
	"github.com/ajacobs/dash0-mcp-server/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "Dash0 MCP Server"
	serverVersion = "1.0.0"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nRequired environment variables:\n")
		fmt.Fprintf(os.Stderr, "  DASH0_AUTH_TOKEN  - Bearer token for API authentication\n")
		fmt.Fprintf(os.Stderr, "\nOptional environment variables:\n")
		fmt.Fprintf(os.Stderr, "  DASH0_REGION      - Region (eu-west-1, us-east-1), default: eu-west-1\n")
		fmt.Fprintf(os.Stderr, "  DASH0_BASE_URL    - Custom base URL (overrides region)\n")
		fmt.Fprintf(os.Stderr, "  DASH0_DATASET     - Dataset to use for all API calls\n")
		fmt.Fprintf(os.Stderr, "  DASH0_DEBUG       - Enable debug logging (true/false)\n")
		fmt.Fprintf(os.Stderr, "  DASH0_MCP_PROFILE - Tool profile (full, demo, readonly, minimal)\n")
		fmt.Fprintf(os.Stderr, "  DASH0_MCP_CONFIG_DIR - Path to config directory\n")
		os.Exit(1)
	}

	// Determine config directory for tools
	configDir := os.Getenv("DASH0_MCP_CONFIG_DIR")
	if configDir == "" {
		// Try relative to executable
		if exe, err := os.Executable(); err == nil {
			configDir = filepath.Join(filepath.Dir(exe), "config")
		}
	}
	if configDir == "" {
		// Fallback to current directory
		configDir = "config"
	}

	// Load tools config and profile
	profileName := os.Getenv("DASH0_MCP_PROFILE")
	var toolsConfig *config.ToolsConfig
	var profile *config.Profile
	var enabledTools map[string]bool

	toolsConfig, profile, err = config.LoadToolsConfig(configDir, profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load tools config from %s: %v\n", configDir, err)
		fmt.Fprintf(os.Stderr, "Using default: all tools enabled\n")
		enabledTools = nil // nil means all tools enabled
	} else {
		enabledTools = config.GetEnabledTools(toolsConfig, profile)
		if profile != nil {
			fmt.Fprintf(os.Stderr, "Loaded profile: %s\n", profile.Name)
			if profile.Description != "" {
				fmt.Fprintf(os.Stderr, "  Description: %s\n", profile.Description)
			}
		}
	}

	// Create API client
	c := client.New(cfg)

	// Create registry with enabled tools filter
	reg := registry.New(enabledTools)

	// Register ALL tool handlers (registry filters by enabled)
	api.RegisterAllTools(reg, c)

	// Log enabled tools if configured
	if toolsConfig != nil && toolsConfig.Settings.LogEnabledTools {
		fmt.Fprintf(os.Stderr, "Enabled tools:\n")
		for _, name := range reg.EnabledToolNames() {
			fmt.Fprintf(os.Stderr, "  ✓ %s\n", name)
		}
	}

	// Create MCP server
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// Register enabled tools with MCP
	for _, tool := range reg.GetEnabledTools() {
		t := tool // Capture loop variable
		handler := reg.GetHandler(t.Name)
		if handler == nil {
			fmt.Fprintf(os.Stderr, "Warning: no handler for tool %s\n", t.Name)
			continue
		}

		s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Extract arguments
			var args map[string]interface{}
			if req.Params.Arguments != nil {
				args = req.Params.Arguments
			} else {
				args = make(map[string]interface{})
			}

			// Execute handler
			result := handler(ctx, args)

			// Convert result to MCP format
			if result.Error != nil {
				return mcp.NewToolResultError(result.Error.Detail), nil
			}

			// Marshal data to JSON
			data, err := json.Marshal(result.Data)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
			}

			return mcp.NewToolResultText(string(data)), nil
		})
	}

	// Log startup information
	fmt.Fprintf(os.Stderr, "%s v%s starting\n", serverName, serverVersion)
	fmt.Fprintf(os.Stderr, "  Region: %s\n", cfg.Region)
	fmt.Fprintf(os.Stderr, "  Base URL: %s\n", cfg.BaseURL)
	if cfg.Dataset != "" {
		fmt.Fprintf(os.Stderr, "  Dataset: %s\n", cfg.Dataset)
	}
	fmt.Fprintf(os.Stderr, "  Tools registered: %d/%d\n", reg.EnabledCount(), reg.ToolCount())
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "  Debug mode: enabled\n")
	}

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
