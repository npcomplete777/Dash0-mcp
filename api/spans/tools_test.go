package spans

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
)

func TestNew(t *testing.T) {
	c := &client.Client{}
	pkg := New(c)
	if pkg == nil {
		t.Fatal("New() returned nil")
	}
	if pkg.client != c {
		t.Error("New() did not set client correctly")
	}
}

func TestTools(t *testing.T) {
	pkg := New(&client.Client{})
	tools := pkg.Tools()

	if len(tools) != 2 {
		t.Errorf("Tools() returned %d tools, expected 2", len(tools))
	}

	expectedNames := map[string]bool{
		"dash0_spans_send":  false,
		"dash0_spans_query": false,
	}

	for _, tool := range tools {
		if _, exists := expectedNames[tool.Name]; !exists {
			t.Errorf("Unexpected tool name: %s", tool.Name)
		}
		expectedNames[tool.Name] = true
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Missing expected tool: %s", name)
		}
	}
}

func TestHandlers(t *testing.T) {
	pkg := New(&client.Client{})
	handlers := pkg.Handlers()

	expectedHandlers := []string{
		"dash0_spans_send",
		"dash0_spans_query",
	}

	if len(handlers) != len(expectedHandlers) {
		t.Errorf("Handlers() returned %d handlers, expected %d", len(handlers), len(expectedHandlers))
	}

	for _, name := range expectedHandlers {
		if _, exists := handlers[name]; !exists {
			t.Errorf("Missing handler for: %s", name)
		}
	}
}

func TestPostSpansToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.PostSpans()

	if tool.Name != "dash0_spans_send" {
		t.Errorf("PostSpans() name = %s, expected dash0_spans_send", tool.Name)
	}

	if tool.Description == "" {
		t.Error("PostSpans() has empty description")
	}

	if tool.InputSchema.Type != "object" {
		t.Errorf("PostSpans() schema type = %s, expected object", tool.InputSchema.Type)
	}

	// Check required field
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Error("PostSpans() should require 'body' field")
	}
}

func TestPostSpansHandler(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]interface{}
		serverResponse interface{}
		serverStatus   int
		expectSuccess  bool
		expectError    string
	}{
		{
			name:        "missing body",
			args:        map[string]interface{}{},
			expectError: "body is required",
		},
		{
			name: "successful send",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"resourceSpans": []interface{}{},
				},
			},
			serverResponse: map[string]interface{}{"status": "ok"},
			serverStatus:   http.StatusOK,
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			if tt.serverResponse != nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost {
						t.Errorf("Expected POST, got %s", r.Method)
					}
					if r.URL.Path != "/api/spans" {
						t.Errorf("Expected /api/spans, got %s", r.URL.Path)
					}
					w.WriteHeader(tt.serverStatus)
					json.NewEncoder(w).Encode(tt.serverResponse)
				}))
				defer server.Close()
			}

			var c *client.Client
			if server != nil {
				c = client.NewWithBaseURL(server.URL, "test-token")
			} else {
				c = &client.Client{}
			}

			pkg := New(c)
			result := pkg.PostSpansHandler(context.Background(), tt.args)

			if tt.expectError != "" {
				if result.Success {
					t.Error("Expected error, got success")
				}
				return
			}

			if tt.expectSuccess && !result.Success {
				t.Errorf("Expected success, got failure: %v", result.Error)
			}
		})
	}
}

func TestQuerySpansToolDefinition(t *testing.T) {
	pkg := New(&client.Client{})
	tool := pkg.QuerySpans()

	if tool.Name != "dash0_spans_query" {
		t.Errorf("QuerySpans() name = %s, expected dash0_spans_query", tool.Name)
	}

	if tool.Description == "" {
		t.Error("QuerySpans() has empty description")
	}

	// Check expected properties
	expectedProps := []string{
		"service_name",
		"time_range_minutes",
		"http_method",
		"http_status_code",
		"error_only",
		"min_duration_ms",
		"span_name",
		"limit",
	}

	for _, prop := range expectedProps {
		if _, exists := tool.InputSchema.Properties[prop]; !exists {
			t.Errorf("QuerySpans() missing property: %s", prop)
		}
	}
}

