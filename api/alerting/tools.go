package alerting

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/npcomplete777/dash0-mcp/internal/client"
	"github.com/npcomplete777/dash0-mcp/internal/formatter"
	"github.com/npcomplete777/dash0-mcp/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

const (
	basePath   = "/api/alerting/check-rules"
	alertsPath = "/api/alerting/alerts"
)

// Compile-time interface check.
var _ registry.ToolProvider = (*Tools)(nil)

// Tools provides MCP tools for Alerting API operations.
type Tools struct {
	client *client.Client
}

// New creates a new Alerting tools instance.
func New(c *client.Client) *Tools {
	return &Tools{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Tools) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.ListCheckRules(),
		p.GetCheckRule(),
		p.CreateCheckRule(),
		p.UpdateCheckRule(),
		p.DeleteCheckRule(),
		p.ActiveAlerts(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Tools) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_alerting_check_rules_list":   p.ListCheckRulesHandler,
		"dash0_alerting_check_rules_get":    p.GetCheckRuleHandler,
		"dash0_alerting_check_rules_create": p.CreateCheckRuleHandler,
		"dash0_alerting_check_rules_update": p.UpdateCheckRuleHandler,
		"dash0_alerting_check_rules_delete": p.DeleteCheckRuleHandler,
		"dash0_alerting_active_alerts":      p.ActiveAlertsHandler,
	}
}

// ListCheckRules returns the dash0_alerting_check_rules_list tool definition.
func (p *Tools) ListCheckRules() mcp.Tool {
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
func (p *Tools) ListCheckRulesHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	result := p.client.Get(ctx, basePath)
	if result.Success {
		result.Markdown = formatCheckRulesList(result.Data)
	}
	return result
}

// formatCheckRulesList formats check rules as a markdown table.
func formatCheckRulesList(data interface{}) string {
	items := extractItems(data)
	if len(items) == 0 {
		return "## Check Rules\n\nNo check rules found.\n"
	}

	headers := []string{"#", "Name", "Expression", "Interval", "For", "Severity", "Origin"}
	var rows [][]string

	for i, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name := extractField(m, "name")
		expr := extractField(m, "expression")
		interval := extractField(m, "interval")
		forDur := extractField(m, "for")
		severity := ""
		if labels, ok := m["labels"].(map[string]interface{}); ok {
			severity = fmt.Sprintf("%v", labels["severity"])
			if severity == "<nil>" {
				severity = ""
			}
		}
		origin := extractNestedField(m, "metadata", "origin")
		if origin == "" {
			origin = extractField(m, "origin")
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			formatter.Truncate(name, 30),
			formatter.Truncate(expr, 50),
			interval,
			forDur,
			severity,
			formatter.Truncate(origin, 30),
		})
	}

	summary := fmt.Sprintf("**Found %d check rules**", len(rows))
	return formatter.Table("Check Rules", summary, headers, rows, "")
}

// extractItems tries to get a slice of items from various response shapes.
func extractItems(data interface{}) []interface{} {
	if data == nil {
		return nil
	}
	if arr, ok := data.([]interface{}); ok {
		return arr
	}
	if m, ok := data.(map[string]interface{}); ok {
		for _, key := range []string{"items", "data", "results", "rules"} {
			if arr, ok := m[key].([]interface{}); ok {
				return arr
			}
		}
	}
	return nil
}

