package imports

import (
	"context"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

// Package provides MCP tools for Import API operations.
type Package struct {
	client *client.Client
}

// New creates a new Imports package.
func New(c *client.Client) *Package {
	return &Package{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Package) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.ImportCheckRule(),
		p.ImportDashboard(),
		p.ImportSyntheticCheck(),
		p.ImportView(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Package) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_import_check_rule":     p.ImportCheckRuleHandler,
		"dash0_import_dashboard":      p.ImportDashboardHandler,
		"dash0_import_synthetic_check": p.ImportSyntheticCheckHandler,
		"dash0_import_view":           p.ImportViewHandler,
	}
}

// ImportCheckRule returns the dash0_import_check_rule tool definition.
func (p *Package) ImportCheckRule() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_import_check_rule",
		Description: "Import a check rule (alert rule) from another observability platform into Dash0. Supports importing Prometheus alert rules and other compatible formats.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The check rule configuration to import. Format depends on the source platform (e.g., Prometheus alert rule YAML converted to JSON).",
				},
			},
			Required: []string{"body"},
		},
	}
}

// ImportCheckRuleHandler handles the dash0_import_check_rule tool.
func (p *Package) ImportCheckRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/import/check-rule", body)
}

// ImportDashboard returns the dash0_import_dashboard tool definition.
func (p *Package) ImportDashboard() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_import_dashboard",
		Description: "Import a dashboard from another observability platform into Dash0. Supports importing Grafana dashboards and other compatible formats.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The dashboard configuration to import. For Grafana dashboards, this should be the dashboard JSON export.",
				},
			},
			Required: []string{"body"},
		},
	}
}

// ImportDashboardHandler handles the dash0_import_dashboard tool.
func (p *Package) ImportDashboardHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/import/dashboard", body)
}

// ImportSyntheticCheck returns the dash0_import_synthetic_check tool definition.
func (p *Package) ImportSyntheticCheck() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_import_synthetic_check",
		Description: "Import a synthetic check from another monitoring platform into Dash0. Supports importing checks from various synthetic monitoring tools.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The synthetic check configuration to import.",
				},
			},
			Required: []string{"body"},
		},
	}
}

// ImportSyntheticCheckHandler handles the dash0_import_synthetic_check tool.
func (p *Package) ImportSyntheticCheckHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/import/synthetic-check", body)
}

// ImportView returns the dash0_import_view tool definition.
func (p *Package) ImportView() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_import_view",
		Description: "Import a saved view from another observability platform into Dash0.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The view configuration to import.",
				},
			},
			Required: []string{"body"},
		},
	}
}

// ImportViewHandler handles the dash0_import_view tool.
func (p *Package) ImportViewHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/import/view", body)
}

// Register registers all import tools with the registry.
func Register(reg *registry.Registry, c *client.Client) {
	p := New(c)
	for _, tool := range p.Tools() {
		handler := p.Handlers()[tool.Name]
		reg.Register(tool, handler)
	}
}