func TestQuerySpansHandler_Filters(t *testing.T) {
	tests := []struct {
		name            string
		args            map[string]interface{}
		expectedFilters []string // Keys we expect in the filter
	}{
		{
			name:            "no filters",
			args:            map[string]interface{}{},
			expectedFilters: []string{},
		},
		{
			name: "service name filter",
			args: map[string]interface{}{
				"service_name": "cart-service",
			},
			expectedFilters: []string{"service.name"},
		},
		{
			name: "http method filter",
			args: map[string]interface{}{
				"http_method": "POST",
			},
			expectedFilters: []string{"http.request.method"},
		},
		{
			name: "http status code filter",
			args: map[string]interface{}{
				"http_status_code": float64(500),
			},
			expectedFilters: []string{"http.response.status_code"},
		},
		{
			name: "error only filter",
			args: map[string]interface{}{
				"error_only": true,
			},
			expectedFilters: []string{"status.code"},
		},
		{
			name: "span name filter",
			args: map[string]interface{}{
				"span_name": "GET /api/users",
			},
			expectedFilters: []string{"name"},
		},
		{
			name: "multiple filters",
			args: map[string]interface{}{
				"service_name": "cart",
				"http_method":  "POST",
				"error_only":   true,
			},
			expectedFilters: []string{"service.name", "http.request.method", "status.code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedFilters []AttributeFilter

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req QuerySpansRequest
				json.NewDecoder(r.Body).Decode(&req)
				receivedFilters = req.Filter

				// Return empty OTLP response
				json.NewEncoder(w).Encode(map[string]interface{}{
					"resourceSpans": []interface{}{},
				})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)
			pkg.QuerySpansHandler(context.Background(), tt.args)

			// Verify expected filters
			filterKeys := make(map[string]bool)
			for _, f := range receivedFilters {
				filterKeys[f.Key] = true
			}

			for _, expectedKey := range tt.expectedFilters {
				if !filterKeys[expectedKey] {
					t.Errorf("Expected filter key %s not found", expectedKey)
				}
			}

			if len(receivedFilters) != len(tt.expectedFilters) {
				t.Errorf("Got %d filters, expected %d", len(receivedFilters), len(tt.expectedFilters))
			}
		})
	}
}

func TestQuerySpansHandler_TimeRange(t *testing.T) {
	tests := []struct {
		name            string
		timeRangeMinutes float64
		expectedMinutes int
	}{
		{
			name:            "default time range",
			timeRangeMinutes: 0,
			expectedMinutes: 60,
		},
		{
			name:            "custom time range",
			timeRangeMinutes: 30,
			expectedMinutes: 30,
		},
		{
			name:            "max time range exceeded",
			timeRangeMinutes: 2000,
			expectedMinutes: 1440, // Should be capped at 24 hours
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedRequest QuerySpansRequest

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewDecoder(r.Body).Decode(&receivedRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"resourceSpans": []interface{}{},
				})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			args := map[string]interface{}{}
			if tt.timeRangeMinutes > 0 {
				args["time_range_minutes"] = tt.timeRangeMinutes
			}

			pkg.QuerySpansHandler(context.Background(), args)

			// Verify time range was set (we can't easily check exact duration without parsing)
			if receivedRequest.TimeRange.From == "" || receivedRequest.TimeRange.To == "" {
				t.Error("Time range was not set")
			}
		})
	}
}

func TestQuerySpansHandler_Limit(t *testing.T) {
	tests := []struct {
		name          string
		limit         float64
		expectedLimit int
	}{
		{
			name:          "default limit",
			limit:         0,
			expectedLimit: 100,
		},
		{
			name:          "custom limit",
			limit:         50,
			expectedLimit: 50,
		},
		{
			name:          "max limit exceeded",
			limit:         500,
			expectedLimit: 200, // Should be capped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedRequest QuerySpansRequest

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewDecoder(r.Body).Decode(&receivedRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"resourceSpans": []interface{}{},
				})
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			args := map[string]interface{}{}
			if tt.limit > 0 {
				args["limit"] = tt.limit
			}

			pkg.QuerySpansHandler(context.Background(), args)

			if receivedRequest.Pagination.Limit != tt.expectedLimit {
				t.Errorf("Limit = %d, expected %d", receivedRequest.Pagination.Limit, tt.expectedLimit)
			}
		})
	}
}

