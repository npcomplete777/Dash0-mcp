package logs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	pkg := New(c)
	if pkg == nil {
		t.Fatal("New returned nil")
	}
	if pkg.client == nil {
		t.Error("client is nil")
	}
}

func TestTools(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	pkg := New(c)
	tools := pkg.Tools()

	if len(tools) != 2 {
		t.Errorf("Tools() returned %d tools, want 2", len(tools))
	}

	// Verify tool names
	expectedNames := map[string]bool{
		"dash0_logs_send":  true,
		"dash0_logs_query": true,
	}

	for _, tool := range tools {
		if !expectedNames[tool.Name] {
			t.Errorf("Unexpected tool name: %s", tool.Name)
		}
		delete(expectedNames, tool.Name)
	}

	for name := range expectedNames {
		t.Errorf("Missing expected tool: %s", name)
	}
}

func TestHandlers(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.example.com",
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	pkg := New(c)
	handlers := pkg.Handlers()

	expectedHandlers := []string{
		"dash0_logs_send",
		"dash0_logs_query",
	}

	for _, name := range expectedHandlers {
		if _, ok := handlers[name]; !ok {
			t.Errorf("Missing handler for %s", name)
		}
	}
}

func TestPostLogsHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/logs" {
			t.Errorf("Path = %s, want /api/logs", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	pkg := New(c)

	// Test with body
	result := pkg.PostLogsHandler(context.Background(), map[string]interface{}{
		"body": map[string]interface{}{
			"resourceLogs": []interface{}{},
		},
	})

	if !result.Success {
		t.Errorf("PostLogsHandler failed: %+v", result.Error)
	}

	// Test without body
	result = pkg.PostLogsHandler(context.Background(), map[string]interface{}{})
	if result.Success {
		t.Error("PostLogsHandler should fail without body")
	}
	if result.Error.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", result.Error.StatusCode)
	}
}

