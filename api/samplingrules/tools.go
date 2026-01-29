package samplingrules

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

// Package provides MCP tools for Sampling Rules API operations.
type Package struct {
	client *client.Client
}

// New creates a new Sampling Rules package.
func New(c *client.Client) *Package {
	return &Package{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Package) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.ListSamplingRules(),
		p.GetSamplingRule(),
		p.CreateSamplingRule(),
		p.UpdateSamplingRule(),
		p.DeleteSamplingRule(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Package) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_sampling_rules_list":   p.ListSamplingRulesHandler,
		"dash0_sampling_rules_get":    p.GetSamplingRuleHandler,
		"dash0_sampling_rules_create": p.CreateSamplingRuleHandler,
		"dash0_sampling_rules_update": p.UpdateSamplingRuleHandler,
		"dash0_sampling_rules_delete": p.DeleteSamplingRuleHandler,
	}
}

// ListSamplingRules returns the dash0_sampling_rules_list tool definition.
func (p *Package) ListSamplingRules() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_sampling_rules_list",
		Description: "List all sampling rules in Dash0. Sampling rules control which traces and logs are ingested, helping manage data volume and costs.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
}

// ListSamplingRulesHandler handles the dash0_sampling_rules_list tool.
func (p *Package) ListSamplingRulesHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	return p.client.Get(ctx, "/api/sampling-rules")
}

// GetSamplingRule returns the dash0_sampling_rules_get tool definition.
func (p *Package) GetSamplingRule() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_sampling_rules_get",
		Description: "Get a specific sampling rule by its origin or ID, including matching conditions and sample rates.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the sampling rule to retrieve.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// GetSamplingRuleHandler handles the dash0_sampling_rules_get tool.
func (p *Package) GetSamplingRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf("/api/sampling-rules/%s", url.PathEscape(originOrID))
	return p.client.Get(ctx, path)
}

// CreateSamplingRule returns the dash0_sampling_rules_create tool definition.
func (p *Package) CreateSamplingRule() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_sampling_rules_create",
		Description: `Create a new sampling rule in Dash0 to control data ingestion rates for specific services, operations, or attributes.

IMPORTANT: Sampling rules use Kubernetes CRD format (Dash0Sampling) with union-type conditions.

Required structure:
- kind: Must be "Dash0Sampling"
- metadata.name: Rule identifier (lowercase, alphanumeric, hyphens)
- spec.enabled: Boolean to enable/disable the rule
- spec.conditions.kind: Condition type ("error", "probabilistic", "ottl", or "and")
- spec.conditions.spec: Condition-specific configuration

Condition types:

1. Error condition (capture all errors):
{
  "kind": "Dash0Sampling",
  "metadata": {"name": "capture-all-errors"},
  "spec": {
    "enabled": true,
    "conditions": {
      "kind": "error",
      "spec": {}
    }
  }
}

2. Probabilistic sampling (e.g., 10% of traces):
{
  "kind": "Dash0Sampling",
  "metadata": {"name": "sample-10-percent"},
  "spec": {
    "enabled": true,
    "conditions": {
      "kind": "probabilistic",
      "spec": {"rate": 0.1}
    }
  }
}
NOTE: Use "rate" (0.0-1.0), NOT "probability" or "percentage"!

3. OTTL expression (OpenTelemetry Transformation Language):
{
  "kind": "Dash0Sampling",
  "metadata": {"name": "slow-requests"},
  "spec": {
    "enabled": true,
    "conditions": {
      "kind": "ottl",
      "spec": {"ottl": "duration > 1000"}
    }
  }
}

4. AND condition (combine multiple conditions):
{
  "kind": "Dash0Sampling",
  "metadata": {"name": "sampled-errors"},
  "spec": {
    "enabled": true,
    "conditions": {
      "kind": "and",
      "spec": {
        "conditions": [
          {"kind": "error", "spec": {}},
          {"kind": "probabilistic", "spec": {"rate": 0.5}}
        ]
      }
    }
  }
}`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The sampling rule configuration in Dash0Sampling CRD format.",
					"properties": map[string]interface{}{
						"kind": map[string]interface{}{
							"type":        "string",
							"description": "Must be 'Dash0Sampling'",
							"enum":        []string{"Dash0Sampling"},
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "Rule metadata",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "Rule identifier (lowercase, alphanumeric, hyphens)",
								},
							},
							"required": []interface{}{"name"},
						},
						"spec": map[string]interface{}{
							"type":        "object",
							"description": "Rule specification",
							"properties": map[string]interface{}{
								"enabled": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether the rule is enabled",
								},
								"conditions": map[string]interface{}{
									"type":        "object",
									"description": "Sampling conditions",
									"properties": map[string]interface{}{
										"kind": map[string]interface{}{
											"type":        "string",
											"description": "Condition type: 'error', 'probabilistic', 'ottl', or 'and'",
											"enum":        []string{"error", "probabilistic", "ottl", "and"},
										},
										"spec": map[string]interface{}{
											"type":        "object",
											"description": "Condition-specific configuration. For error: {}. For probabilistic: {\"rate\": 0.1}. For ottl: {\"ottl\": \"expression\"}. For and: {\"conditions\": [...]}",
										},
									},
									"required": []interface{}{"kind", "spec"},
								},
							},
							"required": []interface{}{"enabled", "conditions"},
						},
					},
					"required": []interface{}{"kind", "metadata", "spec"},
				},
			},
			Required: []string{"body"},
		},
	}
}

// CreateSamplingRuleHandler handles the dash0_sampling_rules_create tool.
func (p *Package) CreateSamplingRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/sampling-rules", body)
}

// UpdateSamplingRule returns the dash0_sampling_rules_update tool definition.
func (p *Package) UpdateSamplingRule() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_sampling_rules_update",
		Description: `Update an existing sampling rule by its origin or ID.

The body should follow the same Dash0Sampling CRD format as create:
{
  "kind": "Dash0Sampling",
  "metadata": {"name": "updated-rule"},
  "spec": {
    "enabled": true,
    "conditions": {
      "kind": "probabilistic",
      "spec": {"rate": 0.2}
    }
  }
}

Remember: Use "rate" (0.0-1.0) for probabilistic sampling, NOT "probability"!`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the sampling rule to update.",
				},
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The updated sampling rule configuration in Dash0Sampling CRD format with conditions.kind and conditions.spec.",
				},
			},
			Required: []string{"origin_or_id", "body"},
		},
	}
}

// UpdateSamplingRuleHandler handles the dash0_sampling_rules_update tool.
func (p *Package) UpdateSamplingRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	path := fmt.Sprintf("/api/sampling-rules/%s", url.PathEscape(originOrID))
	return p.client.Put(ctx, path, body)
}

// DeleteSamplingRule returns the dash0_sampling_rules_delete tool definition.
func (p *Package) DeleteSamplingRule() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_sampling_rules_delete",
		Description: "Delete a sampling rule by its origin or ID.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the sampling rule to delete.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// DeleteSamplingRuleHandler handles the dash0_sampling_rules_delete tool.
func (p *Package) DeleteSamplingRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf("/api/sampling-rules/%s", url.PathEscape(originOrID))
	return p.client.Delete(ctx, path)
}