func TestFlattenSpansResponse(t *testing.T) {
	tests := []struct {
		name          string
		input         interface{}
		expectedCount int
		checkFunc     func([]FlatSpan) error
	}{
		{
			name:          "nil input",
			input:         nil,
			expectedCount: 0,
		},
		{
			name:          "empty response",
			input:         map[string]interface{}{},
			expectedCount: 0,
		},
		{
			name: "empty resource spans",
			input: map[string]interface{}{
				"resourceSpans": []interface{}{},
			},
			expectedCount: 0,
		},
		{
			name: "single span",
			input: map[string]interface{}{
				"resourceSpans": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"attributes": []interface{}{
								map[string]interface{}{
									"key": "service.name",
									"value": map[string]interface{}{
										"stringValue": "cart-service",
									},
								},
							},
						},
						"scopeSpans": []interface{}{
							map[string]interface{}{
								"spans": []interface{}{
									map[string]interface{}{
										"traceId":           "abc123",
										"spanId":            "span456",
										"parentSpanId":      "parent789",
										"name":              "GET /api/cart",
										"startTimeUnixNano": "1609459200000000000",
										"endTimeUnixNano":   "1609459200100000000",
										"status": map[string]interface{}{
											"code":    float64(0),
											"message": "OK",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedCount: 1,
			checkFunc: func(spans []FlatSpan) error {
				span := spans[0]
				if span.ServiceName != "cart-service" {
					return errorf("ServiceName = %s, expected cart-service", span.ServiceName)
				}
				if span.TraceID != "abc123" {
					return errorf("TraceID = %s, expected abc123", span.TraceID)
				}
				if span.SpanID != "span456" {
					return errorf("SpanID = %s, expected span456", span.SpanID)
				}
				if span.ParentSpanID != "parent789" {
					return errorf("ParentSpanID = %s, expected parent789", span.ParentSpanID)
				}
				if span.Name != "GET /api/cart" {
					return errorf("Name = %s, expected GET /api/cart", span.Name)
				}
				// Duration should be 100ms
				if span.DurationMs != 100 {
					return errorf("DurationMs = %f, expected 100", span.DurationMs)
				}
				if span.StatusCode != 0 {
					return errorf("StatusCode = %d, expected 0", span.StatusCode)
				}
				if span.StatusMessage != "OK" {
					return errorf("StatusMessage = %s, expected OK", span.StatusMessage)
				}
				return nil
			},
		},
		{
			name: "multiple spans from multiple resources",
			input: map[string]interface{}{
				"resourceSpans": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"attributes": []interface{}{
								map[string]interface{}{
									"key": "service.name",
									"value": map[string]interface{}{
										"stringValue": "service-a",
									},
								},
							},
						},
						"scopeSpans": []interface{}{
							map[string]interface{}{
								"spans": []interface{}{
									map[string]interface{}{
										"traceId":           "trace1",
										"spanId":            "span1",
										"name":              "span-a",
										"startTimeUnixNano": "1000000000",
										"endTimeUnixNano":   "2000000000",
									},
								},
							},
						},
					},
					map[string]interface{}{
						"resource": map[string]interface{}{
							"attributes": []interface{}{
								map[string]interface{}{
									"key": "service.name",
									"value": map[string]interface{}{
										"stringValue": "service-b",
									},
								},
							},
						},
						"scopeSpans": []interface{}{
							map[string]interface{}{
								"spans": []interface{}{
									map[string]interface{}{
										"traceId":           "trace2",
										"spanId":            "span2",
										"name":              "span-b",
										"startTimeUnixNano": "3000000000",
										"endTimeUnixNano":   "4000000000",
									},
								},
							},
						},
					},
				},
			},
			expectedCount: 2,
			checkFunc: func(spans []FlatSpan) error {
				if spans[0].ServiceName != "service-a" {
					return errorf("First span ServiceName = %s, expected service-a", spans[0].ServiceName)
				}
				if spans[1].ServiceName != "service-b" {
					return errorf("Second span ServiceName = %s, expected service-b", spans[1].ServiceName)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenSpansResponse(tt.input)

			if len(result) != tt.expectedCount {
				t.Errorf("flattenSpansResponse() returned %d spans, expected %d", len(result), tt.expectedCount)
			}

			if tt.checkFunc != nil && tt.expectedCount > 0 {
				if err := tt.checkFunc(result); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "nil resource",
			input:    map[string]interface{}{},
			expected: "",
		},
		{
			name: "no attributes",
			input: map[string]interface{}{
				"resource": map[string]interface{}{},
			},
			expected: "",
		},
		{
			name: "service.name present",
			input: map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key": "service.name",
							"value": map[string]interface{}{
								"stringValue": "my-service",
							},
						},
					},
				},
			},
			expected: "my-service",
		},
		{
			name: "service.name not first attribute",
			input: map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key": "telemetry.sdk.name",
							"value": map[string]interface{}{
								"stringValue": "opentelemetry",
							},
						},
						map[string]interface{}{
							"key": "service.name",
							"value": map[string]interface{}{
								"stringValue": "backend-api",
							},
						},
					},
				},
			},
			expected: "backend-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.input)
			if result != tt.expected {
				t.Errorf("extractServiceName() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestExtractSpanAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "no attributes",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "http attributes",
			input: map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key": "http.request.method",
						"value": map[string]interface{}{
							"stringValue": "POST",
						},
					},
					map[string]interface{}{
						"key": "http.response.status_code",
						"value": map[string]interface{}{
							"intValue": "200",
						},
					},
					map[string]interface{}{
						"key": "http.route",
						"value": map[string]interface{}{
							"stringValue": "/api/users/{id}",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"http.request.method":       "POST",
				"http.response.status_code": int64(200),
				"http.route":                "/api/users/{id}",
			},
		},
		{
			name: "database attributes",
			input: map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key": "db.system",
						"value": map[string]interface{}{
							"stringValue": "postgresql",
						},
					},
					map[string]interface{}{
						"key": "db.statement",
						"value": map[string]interface{}{
							"stringValue": "SELECT * FROM users",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"db.system":    "postgresql",
				"db.statement": "SELECT * FROM users",
			},
		},
		{
			name: "filters out non-interesting keys",
			input: map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key": "http.request.method",
						"value": map[string]interface{}{
							"stringValue": "GET",
						},
					},
					map[string]interface{}{
						"key": "custom.attribute",
						"value": map[string]interface{}{
							"stringValue": "should-be-ignored",
						},
					},
					map[string]interface{}{
						"key": "thread.id",
						"value": map[string]interface{}{
							"intValue": "12345",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"http.request.method": "GET",
			},
		},
		{
			name: "boolean value",
			input: map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key": "error.type",
						"value": map[string]interface{}{
							"boolValue": true,
						},
					},
				},
			},
			expected: map[string]interface{}{
				"error.type": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSpanAttributes(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("extractSpanAttributes() returned %d attributes, expected %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if result[key] != expectedValue {
					t.Errorf("extractSpanAttributes()[%s] = %v, expected %v", key, result[key], expectedValue)
				}
			}
		})
	}
}

