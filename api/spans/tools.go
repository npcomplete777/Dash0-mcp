package spans

import (
	"context"
	"fmt"
	"math"
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
	basePath = "/api/spans"
)

// Compile-time interface check.
var _ registry.ToolProvider = (*Tools)(nil)

// Tools provides MCP tools for Spans API operations.
type Tools struct {
	client *client.Client
}

// New creates a new Spans tools instance.
func New(c *client.Client) *Tools {
	return &Tools{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Tools) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.PostSpans(),
		p.QuerySpans(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Tools) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_spans_send":  p.PostSpansHandler,
		"dash0_spans_query": p.QuerySpansHandler,
	}
}

// PostSpans returns the dash0_spans_send tool definition.
func (p *Tools) PostSpans() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_spans_send",
		Description: "Send OTLP spans to Dash0. Accepts trace data in OTLP JSON format for distributed tracing analysis.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "OTLP spans in JSON format. Should follow the OpenTelemetry Protocol specification for traces.",
				},
			},
			Required: []string{"body"},
		},
	}
}

// PostSpansHandler handles the dash0_spans_send tool.
func (p *Tools) PostSpansHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, basePath, body)
}

// QuerySpans returns the dash0_spans_query tool definition.
func (p *Tools) QuerySpans() mcp.Tool {
	return mcp.Tool{
		Name: "dash0_spans_query",
		Description: `Query spans from Dash0 with filtering by service, HTTP method, status code, and errors.

Returns spans as a formatted markdown table with duration, status, and key attributes.

Example queries:
- Get spans for a service: {"service_name": "cart"}
- Get error spans: {"error_only": true}
- Get slow POST requests: {"http_method": "POST", "min_duration_ms": 1000}
- Get 5xx errors: {"http_status_code": 500}`,
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
				"http_method": map[string]interface{}{
					"type":        "string",
					"description": "Filter by HTTP method (GET, POST, PUT, DELETE, etc)",
				},
				"http_status_code": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by HTTP response status code (e.g., 200, 404, 500)",
				},
				"error_only": map[string]interface{}{
					"type":        "boolean",
					"description": "Only return error spans (status.code = 2)",
				},
				"min_duration_ms": map[string]interface{}{
					"type":        "number",
					"description": "Filter spans with duration >= this value in milliseconds",
				},
				"span_name": map[string]interface{}{
					"type":        "string",
					"description": "Filter by span name (exact match)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Max spans to return (default: 100, max: 200)",
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

// QuerySpansRequest represents the request body for querying spans.
type QuerySpansRequest struct {
	Dataset    string            `json:"dataset,omitempty"`
	TimeRange  TimeRange         `json:"timeRange"`
	Filter     []AttributeFilter `json:"filter,omitempty"`
	Pagination Pagination        `json:"pagination,omitempty"`
}

// FlatSpan represents a flattened span with calculated fields.
type FlatSpan struct {
	TraceID       string                 `json:"trace_id"`
	SpanID        string                 `json:"span_id"`
	ParentSpanID  string                 `json:"parent_span_id,omitempty"`
	Name          string                 `json:"name"`
	ServiceName   string                 `json:"service_name"`
	SpanKind      string                 `json:"span_kind"`
	DurationMs    float64                `json:"duration_ms"`
	StartTime     string                 `json:"start_time"`
	EndTime       string                 `json:"end_time"`
	StatusCode    int                    `json:"status_code"`
	StatusMessage string                 `json:"status_message,omitempty"`
	K8sPodName    string                 `json:"k8s_pod_name,omitempty"`
	EventCount    int                    `json:"event_count"`
	LinkCount     int                    `json:"link_count"`
	HasChildren   bool                   `json:"has_children"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
}

// QuerySpansHandler handles the dash0_spans_query tool.
func (p *Tools) QuerySpansHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
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

	if httpMethod, ok := args["http_method"].(string); ok {
		httpMethod = strings.TrimSpace(httpMethod)
		if httpMethod != "" {
			filters = append(filters, AttributeFilter{
				Key:      "http.request.method",
				Operator: "is",
				Value:    &AttributeFilterValue{StringValue: &httpMethod},
			})
			filterDescs = append(filterDescs, "method="+httpMethod)
		}
	}

	if statusCode, ok := args["http_status_code"].(float64); ok {
		statusStr := strconv.Itoa(int(statusCode))
		filters = append(filters, AttributeFilter{
			Key:      "http.response.status_code",
			Operator: "is",
			Value:    &AttributeFilterValue{IntValue: &statusStr},
		})
		filterDescs = append(filterDescs, "status="+statusStr)
	}

	if spanName, ok := args["span_name"].(string); ok {
		spanName = strings.TrimSpace(spanName)
		if spanName != "" {
			filters = append(filters, AttributeFilter{
				Key:      "name",
				Operator: "is",
				Value:    &AttributeFilterValue{StringValue: &spanName},
			})
			filterDescs = append(filterDescs, "name="+spanName)
		}
	}

	if errorOnly, ok := args["error_only"].(bool); ok && errorOnly {
		errorCode := "2" // OTLP error status code
		filters = append(filters, AttributeFilter{
			Key:      "status.code",
			Operator: "is",
			Value:    &AttributeFilterValue{IntValue: &errorCode},
		})
		filterDescs = append(filterDescs, "errors_only")
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

	// Set limit
	limit := 100
	if l, ok := args["limit"].(float64); ok {
		if l < 0 {
			return client.ErrorResult(400, "limit must not be negative")
		}
		if l > 0 {
			limit = int(l)
			if limit > 200 {
				limit = 200
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
	req := QuerySpansRequest{
		Dataset: dataset,
		TimeRange: TimeRange{
			From: from.Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		Filter:     filters,
		Pagination: Pagination{Limit: limit},
	}

	// Execute query
	result := p.client.PostWithDataset(ctx, basePath, req, dataset)
	if !result.Success {
		return result
	}

	// Flatten the OTLP response
	flatSpans := flattenSpansResponse(result.Data)

	// Derive HasChildren for each span
	deriveHasChildren(flatSpans)

	// Apply client-side duration filter if specified
	if minDuration, ok := args["min_duration_ms"].(float64); ok && minDuration > 0 {
		var filtered []FlatSpan
		for _, span := range flatSpans {
			if span.DurationMs >= minDuration {
				filtered = append(filtered, span)
			}
		}
		flatSpans = filtered
		filterDescs = append(filterDescs, fmt.Sprintf("min_duration>=%.0fms", minDuration))
	}

	// Build markdown table
	md := formatSpansMarkdown(flatSpans, from, now, filterDescs, limit)

	return &client.ToolResult{
		Success:  true,
		Markdown: md,
		Data: map[string]interface{}{
			"spans": flatSpans,
			"count": len(flatSpans),
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

// deriveHasChildren sets HasChildren on each span by checking if its SpanID
// appears as a ParentSpanID in any other span.
func deriveHasChildren(spans []FlatSpan) {
	parentIDs := make(map[string]struct{}, len(spans))
	for _, s := range spans {
		if s.ParentSpanID != "" {
			parentIDs[s.ParentSpanID] = struct{}{}
		}
	}
	for i := range spans {
		if _, ok := parentIDs[spans[i].SpanID]; ok {
			spans[i].HasChildren = true
		}
	}
}

// computeSpanStats calculates summary statistics for the stats line.
func computeSpanStats(spans []FlatSpan) string {
	if len(spans) == 0 {
		return ""
	}

	// Collect durations and counts
	durations := make([]float64, 0, len(spans))
	var totalDuration float64
	var errorCount int
	serviceCounts := make(map[string]int)
	opCounts := make(map[string]int)

	for _, s := range spans {
		durations = append(durations, s.DurationMs)
		totalDuration += s.DurationMs
		if s.StatusCode == 2 {
			errorCount++
		}
		if s.ServiceName != "" {
			serviceCounts[s.ServiceName]++
		}
		if s.Name != "" {
			opCounts[s.Name]++
		}
	}

	n := len(durations)
	sort.Float64s(durations)
	avg := totalDuration / float64(n)
	maxDur := durations[n-1]

	// P95: index = ceil(0.95 * n) - 1, clamped
	p95Idx := int(math.Ceil(0.95*float64(n))) - 1
	if p95Idx < 0 {
		p95Idx = 0
	}
	if p95Idx >= n {
		p95Idx = n - 1
	}
	p95 := durations[p95Idx]

	// Error rate
	errorRate := float64(errorCount) / float64(n) * 100

	// Top services (up to 5)
	type kv struct {
		Key   string
		Count int
	}
	topServices := topN(serviceCounts, 5)
	topOps := topN(opCounts, 3)

	var parts []string
	parts = append(parts, fmt.Sprintf("Avg: %s", formatter.FormatDuration(avg)))
	parts = append(parts, fmt.Sprintf("P95: %s", formatter.FormatDuration(p95)))
	parts = append(parts, fmt.Sprintf("Max: %s", formatter.FormatDuration(maxDur)))
	parts = append(parts, fmt.Sprintf("Error rate: %.1f%% (%d/%d)", errorRate, errorCount, n))

	if len(topServices) > 0 {
		var svcParts []string
		for _, s := range topServices {
			svcParts = append(svcParts, fmt.Sprintf("%s (%d)", s.Key, s.Count))
		}
		parts = append(parts, "Services: "+strings.Join(svcParts, ", "))
	}

	if len(topOps) > 0 {
		var opParts []string
		for _, o := range topOps {
			opParts = append(opParts, fmt.Sprintf("%s (%d)", o.Key, o.Count))
		}
		parts = append(parts, "Ops: "+strings.Join(opParts, ", "))
	}

	return "> **Stats:** " + strings.Join(parts, " | ")
}

type kvPair struct {
	Key   string
	Count int
}

func topN(counts map[string]int, n int) []kvPair {
	pairs := make([]kvPair, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, kvPair{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Count > pairs[j].Count
	})
	if len(pairs) > n {
		pairs = pairs[:n]
	}
	return pairs
}

// formatSpansMarkdown renders spans as a markdown table.
func formatSpansMarkdown(spans []FlatSpan, from, to time.Time, filterDescs []string, limit int) string {
	// Build summary
	summaryParts := []string{fmt.Sprintf("**Found %d spans**", len(spans))}
	summaryParts = append(summaryParts, fmt.Sprintf("Time: %s → %s", from.Format("15:04:05"), to.Format("15:04:05 2006-01-02")))
	if len(filterDescs) > 0 {
		summaryParts = append(summaryParts, "Filters: "+strings.Join(filterDescs, ", "))
	}
	summary := strings.Join(summaryParts, " | ")

	// Add stats block
	stats := computeSpanStats(spans)
	if stats != "" {
		summary = summary + "\n\n" + stats
	}

	headers := []string{"#", "Service", "Name", "Kind", "Duration", "Status", "HTTP", "Pod", "Children", "Trace ID"}
	var rows [][]string

	for i, span := range spans {
		httpInfo := ""
		if method, ok := span.Attributes["http.request.method"].(string); ok {
			httpInfo = method
			if code, ok := span.Attributes["http.response.status_code"]; ok {
				httpInfo += fmt.Sprintf(" %v", code)
			}
			if route, ok := span.Attributes["http.route"].(string); ok {
				httpInfo += " " + formatter.Truncate(route, 30)
			}
		}

		// Build name with event/link details
		displayName := formatter.Truncate(span.Name, 30)
		var details []string
		if span.EventCount > 0 {
			details = append(details, fmt.Sprintf("%d events", span.EventCount))
		}
		if span.LinkCount > 0 {
			details = append(details, fmt.Sprintf("%d links", span.LinkCount))
		}
		if len(details) > 0 {
			displayName += " [" + strings.Join(details, ", ") + "]"
		}

		childrenStr := "no"
		if span.HasChildren {
			childrenStr = "yes"
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			formatter.Truncate(span.ServiceName, 20),
			displayName,
			span.SpanKind,
			formatter.FormatDuration(span.DurationMs),
			formatter.StatusName(span.StatusCode),
			httpInfo,
			formatter.Truncate(span.K8sPodName, 25),
			childrenStr,
			formatter.Truncate(span.TraceID, 16),
		})
	}

	footer := ""
	if len(spans) >= limit {
		footer = fmt.Sprintf("_Showing %d of %d+ spans (limit reached). Use `limit=%d` for more, or narrow filters._", len(spans), len(spans), limit*2)
	}

	return formatter.Table("Span Query Results", summary, headers, rows, footer)
}

// flattenSpansResponse extracts spans from nested OTLP response structure.
func flattenSpansResponse(data interface{}) []FlatSpan {
	var spans []FlatSpan

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return spans
	}

	resourceSpans, ok := dataMap["resourceSpans"].([]interface{})
	if !ok {
		return spans
	}

	for _, rs := range resourceSpans {
		rsMap, ok := rs.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract service name and K8s pod from resource attributes
		serviceName := extractServiceName(rsMap)
		k8sPodName := extractResourceAttribute(rsMap, "k8s.pod.name")

		scopeSpans, ok := rsMap["scopeSpans"].([]interface{})
		if !ok {
			continue
		}

		for _, ss := range scopeSpans {
			ssMap, ok := ss.(map[string]interface{})
			if !ok {
				continue
			}

			spanList, ok := ssMap["spans"].([]interface{})
			if !ok {
				continue
			}

			for _, s := range spanList {
				spanMap, ok := s.(map[string]interface{})
				if !ok {
					continue
				}

				flat := FlatSpan{
					ServiceName: serviceName,
					K8sPodName:  k8sPodName,
				}

				if name, ok := spanMap["name"].(string); ok {
					flat.Name = name
				}
				if traceID, ok := spanMap["traceId"].(string); ok {
					flat.TraceID = traceID
				}
				if spanID, ok := spanMap["spanId"].(string); ok {
					flat.SpanID = spanID
				}
				if parentSpanID, ok := spanMap["parentSpanId"].(string); ok {
					flat.ParentSpanID = parentSpanID
				}

				// Extract span kind
				if kind, ok := spanMap["kind"].(float64); ok {
					flat.SpanKind = formatter.SpanKindName(int(kind))
				} else {
					flat.SpanKind = "UNSPECIFIED"
				}

				// Count events and links
				if events, ok := spanMap["events"].([]interface{}); ok {
					flat.EventCount = len(events)
				}
				if links, ok := spanMap["links"].([]interface{}); ok {
					flat.LinkCount = len(links)
				}

				// Calculate duration
				if startNanoStr, ok := spanMap["startTimeUnixNano"].(string); ok {
					if endNanoStr, ok := spanMap["endTimeUnixNano"].(string); ok {
						startNano, err1 := strconv.ParseInt(startNanoStr, 10, 64)
						endNano, err2 := strconv.ParseInt(endNanoStr, 10, 64)
						if err1 == nil && err2 == nil {
							flat.DurationMs = float64(endNano-startNano) / 1_000_000
							flat.StartTime = time.Unix(0, startNano).UTC().Format(time.RFC3339Nano)
							flat.EndTime = time.Unix(0, endNano).UTC().Format(time.RFC3339Nano)
						}
					}
				}

				// Extract status
				if status, ok := spanMap["status"].(map[string]interface{}); ok {
					if code, ok := status["code"].(float64); ok {
						flat.StatusCode = int(code)
					}
					if msg, ok := status["message"].(string); ok {
						flat.StatusMessage = msg
					}
				}

				// Extract key attributes
				flat.Attributes = extractSpanAttributes(spanMap)

				spans = append(spans, flat)
			}
		}
	}

	return spans
}

// extractServiceName gets service.name from resource attributes.
func extractServiceName(rsMap map[string]interface{}) string {
	return otlp.ExtractServiceName(rsMap)
}

// extractResourceAttribute extracts a specific attribute from resource attributes.
func extractResourceAttribute(rsMap map[string]interface{}, key string) string {
	resource, ok := rsMap["resource"].(map[string]interface{})
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

// extractSpanAttributes extracts commonly used attributes from a span.
func extractSpanAttributes(spanMap map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	attrs, ok := spanMap["attributes"].([]interface{})
	if !ok {
		return result
	}

	// Keys we want to extract
	interestingKeys := map[string]bool{
		"http.request.method":       true,
		"http.response.status_code": true,
		"http.route":                true,
		"http.url":                  true,
		"http.target":               true,
		"db.system":                 true,
		"db.statement":              true,
		"rpc.method":                true,
		"rpc.service":               true,
		"messaging.system":          true,
		"messaging.operation":       true,
		"error.type":                true,
		"exception.type":            true,
		"exception.message":         true,
	}

	for _, attr := range attrs {
		attrMap, ok := attr.(map[string]interface{})
		if !ok {
			continue
		}

		key, ok := attrMap["key"].(string)
		if !ok || !interestingKeys[key] {
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

// Register registers all spans tools with the registry.
func Register(reg *registry.Registry, c *client.Client) {
	p := New(c)
	for _, tool := range p.Tools() {
		handler := p.Handlers()[tool.Name]
		reg.Register(tool, handler)
	}
}
