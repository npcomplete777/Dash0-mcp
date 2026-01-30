package logs

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

// Package provides MCP tools for Logs API operations.
type Package struct {
	client *client.Client
}

// New creates a new Logs package.
func New(c *client.Client) *Package {
	return &Package{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Package) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.PostLogs(),
		p.QueryLogs(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Package) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_logs_send":  p.PostLogsHandler,
		"dash0_logs_query": p.QueryLogsHandler,
	}
}

// PostLogs returns the dash0_logs_send tool definition.
func (p *Package) PostLogs() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_logs_send",
		Description: "Send OTLP log records to Dash0. Accepts log data in OTLP JSON format for ingestion into the Dash0 observability platform.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "OTLP log records in JSON format. Should follow the OpenTelemetry Protocol specification for logs.",
				},
			},
			Required: []string{"body"},
		},
	}
}

// PostLogsHandler handles the dash0_logs_send tool.
func (p *Package) PostLogsHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/logs", body)
}

// QueryLogs returns the dash0_logs_query tool definition.
func (p *Package) QueryLogs() mcp.Tool {
	return mcp.Tool{
		Name: "dash0_logs_query",
		Description: `Query logs from Dash0 with filtering by service and time range.

Returns flattened log records with severity, body, and trace context.

NOTE: Only service_name filtering works reliably via the API. Severity filtering
is applied client-side after fetching results.

Example queries:
- Get logs for a service: {"service_name": "cart"}
- Get recent logs: {"time_range_minutes": 15}
- Get error logs for a service: {"service_name": "frontend", "min_severity": "ERROR"}`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"service_name": map[string]interface{}{
					"type":        "string",
					"description": "Filter by service name (exact match)",
				},
				"time_range_minutes": map[string]interface{}{
					"type":        "integer",
					"description": "Minutes back to search (default: 60, max: 1440)",
				},
				"min_severity": map[string]interface{}{
					"type":        "string",
					"description": "Minimum severity level: TRACE, DEBUG, INFO, WARN, ERROR, FATAL (applied client-side)",
					"enum":        []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
				},
				"body_contains": map[string]interface{}{
					"type":        "string",
					"description": "Filter logs where body contains this text (case-insensitive, applied client-side)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Max logs to return (default: 100, max: 500)",
				},
			},
		},
	}
}

// AttributeFilter represents a filter condition for log queries.
type AttributeFilter struct {
	Key      string                `json:"key"`
	Operator string                `json:"operator"`
	Value    *AttributeFilterValue `json:"value,omitempty"`
}

// AttributeFilterValue represents the value in a filter condition.
type AttributeFilterValue struct {
	StringValue *string `json:"stringValue,omitempty"`
	IntValue    *string `json:"intValue,omitempty"`
	BoolValue   *bool   `json:"boolValue,omitempty"`
}

// TimeRange represents a time range for queries.
type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Pagination represents pagination settings.
type Pagination struct {
	Limit int `json:"limit,omitempty"`
}

// QueryLogsRequest represents the request body for querying logs.
type QueryLogsRequest struct {
	TimeRange  TimeRange         `json:"timeRange"`
	Filter     []AttributeFilter `json:"filter,omitempty"`
	Pagination Pagination        `json:"pagination,omitempty"`
}

