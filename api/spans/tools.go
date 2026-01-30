package spans

import (
	"context"
	"strconv"
	"time"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

// Package provides MCP tools for Spans API operations.
type Package struct {
	client *client.Client
}

// New creates a new Spans package.
func New(c *client.Client) *Package {
	return &Package{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Package) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.PostSpans(),
		p.QuerySpans(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Package) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_spans_send":  p.PostSpansHandler,
		"dash0_spans_query": p.QuerySpansHandler,
	}
}

// PostSpans returns the dash0_spans_send tool definition.
func (p *Package) PostSpans() mcp.Tool {
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
func (p *Package) PostSpansHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, "/api/spans", body)
}

// QuerySpans returns the dash0_spans_query tool definition.
func (p *Package) QuerySpans() mcp.Tool {
	return mcp.Tool{
		Name: "dash0_spans_query",
		Description: `Query spans from Dash0 with filtering by service, HTTP method, status code, and errors.

Returns flattened span data with calculated duration. Filters use exact match by default.

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
			},
		},
	}
}

// AttributeFilter represents a filter condition for span queries.
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

// QuerySpansRequest represents the request body for querying spans.
type QuerySpansRequest struct {
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
	DurationMs    float64                `json:"duration_ms"`
	StartTime     string                 `json:"start_time"`
	EndTime       string                 `json:"end_time"`
	StatusCode    int                    `json:"status_code"`
	StatusMessage string                 `json:"status_message,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
}

// QuerySpansHandler handles the dash0_spans_query tool.
func (p *Package) QuerySpansHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	// Build filters
	var filters []AttributeFilter

	if serviceName, ok := args["service_name"].(string); ok && serviceName != "" {
		filters = append(filters, AttributeFilter{
			Key:      "service.name",
			Operator: "is",
			Value:    &AttributeFilterValue{StringValue: &serviceName},
		})
	}

	if httpMethod, ok := args["http_method"].(string); ok && httpMethod != "" {
		filters = append(filters, AttributeFilter{
			Key:      "http.request.method",
			Operator: "is",
			Value:    &AttributeFilterValue{StringValue: &httpMethod},
		})
	}

	if statusCode, ok := args["http_status_code"].(float64); ok {
		statusStr := strconv.Itoa(int(statusCode))
		filters = append(filters, AttributeFilter{
			Key:      "http.response.status_code",
			Operator: "is",
			Value:    &AttributeFilterValue{IntValue: &statusStr},
		})
	}

	if spanName, ok := args["span_name"].(string); ok && spanName != "" {
		filters = append(filters, AttributeFilter{
			Key:      "name",
			Operator: "is",
			Value:    &AttributeFilterValue{StringValue: &spanName},
		})
	}

	if errorOnly, ok := args["error_only"].(bool); ok && errorOnly {
		errorCode := "2" // OTLP error status code
		filters = append(filters, AttributeFilter{
			Key:      "status.code",
			Operator: "is",
			Value:    &AttributeFilterValue{IntValue: &errorCode},
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

	// Set limit
	limit := 100
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 200 {
			limit = 200
		}
	}

	// Build request
	req := QuerySpansRequest{
		TimeRange: TimeRange{
			From: from.Format(time.RFC3339),
			To:   now.Format(time.RFC3339),
		},
		Filter:     filters,
		Pagination: Pagination{Limit: limit},
	}

	// Execute query
	result := p.client.Post(ctx, "/api/spans", req)
	if !result.Success {
		return result
	}

	// Flatten the OTLP response
	flatSpans := flattenSpansResponse(result.Data)

	// Apply client-side duration filter if specified
	if minDuration, ok := args["min_duration_ms"].(float64); ok && minDuration > 0 {
		var filtered []FlatSpan
		for _, span := range flatSpans {
			if span.DurationMs >= minDuration {
				filtered = append(filtered, span)
			}
		}
		flatSpans = filtered
	}

	return &client.ToolResult{
		Success: true,
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

		// Extract service name from resource attributes
		serviceName := extractServiceName(rsMap)

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

// extractSpanAttributes extracts commonly used attributes from a span.
func extractSpanAttributes(spanMap map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	attrs, ok := spanMap["attributes"].([]interface{})
	if !ok {
		return result
	}

	// Keys we want to extract
	interestingKeys := map[string]bool{
		"http.request.method":      true,
		"http.response.status_code": true,
		"http.route":               true,
		"http.url":                 true,
		"http.target":              true,
		"db.system":                true,
		"db.statement":             true,
		"rpc.method":               true,
		"rpc.service":              true,
		"messaging.system":         true,
		"messaging.operation":      true,
		"error.type":               true,
		"exception.type":           true,
		"exception.message":        true,
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