func extractField(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func extractNestedField(m map[string]interface{}, keys ...string) string {
	current := m
	for i, key := range keys {
		if i == len(keys)-1 {
			return extractField(current, key)
		}
		next, ok := current[key].(map[string]interface{})
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

// GetCheckRule returns the dash0_alerting_check_rules_get tool definition.
func (p *Tools) GetCheckRule() mcp.Tool {
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
func (p *Tools) GetCheckRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf(basePath+"/%s", url.PathEscape(originOrID))
	return p.client.Get(ctx, path)
}

// CreateCheckRule returns the dash0_alerting_check_rules_create tool definition.
func (p *Tools) CreateCheckRule() mcp.Tool {
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
func (p *Tools) CreateCheckRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, basePath, body)
}

// UpdateCheckRule returns the dash0_alerting_check_rules_update tool definition.
func (p *Tools) UpdateCheckRule() mcp.Tool {
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
func (p *Tools) UpdateCheckRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	path := fmt.Sprintf(basePath+"/%s", url.PathEscape(originOrID))
	return p.client.Put(ctx, path, body)
}

// DeleteCheckRule returns the dash0_alerting_check_rules_delete tool definition.
func (p *Tools) DeleteCheckRule() mcp.Tool {
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
func (p *Tools) DeleteCheckRuleHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf(basePath+"/%s", url.PathEscape(originOrID))
	return p.client.Delete(ctx, path)
}

// ActiveAlerts returns the dash0_alerting_active_alerts tool definition.
func (p *Tools) ActiveAlerts() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_alerting_active_alerts",
		Description: "List currently firing and pending alerts in Dash0. Shows active alert instances with severity, duration, and labels - unlike check_rules_list which shows rule definitions.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Filter alerts by state. Use 'firing' for actively firing alerts, 'pending' for alerts waiting to fire, or 'all' for both.",
					"enum":        []interface{}{"firing", "pending", "all"},
					"default":     "all",
				},
			},
		},
	}
}

// ActiveAlertsHandler handles the dash0_alerting_active_alerts tool.
func (p *Tools) ActiveAlertsHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	path := alertsPath

	state, _ := args["state"].(string)
	if state != "" && state != "all" {
		path = fmt.Sprintf("%s?state=%s", alertsPath, url.QueryEscape(state))
	}

	result := p.client.Get(ctx, path)
	if result.Success {
		result.Markdown = formatActiveAlerts(result.Data, state)
	}
	return result
}

// formatActiveAlerts formats active alert instances as a markdown table.
func formatActiveAlerts(data interface{}, stateFilter string) string {
	items := extractItems(data)
	if len(items) == 0 {
		return "## Active Alerts\n\nNo active alerts found.\n"
	}

	headers := []string{"#", "Name", "State", "Severity", "Since", "Duration", "Labels"}
	var rows [][]string
	var firingCount, pendingCount int

	for i, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name := extractField(m, "name")
		if name == "" {
			name = extractNestedField(m, "labels", "alertname")
		}

		state := extractField(m, "state")
		switch state {
		case "firing":
			firingCount++
		case "pending":
			pendingCount++
		}

		severity := extractNestedField(m, "labels", "severity")
		if severity == "" {
			severity = extractField(m, "severity")
		}

		activeAt := extractField(m, "activeAt")
		if activeAt == "" {
			activeAt = extractField(m, "startsAt")
		}

		duration := ""
		if activeAt != "" {
			if t, err := time.Parse(time.RFC3339, activeAt); err == nil {
				dur := time.Since(t)
				duration = formatAlertDuration(dur)
			}
			// Shorten the timestamp for display
			if len(activeAt) > 19 {
				activeAt = activeAt[:19]
			}
		}

		// Collect labels, excluding alertname and severity (already shown).
		labelStr := ""
		if labels, ok := m["labels"].(map[string]interface{}); ok {
			var parts []string
			for k, v := range labels {
				if k == "alertname" || k == "severity" {
					continue
				}
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
			labelStr = formatter.Truncate(strings.Join(parts, ", "), 50)
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			formatter.Truncate(name, 30),
			state,
			severity,
			activeAt,
			duration,
			labelStr,
		})
	}

	total := firingCount + pendingCount
	if total == 0 {
		total = len(rows)
	}
	summary := fmt.Sprintf("**%d active alerts** (%d firing, %d pending)", total, firingCount, pendingCount)
	return formatter.Table("Active Alerts", summary, headers, rows, "")
}

// formatAlertDuration formats a duration into a human-readable string for alerts.
func formatAlertDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m > 0 {
			return fmt.Sprintf("%dh%dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	if h > 0 {
		return fmt.Sprintf("%dd%dh", days, h)
	}
	return fmt.Sprintf("%dd", days)
}

// Register registers all alerting tools with the registry.
func Register(reg *registry.Registry, c *client.Client) {
	p := New(c)
	for _, tool := range p.Tools() {
		handler := p.Handlers()[tool.Name]
		reg.Register(tool, handler)
	}
}
