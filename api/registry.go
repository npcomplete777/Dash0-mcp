// Package api provides the unified registry for all Dash0 MCP tools.
package api

import (
	"context"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	mcp "github.com/mark3labs/mcp-go/mcp"

	"github.com/ajacobs/dash0-mcp-server/api/alerting"
	"github.com/ajacobs/dash0-mcp-server/api/dashboards"
	"github.com/ajacobs/dash0-mcp-server/api/imports"
	"github.com/ajacobs/dash0-mcp-server/api/logs"
	"github.com/ajacobs/dash0-mcp-server/api/samplingrules"
	"github.com/ajacobs/dash0-mcp-server/api/spans"
	"github.com/ajacobs/dash0-mcp-server/api/syntheticchecks"
	"github.com/ajacobs/dash0-mcp-server/api/views"
)

// ToolHandler is a function that handles an MCP tool call.
type ToolHandler func(ctx context.Context, args map[string]interface{}) *client.ToolResult

// toolsProvider is an interface for packages that provide MCP tools.
type toolsProvider interface {
	Tools() []mcp.Tool
	Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult
}

// Registry holds all domain packages and provides unified access to tools.
type Registry struct {
	tools    []mcp.Tool
	handlers map[string]ToolHandler
}

// NewRegistry creates a new registry with all packages initialized.
func NewRegistry(c *client.Client) *Registry {
	r := &Registry{
		handlers: make(map[string]ToolHandler),
	}

	// Create package instances - order reflects logical grouping
	providers := []toolsProvider{
		// Telemetry data ingestion
		logs.New(c),
		spans.New(c),

		// Configuration management
		alerting.New(c),
		dashboards.New(c),
		views.New(c),
		syntheticchecks.New(c),
		samplingrules.New(c),

		// Migration/import
		imports.New(c),
	}

	// Collect tools and handlers from all packages
	for _, p := range providers {
		r.tools = append(r.tools, p.Tools()...)
		for name, handler := range p.Handlers() {
			r.handlers[name] = ToolHandler(handler)
		}
	}

	return r
}

// AllTools returns all MCP tools from all packages.
func (r *Registry) AllTools() []mcp.Tool {
	return r.tools
}

// HandleTool routes a tool call to the appropriate handler.
func (r *Registry) HandleTool(ctx context.Context, toolName string, args map[string]interface{}) *client.ToolResult {
	handler, ok := r.handlers[toolName]
	if !ok {
		return client.ErrorResult(404, "unknown tool: "+toolName)
	}
	return handler(ctx, args)
}

// ToolCount returns the total number of registered tools.
func (r *Registry) ToolCount() int {
	return len(r.handlers)
}

// GetHandler returns the handler for a specific tool.
func (r *Registry) GetHandler(toolName string) (ToolHandler, bool) {
	h, ok := r.handlers[toolName]
	return h, ok
}

// HasTool returns true if the registry contains a tool with the given name.
func (r *Registry) HasTool(toolName string) bool {
	_, ok := r.handlers[toolName]
	return ok
}

// ToolNames returns a list of all registered tool names.
func (r *Registry) ToolNames() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}
