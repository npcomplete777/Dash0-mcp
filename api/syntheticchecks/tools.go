package syntheticchecks

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

// Package provides MCP tools for Synthetic Checks API operations.
type Package struct {
	client *client.Client
}

// New creates a new Synthetic Checks package.
func New(c *client.Client) *Package {
	return &Package{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Package) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.ListSyntheticChecks(),
		p.GetSyntheticCheck(),
		p.CreateSyntheticCheck(),
		p.UpdateSyntheticCheck(),
		p.DeleteSyntheticCheck(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Package) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_synthetic_checks_list":   p.ListSyntheticChecksHandler,
		"dash0_synthetic_checks_get":    p.GetSyntheticCheckHandler,
		"dash0_synthetic_checks_create": p.CreateSyntheticCheckHandler,
		"dash0_synthetic_checks_update": p.UpdateSyntheticCheckHandler,
		"dash0_synthetic_checks_delete": p.DeleteSyntheticCheckHandler,
	}
}

// ListSyntheticChecks returns the dash0_synthetic_checks_list tool definition.
func (p *Package) ListSyntheticChecks() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_synthetic_checks_list",
		Description: "List all synthetic checks in Dash0. Synthetic checks proactively monitor application availability and performance from multiple locations.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
}

// ListSyntheticChecksHandler handles the dash0_synthetic_checks_list tool.
func (p *Package) ListSyntheticChecksHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	return p.client.Get(ctx, "/api/synthetic-checks")
}

// GetSyntheticCheck returns the dash0_synthetic_checks_get tool definition.
func (p *Package) GetSyntheticCheck() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_synthetic_checks_get",
		Description: "Get a specific synthetic check by its origin or ID, including configuration and check results.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the synthetic check to retrieve.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// GetSyntheticCheckHandler handles the dash0_synthetic_checks_get tool.
func (p *Package) GetSyntheticCheckHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf("/api/synthetic-checks/%s", url.PathEscape(originOrID))
	return p.client.Get(ctx, path)
}

// CreateSyntheticCheck returns the dash0_synthetic_checks_create tool definition.
func (p *Package) CreateSyntheticCheck() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_synthetic_checks_create",
		Description: `Create a new synthetic check in Dash0 for proactive monitoring of endpoints, APIs, or browser-based workflows.

IMPORTANT: Synthetic checks use Kubernetes CRD format (Dash0SyntheticCheck) with NESTED plugin structure.

Required structure:
- kind: Must be "Dash0SyntheticCheck"
- metadata.name: Check identifier (lowercase, alphanumeric, hyphens)
- spec.enabled: Boolean to enable/disable the check
- spec.plugin.kind: Plugin type (e.g., "http")
- spec.plugin.spec.request: Request configuration (CRITICAL: nested inside plugin.spec!)
- spec.schedule.interval: Check frequency (e.g., "1m", "5m")
- spec.schedule.locations: Array of locations (e.g., ["eu-west-1"])
- spec.schedule.strategy: Execution strategy (e.g., "all_locations")

Example body (simple HTTP check):
{
  "kind": "Dash0SyntheticCheck",
  "metadata": {"name": "api-health-check"},
  "spec": {
    "enabled": true,
    "plugin": {
      "kind": "http",
      "spec": {
        "request": {
          "method": "get",
          "url": "https://api.example.com/health",
          "redirects": "follow"
        }
      }
    },
    "schedule": {
      "interval": "5m",
      "locations": ["eu-west-1"],
      "strategy": "all_locations"
    }
  }
}

Example with headers and retries:
{
  "kind": "Dash0SyntheticCheck",
  "metadata": {"name": "authenticated-api-check"},
  "spec": {
    "enabled": true,
    "plugin": {
      "kind": "http",
      "spec": {
        "request": {
          "method": "get",
          "url": "https://api.example.com/v1/status",
          "redirects": "follow",
          "headers": {
            "Accept": "application/json"
          }
        }
      }
    },
    "schedule": {
      "interval": "1m",
      "locations": ["eu-west-1", "us-east-1"],
      "strategy": "all_locations"
    },
    "retries": {
      "count": 2,
      "delay": "5s"
    }
  }
}

Available locations: eu-west-1, us-east-1, us-west-2, ap-southeast-1, etc.`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The synthetic check configuration in Dash0SyntheticCheck CRD format.",
					"properties": map[string]interface{}{
						"kind": map[string]interface{}{
							"type":        "string",
							"description": "Must be 'Dash0SyntheticCheck'",
							"enum":        []string{"Dash0SyntheticCheck"},
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "Check metadata",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "Check identifier (lowercase, alphanumeric, hyphens)",
								},
							},
							"required": []interface{}{"name"},
						},
						"spec": map[string]interface{}{
							"type":        "object",
							"description": "Check specification",
							"properties": map[string]interface{}{
								"enabled": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether the check is enabled",
								},
								"plugin": map[string]interface{}{
									"type":        "object",
									"description": "Plugin configuration with kind and nested spec.request",
									"properties": map[string]interface{}{
										"kind": map[string]interface{}{
											"type":        "string",
											"description": "Plugin type (e.g., 'http')",
										},
										"spec": map[string]interface{}{
											"type":        "object",
											"description": "Plugin spec containing request configuration",
											"properties": map[string]interface{}{
												"request": map[string]interface{}{
													"type":        "object",
													"description": "HTTP request configuration",
													"properties": map[string]interface{}{
														"method": map[string]interface{}{
															"type":        "string",
															"description": "HTTP method (get, post, put, delete)",
														},
														"url": map[string]interface{}{
															"type":        "string",
															"description": "URL to check",
														},
														"redirects": map[string]interface{}{
															"type":        "string",
															"description": "Redirect handling (follow, reject)",
														},
														"headers": map[string]interface{}{
															"type":        "object",
															"description": "HTTP headers",
														},
													},
												},
											},
										},
									},
								},
								"schedule": map[string]interface{}{
									"type":        "object",
									"description": "Schedule configuration",
									"properties": map[string]interface{}{
										"interval": map[string]interface{}{
											"type":        "string",
											"description": "Check frequency (e.g., '1m', '5m')",
										},
										"locations": map[string]interface{}{
											"type":        "array",
											"description": "Array of check locations (e.g., ['eu-west-1'])",
										},
										"strategy": map[string]interface{}{
											"type":        "string",
											"description": "Execution strategy (e.g., 'all_locations')",
										},
									},
									"required": []interface{}{"interval", "locations"},
								},
								"retries": map[string]interface{}{
									"type":        "object",
									"description": "Retry configuration (optional)",
									"properties": map[string]interface{}{
										"count": map[string]interface{}{
											"type":        "integer",
											"description": "Number of retries",
										},
										"delay": map[string]interface{}{
											"type":        "string",
											"description": "Delay between retries (e.g., '5s')",
										},
									},
								},
							},
							"required": []interface{}{"enabled", "plugin", "schedule"},
						},
					},
					"required": []interface{}{"kind", "metadata", "spec"},
				},
			},
			Required: []string{"body"},
		},
	}
}

