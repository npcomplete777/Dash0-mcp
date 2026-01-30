package dashboards

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

// Package provides MCP tools for Dashboards API operations.
type Package struct {
	client *client.Client
}

// New creates a new Dashboards package.
func New(c *client.Client) *Package {
	return &Package{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Package) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.ListDashboards(),
		p.GetDashboard(),
		p.CreateDashboard(),
		p.UpdateDashboard(),
		p.DeleteDashboard(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Package) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_dashboards_list":   p.ListDashboardsHandler,
		"dash0_dashboards_get":    p.GetDashboardHandler,
		"dash0_dashboards_create": p.CreateDashboardHandler,
		"dash0_dashboards_update": p.UpdateDashboardHandler,
		"dash0_dashboards_delete": p.DeleteDashboardHandler,
	}
}

// ListDashboards returns the dash0_dashboards_list tool definition.
func (p *Package) ListDashboards() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_dashboards_list",
		Description: "List all dashboards in Dash0. Returns dashboard metadata including names, IDs, and modification times.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
}

// ListDashboardsHandler handles the dash0_dashboards_list tool.
func (p *Package) ListDashboardsHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	return p.client.Get(ctx, "/api/dashboards")
}

// GetDashboard returns the dash0_dashboards_get tool definition.
func (p *Package) GetDashboard() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_dashboards_get",
		Description: "Get a specific dashboard by its origin or ID, including all panels and configuration.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the dashboard to retrieve.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// GetDashboardHandler handles the dash0_dashboards_get tool.
func (p *Package) GetDashboardHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf("/api/dashboards/%s", url.PathEscape(originOrID))
	return p.client.Get(ctx, path)
}

// CreateDashboard returns the dash0_dashboards_create tool definition.
func (p *Package) CreateDashboard() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_dashboards_create",
		Description: `Create a new dashboard in Dash0 with panels for visualizing metrics, logs, and traces.

IMPORTANT: Dashboards use Kubernetes CRD format (Perses format).

Required structure:
- kind: Must be "PersesDashboard"
- metadata.name: Dashboard identifier (lowercase, alphanumeric, hyphens)
- spec.display.name: Human-readable dashboard title
- spec.panels: Array of panel definitions (can be empty)

Example body:
{
  "kind": "PersesDashboard",
  "metadata": {"name": "my-service-dashboard"},
  "spec": {
    "display": {"name": "My Service Dashboard"},
    "panels": []
  }
}

With panels:
{
  "kind": "PersesDashboard",
  "metadata": {"name": "api-metrics"},
  "spec": {
    "display": {"name": "API Metrics Dashboard"},
    "panels": [
      {
        "kind": "Panel",
        "spec": {
          "display": {"name": "Request Rate"},
          "plugin": {
            "kind": "TimeSeriesChart",
            "spec": {"legend": {"position": "bottom"}}
          },
          "queries": [
            {
              "kind": "TimeSeriesQuery",
              "spec": {
                "plugin": {
                  "kind": "PrometheusTimeSeriesQuery",
                  "spec": {"query": "rate(http_requests_total[5m])"}
                }
              }
            }
          ]
        }
      }
    ]
  }
}`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The dashboard configuration in Perses CRD format.",
					"properties": map[string]interface{}{
						"kind": map[string]interface{}{
							"type":        "string",
							"description": "Must be 'PersesDashboard'",
							"enum":        []string{"PersesDashboard"},
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "Dashboard metadata",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "Dashboard identifier (lowercase, alphanumeric, hyphens)",
								},
							},
							"required": []interface{}{"name"},
						},
						"spec": map[string]interface{}{
							"type":        "object",
							"description": "Dashboard specification",
							"properties": map[string]interface{}{
								"display": map[string]interface{}{
									"type":        "object",
									"description": "Display settings",
									"properties": map[string]interface{}{
										"name": map[string]interface{}{
											"type":        "string",
											"description": "Human-readable dashboard title",
										},
									},
								},
								"panels": map[string]interface{}{
									"type":        "array",
									"description": "Array of panel definitions",
								},
							},
						},
					},
					"required": []interface{}{"kind", "metadata", "spec"},
				},
			},
			Required: []string{"body"},
		},
	}
}

// CreateDashboardHandler handles the dash0_dashboards_create tool.
func (p *Package) CreateDashboardHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/dashboards", body)
}

// UpdateDashboard returns the dash0_dashboards_update tool definition.
func (p *Package) UpdateDashboard() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_dashboards_update",
		Description: `Update an existing dashboard by its origin or ID.

The body should follow the same Perses CRD format as create:
{
  "kind": "PersesDashboard",
  "metadata": {"name": "updated-dashboard"},
  "spec": {
    "display": {"name": "Updated Dashboard Title"},
    "panels": []
  }
}`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the dashboard to update.",
				},
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The updated dashboard configuration in Perses CRD format.",
					"properties": map[string]interface{}{
						"kind": map[string]interface{}{
							"type":        "string",
							"description": "Must be 'PersesDashboard'",
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "Dashboard metadata with name",
						},
						"spec": map[string]interface{}{
							"type":        "object",
							"description": "Dashboard specification with display and panels",
						},
					},
					"required": []interface{}{"kind", "metadata", "spec"},
				},
			},
			Required: []string{"origin_or_id", "body"},
		},
	}
}

// UpdateDashboardHandler handles the dash0_dashboards_update tool.
func (p *Package) UpdateDashboardHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	path := fmt.Sprintf("/api/dashboards/%s", url.PathEscape(originOrID))
	return p.client.Put(ctx, path, body)
}

// DeleteDashboard returns the dash0_dashboards_delete tool definition.
func (p *Package) DeleteDashboard() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_dashboards_delete",
		Description: "Delete a dashboard by its origin or ID.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the dashboard to delete.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// DeleteDashboardHandler handles the dash0_dashboards_delete tool.
func (p *Package) DeleteDashboardHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf("/api/dashboards/%s", url.PathEscape(originOrID))
	return p.client.Delete(ctx, path)
}

// Register registers all dashboard tools with the registry.
func Register(reg *registry.Registry, c *client.Client) {
	p := New(c)
	for _, tool := range p.Tools() {
		handler := p.Handlers()[tool.Name]
		reg.Register(tool, handler)
	}
}