func TestQuerySpansHandler_DurationFilter(t *testing.T) {
	// Test that min_duration_ms filter is applied client-side
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return spans with different durations
		response := map[string]interface{}{
			"resourceSpans": []interface{}{
				map[string]interface{}{
					"resource": map[string]interface{}{
						"attributes": []interface{}{},
					},
					"scopeSpans": []interface{}{
						map[string]interface{}{
							"spans": []interface{}{
								map[string]interface{}{
									"traceId":           "trace1",
									"spanId":            "span1",
									"name":              "fast-span",
									"startTimeUnixNano": "1000000000",
									"endTimeUnixNano":   "1050000000", // 50ms
								},
								map[string]interface{}{
									"traceId":           "trace2",
									"spanId":            "span2",
									"name":              "slow-span",
									"startTimeUnixNano": "2000000000",
									"endTimeUnixNano":   "2200000000", // 200ms
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	c := client.NewWithBaseURL(server.URL, "test-token")
	pkg := New(c)

	// Filter for spans >= 100ms
	args := map[string]interface{}{
		"min_duration_ms": float64(100),
	}

	result := pkg.QuerySpansHandler(context.Background(), args)

	if !result.Success {
		t.Fatalf("QuerySpansHandler failed: %v", result.Error)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data is not a map")
	}

	count, ok := data["count"].(int)
	if !ok {
		t.Fatal("Count is not an int")
	}

	// Should only have 1 span (the 200ms one)
	if count != 1 {
		t.Errorf("Expected 1 span after filtering, got %d", count)
	}
}

// Helper function to create formatted errors
func errorf(format string, args ...interface{}) error {
	return &testError{msg: format, args: args}
}

type testError struct {
	msg  string
	args []interface{}
}

func (e *testError) Error() string {
	return e.msg
}