// FlatLog represents a flattened log record.
type FlatLog struct {
	Timestamp      string                 `json:"timestamp"`
	ServiceName    string                 `json:"service_name"`
	SeverityText   string                 `json:"severity_text"`
	SeverityNumber int                    `json:"severity_number"`
	Body           string                 `json:"body"`
	TraceID        string                 `json:"trace_id,omitempty"`
	SpanID         string                 `json:"span_id,omitempty"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
}

// severityOrder defines the ordering of severity levels.
var severityOrder = map[string]int{
	"TRACE": 1,
	"DEBUG": 5,
	"INFO":  9,
	"WARN":  13,
	"ERROR": 17,
	"FATAL": 21,
}

// QueryLogsHandler handles the dash0_logs_query tool.
func (p *Package) QueryLogsHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	// Build filters
	var filters []AttributeFilter

	if serviceName, ok := args["service_name"].(string); ok && serviceName != "" {
		filters = append(filters, AttributeFilter{
			Key:      "service.name",
			Operator: "is",
			Value:    &AttributeFilterValue{StringValue: &serviceName},
		})
	}

	// Calculate time range
	now := time.Now().UTC()
	minutes := 60
	if m, ok := args["time_range_minutes"].(float64); ok && m > 0 {
		minutes = int(m)
		if minutes > 1440 {
			minutes = 1440 // Max 24 hours
		}
	}
	from := now.Add(-time.Duration(minutes) * time.Minute)

	// Set limit (fetch more for client-side filtering)
	limit := 100
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 500 {
			limit = 500
		}
	}

	// Build request
	req := QueryLogsRequest{
		TimeRange: TimeRange{
			From: from.Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		Filter:     filters,
		Pagination: Pagination{Limit: limit * 2}, // Fetch extra for client-side filtering
	}

	// Execute query
	result := p.client.Post(ctx, "/api/logs", req)
	if !result.Success {
		return result
	}

	// Flatten the OTLP response
	flatLogs := flattenLogsResponse(result.Data)

	// Apply client-side severity filter if specified
	if minSeverity, ok := args["min_severity"].(string); ok && minSeverity != "" {
		minLevel := severityOrder[minSeverity]
		var filtered []FlatLog
		for _, log := range flatLogs {
			if log.SeverityNumber >= minLevel {
				filtered = append(filtered, log)
			}
		}
		flatLogs = filtered
	}

	// Apply client-side body contains filter if specified
	if bodyContains, ok := args["body_contains"].(string); ok && bodyContains != "" {
		bodyContainsLower := strings.ToLower(bodyContains)
		var filtered []FlatLog
		for _, log := range flatLogs {
			if strings.Contains(strings.ToLower(log.Body), bodyContainsLower) {
				filtered = append(filtered, log)
			}
		}
		flatLogs = filtered
	}

	// Apply final limit
	if len(flatLogs) > limit {
		flatLogs = flatLogs[:limit]
	}

	return &client.ToolResult{
		Success: true,
		Data: map[string]interface{}{
			"logs":  flatLogs,
			"count": len(flatLogs),
			"query": map[string]interface{}{
				"time_range": map[string]string{
					"from": from.Format(time.RFC3339),
					"to":   now.Format(time.RFC3339),
				},
				"filters": filters,
				"limit":   limit,
			},
		},
	}
}

// flattenLogsResponse extracts logs from nested OTLP response structure.
func flattenLogsResponse(data interface{}) []FlatLog {
	var logs []FlatLog

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return logs
	}

	resourceLogs, ok := dataMap["resourceLogs"].([]interface{})
	if !ok {
		return logs
	}

	for _, rl := range resourceLogs {
		rlMap, ok := rl.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract service name from resource attributes
		serviceName := extractServiceName(rlMap)

		scopeLogs, ok := rlMap["scopeLogs"].([]interface{})
		if !ok {
			continue
		}

		for _, sl := range scopeLogs {
			slMap, ok := sl.(map[string]interface{})
			if !ok {
				continue
			}

			logRecords, ok := slMap["logRecords"].([]interface{})
			if !ok {
				continue
			}

			for _, lr := range logRecords {
				logMap, ok := lr.(map[string]interface{})
				if !ok {
					continue
				}

				flat := FlatLog{
					ServiceName: serviceName,
				}

				// Extract timestamp
				if timeNanoStr, ok := logMap["timeUnixNano"].(string); ok {
					if timeNano, err := strconv.ParseInt(timeNanoStr, 10, 64); err == nil {
						flat.Timestamp = time.Unix(0, timeNano).UTC().Format(time.RFC3339Nano)
					}
				} else if observedTimeStr, ok := logMap["observedTimeUnixNano"].(string); ok {
					if timeNano, err := strconv.ParseInt(observedTimeStr, 10, 64); err == nil {
						flat.Timestamp = time.Unix(0, timeNano).UTC().Format(time.RFC3339Nano)
					}
				}

				// Extract severity
				if sevText, ok := logMap["severityText"].(string); ok {
					flat.SeverityText = sevText
				}
				if sevNum, ok := logMap["severityNumber"].(float64); ok {
					flat.SeverityNumber = int(sevNum)
				}

				// Extract body
				if body, ok := logMap["body"].(map[string]interface{}); ok {
					if strVal, ok := body["stringValue"].(string); ok {
						flat.Body = strVal
					}
				}

				// Extract trace context
				if traceID, ok := logMap["traceId"].(string); ok {
					flat.TraceID = traceID
				}
				if spanID, ok := logMap["spanId"].(string); ok {
					flat.SpanID = spanID
				}

				// Extract key attributes
				flat.Attributes = extractLogAttributes(logMap)

				logs = append(logs, flat)
			}
		}
	}

	return logs
}

// extractServiceName gets service.name from resource attributes.
func extractServiceName(rlMap map[string]interface{}) string {
	resource, ok := rlMap["resource"].(map[string]interface{})
	if !ok {
		return ""
	}

	attrs, ok := resource["attributes"].([]interface{})
	if !ok {
		return ""
	}

	for _, attr := range attrs {
		attrMap, ok := attr.(map[string]interface{})
		if !ok {
			continue
		}
		if key, ok := attrMap["key"].(string); ok && key == "service.name" {
			if value, ok := attrMap["value"].(map[string]interface{}); ok {
				if strVal, ok := value["stringValue"].(string); ok {
					return strVal
				}
			}
		}
	}

	return ""
}

// extractLogAttributes extracts commonly used attributes from a log record.
func extractLogAttributes(logMap map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	attrs, ok := logMap["attributes"].([]interface{})
	if !ok {
		return result
	}

	for _, attr := range attrs {
		attrMap, ok := attr.(map[string]interface{})
		if !ok {
			continue
		}

		key, ok := attrMap["key"].(string)
		if !ok {
			continue
		}

		if value, ok := attrMap["value"].(map[string]interface{}); ok {
			if strVal, ok := value["stringValue"].(string); ok {
				result[key] = strVal
			} else if intVal, ok := value["intValue"].(string); ok {
				if i, err := strconv.ParseInt(intVal, 10, 64); err == nil {
					result[key] = i
				}
			} else if boolVal, ok := value["boolValue"].(bool); ok {
				result[key] = boolVal
			}
		}
	}

	return result
}

// Register registers all logs tools with the registry.
func Register(reg *registry.Registry, c *client.Client) {
	p := New(c)
	for _, tool := range p.Tools() {
		handler := p.Handlers()[tool.Name]
		reg.Register(tool, handler)
	}
}