// CreateSyntheticCheckHandler handles the dash0_synthetic_checks_create tool.
func (p *Package) CreateSyntheticCheckHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/synthetic-checks", body)
}

// UpdateSyntheticCheck returns the dash0_synthetic_checks_update tool definition.
func (p *Package) UpdateSyntheticCheck() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_synthetic_checks_update",
		Description: `Update an existing synthetic check by its origin or ID.

The body should follow the same Dash0SyntheticCheck CRD format as create:
{
  "kind": "Dash0SyntheticCheck",
  "metadata": {"name": "updated-check"},
  "spec": {
    "enabled": true,
    "plugin": {
      "kind": "http",
      "spec": {
        "request": {
          "method": "get",
          "url": "https://api.example.com/health",
          "redirects": "follow"
        }
      }
    },
    "schedule": {
      "interval": "5m",
      "locations": ["eu-west-1"],
      "strategy": "all_locations"
    }
  }
}`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the synthetic check to update.",
				},
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The updated synthetic check configuration in Dash0SyntheticCheck CRD format with nested plugin.spec.request structure.",
				},
			},
			Required: []string{"origin_or_id", "body"},
		},
	}
}

// UpdateSyntheticCheckHandler handles the dash0_synthetic_checks_update tool.
func (p *Package) UpdateSyntheticCheckHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	path := fmt.Sprintf("/api/synthetic-checks/%s", url.PathEscape(originOrID))
	return p.client.Put(ctx, path, body)
}

// DeleteSyntheticCheck returns the dash0_synthetic_checks_delete tool definition.
func (p *Package) DeleteSyntheticCheck() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_synthetic_checks_delete",
		Description: "Delete a synthetic check by its origin or ID.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the synthetic check to delete.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// DeleteSyntheticCheckHandler handles the dash0_synthetic_checks_delete tool.
func (p *Package) DeleteSyntheticCheckHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf("/api/synthetic-checks/%s", url.PathEscape(originOrID))
	return p.client.Delete(ctx, path)
}

// Register registers all synthetic checks tools with the registry.
func Register(reg *registry.Registry, c *client.Client) {
	p := New(c)
	for _, tool := range p.Tools() {
		handler := p.Handlers()[tool.Name]
		reg.Register(tool, handler)
	}
}
