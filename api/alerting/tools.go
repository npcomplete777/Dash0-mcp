package alerting

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

// Package provides MCP tools for Alerting API operations.
type Package struct {
	client *client.Client
}

// New creates a new Alerting package.
func New(c *client.Client) *Package {
	return &Package{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Package) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.ListCheckRules(),
		p.GetCheckRule(),
		p.CreateCheckRule(),
		p.UpdateCheckRule(),
		p.DeleteCheckRule(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Package) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_alerting_check_rules_list":   p.ListCheckRulesHandler,
		"dash0_alerting_check_rules_get":    p.GetCheckRuleHandler,
		"dash0_alerting_check_rules_create": p.CreateCheckRuleHandler,
		"dash0_alerting_check_rules_update": p.UpdateCheckRuleHandler,
		"dash0_alerting_check_rules_delete": p.DeleteCheckRuleHandler,
	}
}

// ListCheckRules returns the dash0_alerting_check_rules_list tool definition.
func (p *Package) ListCheckRules() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_alerting_check_rules_list",
		Description: "List all check rules (Prometheus-style alert rules) configured in Dash0.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
}

// ListCheckRulesHandler handles the dash0_alerting_check_rules_list tool.
func (p *Package) ListCheckRulesHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	return p.client.Get(ctx, "/api/alerting/check-rules")
}

// GetCheckRule returns the dash0_alerting_check_rules_get tool definition.
func (p *Package) GetCheckRule() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_alerting_check_rules_get",
		Description: "Get a specific check rule by its origin or ID.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the check rule to retrieve.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// GetCheckRuleHandler handles the dash0_alerting_check_rules_get tool.
func (p *Package) GetCheckRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf("/api/alerting/check-rules/%s", url.PathEscape(originOrID))
	return p.client.Get(ctx, path)
}

// CreateCheckRule returns the dash0_alerting_check_rules_create tool definition.
func (p *Package) CreateCheckRule() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_alerting_check_rules_create",
		Description: `Create a new check rule (Prometheus-style alert rule) in Dash0.

IMPORTANT: Check rules use plain JSON format (NOT Kubernetes CRD format).

Required fields:
- name: The alert rule name (NOT "alert")
- expression: PromQL expression (NOT "expr")
- interval: Evaluation frequency (e.g., "1m", "30s")
- for: Duration threshold before firing (e.g., "5m", "1m")

Optional fields:
- labels: Key-value pairs for alert routing (e.g., {"severity": "critical"})
- annotations: Key-value pairs for alert details (e.g., {"summary": "...", "description": "..."})
- keepFiringFor: How long to keep firing after condition resolves (default: "0s")

Example body:
{
  "name": "HighErrorRate",
  "expression": "rate(http_requests_total{status=~\"5..\"}[5m]) > 0.05",
  "interval": "1m",
  "for": "5m",
  "labels": {"severity": "critical", "team": "platform"},
  "annotations": {
    "summary": "High error rate detected",
    "description": "Error rate exceeds 5% for 5 minutes"
  }
}`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The check rule configuration with name, expression, interval, for, labels, and annotations.",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "The alert rule name",
						},
						"expression": map[string]interface{}{
							"type":        "string",
							"description": "PromQL expression to evaluate",
						},
						"interval": map[string]interface{}{
							"type":        "string",
							"description": "Evaluation frequency (e.g., '1m', '30s')",
						},
						"for": map[string]interface{}{
							"type":        "string",
							"description": "Duration threshold before firing (e.g., '5m')",
						},
						"labels": map[string]interface{}{
							"type":        "object",
							"description": "Key-value pairs for alert routing",
						},
						"annotations": map[string]interface{}{
							"type":        "object",
							"description": "Key-value pairs for alert details (summary, description)",
						},
						"keepFiringFor": map[string]interface{}{
							"type":        "string",
							"description": "How long to keep firing after condition resolves",
						},
					},
					"required": []interface{}{"name", "expression", "interval", "for"},
				},
			},
			Required: []string{"body"},
		},
	}
}

// CreateCheckRuleHandler handles the dash0_alerting_check_rules_create tool.
func (p *Package) CreateCheckRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/alerting/check-rules", body)
}

// UpdateCheckRule returns the dash0_alerting_check_rules_update tool definition.
func (p *Package) UpdateCheckRule() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_alerting_check_rules_update",
		Description: `Update an existing check rule by its origin or ID.

The body should follow the same format as create:
{
  "name": "UpdatedAlertName",
  "expression": "rate(http_requests_total{status=~\"5..\"}[5m]) > 0.1",
  "interval": "1m",
  "for": "5m",
  "labels": {"severity": "warning"},
  "annotations": {"summary": "Updated alert"}
}`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the check rule to update.",
				},
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The updated check rule configuration with name, expression, interval, for, labels, and annotations.",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "The alert rule name",
						},
						"expression": map[string]interface{}{
							"type":        "string",
							"description": "PromQL expression to evaluate",
						},
						"interval": map[string]interface{}{
							"type":        "string",
							"description": "Evaluation frequency (e.g., '1m', '30s')",
						},
						"for": map[string]interface{}{
							"type":        "string",
							"description": "Duration threshold before firing (e.g., '5m')",
						},
						"labels": map[string]interface{}{
							"type":        "object",
							"description": "Key-value pairs for alert routing",
						},
						"annotations": map[string]interface{}{
							"type":        "object",
							"description": "Key-value pairs for alert details",
						},
					},
					"required": []interface{}{"name", "expression", "interval", "for"},
				},
			},
			Required: []string{"origin_or_id", "body"},
		},
	}
}

// UpdateCheckRuleHandler handles the dash0_alerting_check_rules_update tool.
func (p *Package) UpdateCheckRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	path := fmt.Sprintf("/api/alerting/check-rules/%s", url.PathEscape(originOrID))
	return p.client.Put(ctx, path, body)
}

// DeleteCheckRule returns the dash0_alerting_check_rules_delete tool definition.
func (p *Package) DeleteCheckRule() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_alerting_check_rules_delete",
		Description: "Delete a check rule by its origin or ID.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the check rule to delete.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// DeleteCheckRuleHandler handles the dash0_alerting_check_rules_delete tool.
func (p *Package) DeleteCheckRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf("/api/alerting/check-rules/%s", url.PathEscape(originOrID))
	return p.client.Delete(ctx, path)
}
