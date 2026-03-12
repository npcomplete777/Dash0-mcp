package logs

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/npcomplete777/dash0-mcp/internal/client"
	"github.com/npcomplete777/dash0-mcp/internal/formatter"
	"github.com/npcomplete777/dash0-mcp/internal/otlp"
	"github.com/npcomplete777/dash0-mcp/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

const (
	basePath = "/api/logs"
)

// Compile-time interface check.
var _ registry.ToolProvider = (*Tools)(nil)

// Tools provides MCP tools for Logs API operations.
type Tools struct {
	client *client.Client
}

// New creates a new Logs tools instance.
func New(c *client.Client) *Tools {
	return &Tools{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Tools) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.PostLogs(),
		p.QueryLogs(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Tools) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_logs_send":  p.PostLogsHandler,
		"dash0_logs_query": p.QueryLogsHandler,
	}
}

// PostLogs returns the dash0_logs_send tool definition.
func (p *Tools) PostLogs() mcp.Tool {
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
func (p *Tools) PostLogsHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, basePath, body)
}

// QueryLogs returns the dash0_logs_query tool definition.
func (p *Tools) QueryLogs() mcp.Tool {
	return mcp.Tool{
		Name: "dash0_logs_query",
		Description: `Query logs from Dash0 with filtering by service and time range.

Returns logs as a formatted markdown table with severity, body, and trace context.

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
				"dataset": map[string]interface{}{
					"type":        "string",
					"description": "Dash0 dataset to query (e.g., 'astronomy-demo'). If omitted, uses the globally configured dataset or 'default'.",
				},
			},
		},
	}
}

// Type aliases for shared OTLP types.
type AttributeFilter = otlp.AttributeFilter
type AttributeFilterValue = otlp.AttributeFilterValue
type TimeRange = otlp.TimeRange
type Pagination = otlp.Pagination

// QueryLogsRequest represents the request body for querying logs.
type QueryLogsRequest struct {
	Dataset    string            `json:"dataset,omitempty"`
	TimeRange  TimeRange         `json:"timeRange"`
	Filter     []AttributeFilter `json:"filter,omitempty"`
	Pagination Pagination        `json:"pagination,omitempty"`
}

// FlatLog represents a flattened log record.
type FlatLog struct {
	Timestamp        string                 `json:"timestamp"`
	ServiceName      string                 `json:"service_name"`
	SeverityText     string                 `json:"severity_text"`
	SeverityNumber   int                    `json:"severity_number"`
	Body             string                 `json:"body"`
	TraceID          string                 `json:"trace_id,omitempty"`
	SpanID           string                 `json:"span_id,omitempty"`
	K8sNamespace     string                 `json:"k8s_namespace,omitempty"`
	K8sPodName       string                 `json:"k8s_pod_name,omitempty"`
	K8sContainerName string                 `json:"k8s_container_name,omitempty"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
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
func (p *Tools) QueryLogsHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	// Build filters
	var filters []AttributeFilter
	var filterDescs []string

	if serviceName, ok := args["service_name"].(string); ok {
		serviceName = strings.TrimSpace(serviceName)
		if serviceName != "" {
			filters = append(filters, AttributeFilter{
				Key:      "service.name",
				Operator: "is",
				Value:    &AttributeFilterValue{StringValue: &serviceName},
			})
			filterDescs = append(filterDescs, "service="+serviceName)
		}
	}

	// Calculate time range
	now := time.Now().UTC()
	minutes := 60
	if m, ok := args["time_range_minutes"].(float64); ok {
		if m < 0 {
			return client.ErrorResult(400, "time_range_minutes must not be negative")
		}
		if m > 0 {
			minutes = int(m)
			if minutes > 1440 {
				minutes = 1440 // Max 24 hours
			}
		}
	}
	from := now.Add(-time.Duration(minutes) * time.Minute)

	// Set limit (fetch more for client-side filtering)
	limit := 100
	if l, ok := args["limit"].(float64); ok {
		if l < 0 {
			return client.ErrorResult(400, "limit must not be negative")
		}
		if l > 0 {
			limit = int(l)
			if limit > 500 {
				limit = 500
			}
		}
	}

	// Resolve dataset: per-tool param overrides global config
	dataset := ""
	if ds, ok := args["dataset"].(string); ok && ds != "" {
		dataset = ds
	} else {
		dataset = p.client.GetDataset()
	}

	// Build request
	req := QueryLogsRequest{
		Dataset: dataset,
		TimeRange: TimeRange{
			From: from.Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		Filter:     filters,
		Pagination: Pagination{Limit: limit * 2}, // Fetch extra for client-side filtering
	}

	// Execute query
	result := p.client.PostWithDataset(ctx, basePath, req, dataset)
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
		filterDescs = append(filterDescs, "severity>="+minSeverity)
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
		filterDescs = append(filterDescs, "body~"+bodyContains)
	}

	// Apply final limit
	if len(flatLogs) > limit {
		flatLogs = flatLogs[:limit]
	}

	// Build markdown table
	md := formatLogsMarkdown(flatLogs, from, now, filterDescs, limit)

	return &client.ToolResult{
		Success:  true,
		Markdown: md,
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

// formatLogsMarkdown renders logs as a markdown table with summary statistics.
func formatLogsMarkdown(logs []FlatLog, from, to time.Time, filterDescs []string, limit int) string {
	summaryParts := []string{fmt.Sprintf("**Found %d logs**", len(logs))}
	summaryParts = append(summaryParts, fmt.Sprintf("Time: %s → %s", from.Format("15:04:05"), to.Format("15:04:05 2006-01-02")))
	if len(filterDescs) > 0 {
		summaryParts = append(summaryParts, "Filters: "+strings.Join(filterDescs, ", "))
	}
	summary := strings.Join(summaryParts, " | ")

	// Build statistics
	if len(logs) > 0 {
		summary += "\n\n" + buildLogStats(logs)
	}

	headers := []string{"#", "Timestamp", "Service", "Severity", "Body", "Pod", "Trace ID"}
	var rows [][]string

	for i, log := range logs {
		ts := log.Timestamp
		if t, err := time.Parse(time.RFC3339Nano, log.Timestamp); err == nil {
			ts = t.Format("15:04:05.000")
		}

		pod := log.K8sPodName
		if pod == "" && log.K8sContainerName != "" {
			pod = log.K8sContainerName
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			ts,
			formatter.Truncate(log.ServiceName, 20),
			log.SeverityText,
			formatter.Truncate(log.Body, 60),
			formatter.Truncate(pod, 25),
			formatter.Truncate(log.TraceID, 16),
		})
	}

	footer := ""
	if len(logs) >= limit {
		footer = fmt.Sprintf("_Showing %d of %d+ logs (limit reached). Use limit=500 for more, or add filters._", len(logs), len(logs))
	}

	return formatter.Table("Log Query Results", summary, headers, rows, footer)
}

// buildLogStats computes summary statistics for a set of logs.
func buildLogStats(logs []FlatLog) string {
	// Severity distribution
	sevCounts := make(map[string]int)
	// Service counts
	svcCounts := make(map[string]int)
	// Pod counts
	podCounts := make(map[string]int)
	// Trace ID presence
	withTrace := 0

	for _, log := range logs {
		sev := log.SeverityText
		if sev == "" {
			sev = "UNSET"
		}
		sevCounts[sev]++

		if log.ServiceName != "" {
			svcCounts[log.ServiceName]++
		}

		pod := log.K8sPodName
		if pod != "" {
			podCounts[pod]++
		}

		if log.TraceID != "" {
			withTrace++
		}
	}

	var statParts []string

	// Severity distribution in priority order
	sevOrder := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE", "UNSET"}
	var sevParts []string
	for _, sev := range sevOrder {
		if count, ok := sevCounts[sev]; ok && count > 0 {
			sevParts = append(sevParts, fmt.Sprintf("%s: %d", sev, count))
		}
	}
	if len(sevParts) > 0 {
		statParts = append(statParts, strings.Join(sevParts, " | "))
	}

	// Top services (up to 5)
	if len(svcCounts) > 0 {
		type kv struct {
			key   string
			count int
		}
		sorted := make([]kv, 0, len(svcCounts))
		for k, v := range svcCounts {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})
		maxSvc := 5
		if len(sorted) < maxSvc {
			maxSvc = len(sorted)
		}
		var svcParts []string
		for _, s := range sorted[:maxSvc] {
			svcParts = append(svcParts, fmt.Sprintf("%s (%d)", s.key, s.count))
		}
		statParts = append(statParts, "Services: "+strings.Join(svcParts, ", "))
	}

	// Trace percentage
	tracePercent := withTrace * 100 / len(logs)
	statParts = append(statParts, fmt.Sprintf("With traces: %d%%", tracePercent))

	// Top pods (up to 3), only if any exist
	if len(podCounts) > 0 {
		type kv struct {
			key   string
			count int
		}
		sorted := make([]kv, 0, len(podCounts))
		for k, v := range podCounts {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})
		maxPod := 3
		if len(sorted) < maxPod {
			maxPod = len(sorted)
		}
		var podParts []string
		for _, p := range sorted[:maxPod] {
			podParts = append(podParts, fmt.Sprintf("%s (%d)", p.key, p.count))
		}
		statParts = append(statParts, "Pods: "+strings.Join(podParts, ", "))
	}

	return "> **Stats:** " + strings.Join(statParts, " | ")
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

		// Extract service name and K8s info from resource attributes
		serviceName := extractServiceName(rlMap)
		k8sNamespace := extractResourceAttribute(rlMap, "k8s.namespace.name")
		k8sPodName := extractResourceAttribute(rlMap, "k8s.pod.name")
		k8sContainerName := extractResourceAttribute(rlMap, "k8s.container.name")

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
					ServiceName:      serviceName,
					K8sNamespace:     k8sNamespace,
					K8sPodName:       k8sPodName,
					K8sContainerName: k8sContainerName,
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
	return otlp.ExtractServiceName(rlMap)
}

// extractResourceAttribute extracts a specific attribute from resource attributes.
func extractResourceAttribute(rlMap map[string]interface{}, key string) string {
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
		if attrMap["key"] == key {
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
