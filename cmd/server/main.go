// Package main provides the entry point for the Dash0 MCP server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ajacobs/dash0-mcp-server/api"
	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/config"
	mcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "Dash0 MCP Server"
	serverVersion = "1.0.0"
)

func main() {
	// Ensure all logging goes to stderr to preserve stdout for MCP protocol
	log.SetOutput(os.Stderr)

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
		fmt.Fprintf(os.Stderr, "  DASH0_DEBUG       - Enable debug logging (true/false)\n")
		os.Exit(1)
	}

	// Create API client
	c := client.New(cfg)

	// Create tool registry
	registry := api.NewRegistry(c)

	// Create MCP server
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// Register all tools from the registry
	for _, tool := range registry.AllTools() {
		t := tool // Capture loop variable
		handler, ok := registry.GetHandler(t.Name)
		if !ok {
			log.Printf("Warning: no handler for tool %s\n", t.Name)
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
	fmt.Fprintf(os.Stderr, "  Tools registered: %d\n", registry.ToolCount())
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "  Debug mode: enabled\n")
	}

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