func TestFlattenLogsResponse(t *testing.T) {
	// Test with valid OTLP logs response
	otlpResponse := map[string]interface{}{
		"resourceLogs": []interface{}{
			map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key": "service.name",
							"value": map[string]interface{}{
								"stringValue": "test-service",
							},
						},
					},
				},
				"scopeLogs": []interface{}{
					map[string]interface{}{
						"logRecords": []interface{}{
							map[string]interface{}{
								"timeUnixNano":   "1704067200000000000",
								"severityText":   "ERROR",
								"severityNumber": float64(17),
								"body": map[string]interface{}{
									"stringValue": "Test error message",
								},
								"traceId": "abc123",
								"spanId":  "def456",
								"attributes": []interface{}{
									map[string]interface{}{
										"key": "custom.attr",
										"value": map[string]interface{}{
											"stringValue": "custom-value",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	logs := flattenLogsResponse(otlpResponse)

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}

	log := logs[0]

	if log.ServiceName != "test-service" {
		t.Errorf("ServiceName = %q, want %q", log.ServiceName, "test-service")
	}

	if log.SeverityText != "ERROR" {
		t.Errorf("SeverityText = %q, want %q", log.SeverityText, "ERROR")
	}

	if log.SeverityNumber != 17 {
		t.Errorf("SeverityNumber = %d, want %d", log.SeverityNumber, 17)
	}

	if log.Body != "Test error message" {
		t.Errorf("Body = %q, want %q", log.Body, "Test error message")
	}

	if log.TraceID != "abc123" {
		t.Errorf("TraceID = %q, want %q", log.TraceID, "abc123")
	}

	if log.SpanID != "def456" {
		t.Errorf("SpanID = %q, want %q", log.SpanID, "def456")
	}

	if log.Attributes["custom.attr"] != "custom-value" {
		t.Errorf("Attributes[custom.attr] = %v, want %q", log.Attributes["custom.attr"], "custom-value")
	}

	// Verify timestamp was parsed
	if log.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestFlattenLogsResponseEmpty(t *testing.T) {
	// Test with nil
	logs := flattenLogsResponse(nil)
	if len(logs) != 0 {
		t.Errorf("Expected 0 logs for nil input, got %d", len(logs))
	}

	// Test with empty map
	logs = flattenLogsResponse(map[string]interface{}{})
	if len(logs) != 0 {
		t.Errorf("Expected 0 logs for empty map, got %d", len(logs))
	}

	// Test with non-map input
	logs = flattenLogsResponse("invalid")
	if len(logs) != 0 {
		t.Errorf("Expected 0 logs for non-map input, got %d", len(logs))
	}
}

func TestFlattenLogsResponseMultipleLogs(t *testing.T) {
	otlpResponse := map[string]interface{}{
		"resourceLogs": []interface{}{
			map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key":   "service.name",
							"value": map[string]interface{}{"stringValue": "service-1"},
						},
					},
				},
				"scopeLogs": []interface{}{
					map[string]interface{}{
						"logRecords": []interface{}{
							map[string]interface{}{
								"severityText": "INFO",
								"body":         map[string]interface{}{"stringValue": "Log 1"},
							},
							map[string]interface{}{
								"severityText": "WARN",
								"body":         map[string]interface{}{"stringValue": "Log 2"},
							},
						},
					},
				},
			},
			map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key":   "service.name",
							"value": map[string]interface{}{"stringValue": "service-2"},
						},
					},
				},
				"scopeLogs": []interface{}{
					map[string]interface{}{
						"logRecords": []interface{}{
							map[string]interface{}{
								"severityText": "ERROR",
								"body":         map[string]interface{}{"stringValue": "Log 3"},
							},
						},
					},
				},
			},
		},
	}

	logs := flattenLogsResponse(otlpResponse)

	if len(logs) != 3 {
		t.Fatalf("Expected 3 logs, got %d", len(logs))
	}

	// Verify service names
	if logs[0].ServiceName != "service-1" || logs[1].ServiceName != "service-1" {
		t.Error("First two logs should have service-1")
	}
	if logs[2].ServiceName != "service-2" {
		t.Error("Third log should have service-2")
	}
}

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name: "valid service name",
			input: map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key":   "service.name",
							"value": map[string]interface{}{"stringValue": "my-service"},
						},
					},
				},
			},
			expected: "my-service",
		},
		{
			name:     "no resource",
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
			name: "no service.name attribute",
			input: map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key":   "other.attr",
							"value": map[string]interface{}{"stringValue": "value"},
						},
					},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.input)
			if result != tt.expected {
				t.Errorf("extractServiceName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSeverityOrder(t *testing.T) {
	// Verify severity ordering is correct
	if severityOrder["TRACE"] >= severityOrder["DEBUG"] {
		t.Error("TRACE should be less than DEBUG")
	}
	if severityOrder["DEBUG"] >= severityOrder["INFO"] {
		t.Error("DEBUG should be less than INFO")
	}
	if severityOrder["INFO"] >= severityOrder["WARN"] {
		t.Error("INFO should be less than WARN")
	}
	if severityOrder["WARN"] >= severityOrder["ERROR"] {
		t.Error("WARN should be less than ERROR")
	}
	if severityOrder["ERROR"] >= severityOrder["FATAL"] {
		t.Error("ERROR should be less than FATAL")
	}
}

func TestQueryLogsHandlerValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"resourceLogs": []interface{}{},
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	pkg := New(c)

	// Test with valid parameters
	result := pkg.QueryLogsHandler(context.Background(), map[string]interface{}{
		"service_name":       "test-service",
		"time_range_minutes": float64(30),
		"min_severity":       "ERROR",
		"limit":              float64(50),
	})

	if !result.Success {
		t.Errorf("QueryLogsHandler failed: %+v", result.Error)
	}

	// Verify response structure
	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data is not a map")
	}

	if _, ok := data["logs"]; !ok {
		t.Error("Response missing 'logs' field")
	}
	if _, ok := data["count"]; !ok {
		t.Error("Response missing 'count' field")
	}
	if _, ok := data["query"]; !ok {
		t.Error("Response missing 'query' field")
	}
}

func TestQueryLogsHandlerLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"resourceLogs": []interface{}{},
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		BaseURL:   server.URL,
		AuthToken: "test-token",
	}
	c := client.New(cfg)

	pkg := New(c)

	// Test with excessive time range (should be capped at 1440)
	result := pkg.QueryLogsHandler(context.Background(), map[string]interface{}{
		"time_range_minutes": float64(10000),
	})

	if !result.Success {
		t.Errorf("QueryLogsHandler failed: %+v", result.Error)
	}

	// Test with excessive limit (should be capped at 500)
	result = pkg.QueryLogsHandler(context.Background(), map[string]interface{}{
		"limit": float64(10000),
	})

	if !result.Success {
		t.Errorf("QueryLogsHandler failed: %+v", result.Error)
	}
}

func TestExtractLogAttributes(t *testing.T) {
	logMap := map[string]interface{}{
		"attributes": []interface{}{
			map[string]interface{}{
				"key":   "string.attr",
				"value": map[string]interface{}{"stringValue": "string-value"},
			},
			map[string]interface{}{
				"key":   "int.attr",
				"value": map[string]interface{}{"intValue": "42"},
			},
			map[string]interface{}{
				"key":   "bool.attr",
				"value": map[string]interface{}{"boolValue": true},
			},
		},
	}

	attrs := extractLogAttributes(logMap)

	if attrs["string.attr"] != "string-value" {
		t.Errorf("string.attr = %v, want %q", attrs["string.attr"], "string-value")
	}

	if attrs["int.attr"] != int64(42) {
		t.Errorf("int.attr = %v, want %d", attrs["int.attr"], 42)
	}

	if attrs["bool.attr"] != true {
		t.Errorf("bool.attr = %v, want %v", attrs["bool.attr"], true)
	}
}
