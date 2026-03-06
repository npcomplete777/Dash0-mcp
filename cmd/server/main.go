// Package main provides the entry point for the Dash0 MCP server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/npcomplete777/dash0-mcp/api"
	"github.com/npcomplete777/dash0-mcp/internal/client"
	"github.com/npcomplete777/dash0-mcp/internal/config"
	"github.com/npcomplete777/dash0-mcp/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "Dash0 MCP Server"
	serverVersion = "1.0.0"
)

func main() {
	// Set up structured logging
	level := slog.LevelInfo
	if debug := os.Getenv("DASH0_DEBUG"); debug == "true" || debug == "1" || debug == "yes" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		slog.Error("configuration validation error", "error", err)
		slog.Info("required environment variables",
			"DASH0_AUTH_TOKEN", "Bearer token for API authentication",
		)
		slog.Info("optional environment variables",
			"DASH0_REGION", "Region (eu-west-1, us-east-1), default: eu-west-1",
			"DASH0_BASE_URL", "Custom base URL (overrides region)",
			"DASH0_DATASET", "Dataset to use for all API calls",
			"DASH0_DEBUG", "Enable debug logging (true/false)",
			"DASH0_MCP_PROFILE", "Tool profile (full, demo, readonly, minimal)",
			"DASH0_MCP_CONFIG_DIR", "Path to config directory",
		)
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
		slog.Warn("could not load tools config, using default: all tools enabled", "config_dir", configDir, "error", err)
		enabledTools = nil // nil means all tools enabled
	} else {
		enabledTools = config.GetEnabledTools(toolsConfig, profile)
		if profile != nil {
			slog.Info("loaded profile", "name", profile.Name, "description", profile.Description)
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
		for _, name := range reg.EnabledToolNames() {
			slog.Info("tool enabled", "name", name)
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
			slog.Warn("no handler for tool", "tool", t.Name)
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
	attrs := []any{
		"version", serverVersion,
		"region", cfg.Region,
		"base_url", cfg.BaseURL,
		"tools_enabled", reg.EnabledCount(),
		"tools_total", reg.ToolCount(),
	}
	if cfg.Dataset != "" {
		attrs = append(attrs, "dataset", cfg.Dataset)
	}
	if cfg.Debug {
		attrs = append(attrs, "debug", true)
	}
	slog.Info(serverName+" starting", attrs...)

	// Set up graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		slog.Info("shutdown signal received")
	}()

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
