package logs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/npcomplete777/dash0-mcp/internal/client"
)

func TestTools_Tools(t *testing.T) {
	c := client.NewWithBaseURL("http://example.com", "test-token")
	pkg := New(c)

	tools := pkg.Tools()

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	expectedNames := map[string]bool{
		"dash0_logs_send":  false,
		"dash0_logs_query": false,
	}

	for _, tool := range tools {
		if _, exists := expectedNames[tool.Name]; !exists {
			t.Errorf("unexpected tool name: %s", tool.Name)
		}
		expectedNames[tool.Name] = true
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected tool %s not found", name)
		}
	}
}

func TestTools_Handlers(t *testing.T) {
	c := client.NewWithBaseURL("http://example.com", "test-token")
	pkg := New(c)

	handlers := pkg.Handlers()

	if len(handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(handlers))
	}

	expectedHandlers := []string{"dash0_logs_send", "dash0_logs_query"}
	for _, name := range expectedHandlers {
		if _, exists := handlers[name]; !exists {
			t.Errorf("handler %s not found", name)
		}
	}
}

func TestPostLogsHandler(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]interface{}
		serverCode  int
		serverResp  interface{}
		wantSuccess bool
	}{
		{
			name: "successful send",
			args: map[string]interface{}{
				"body": map[string]interface{}{
					"resourceLogs": []interface{}{},
				},
			},
			serverCode:  http.StatusOK,
			serverResp:  map[string]interface{}{"status": "ok"},
			wantSuccess: true,
		},
		{
			name:        "missing body",
			args:        map[string]interface{}{},
			wantSuccess: false,
		},
		{
			name: "server error",
			args: map[string]interface{}{
				"body": map[string]interface{}{},
			},
			serverCode:  http.StatusInternalServerError,
			serverResp:  map[string]interface{}{"error": "internal error"},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			if tt.serverCode != 0 {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost {
						t.Errorf("expected POST, got %s", r.Method)
					}
					if r.URL.Path != "/api/logs" {
						t.Errorf("expected /api/logs, got %s", r.URL.Path)
					}
					w.WriteHeader(tt.serverCode)
					json.NewEncoder(w).Encode(tt.serverResp)
				}))
				defer server.Close()
			}

			var c *client.Client
			if server != nil {
				c = client.NewWithBaseURL(server.URL, "test-token")
			} else {
				c = client.NewWithBaseURL("http://example.com", "test-token")
			}
			pkg := New(c)

			result := pkg.PostLogsHandler(context.Background(), tt.args)

			if result.Success != tt.wantSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.wantSuccess)
			}
		})
	}
}

func TestQueryLogsHandler(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]interface{}
		serverCode  int
		serverResp  interface{}
		wantSuccess bool
		checkResult func(t *testing.T, result *client.ToolResult)
	}{
		{
			name:       "basic query",
			args:       map[string]interface{}{},
			serverCode: http.StatusOK,
			serverResp: map[string]interface{}{
				"resourceLogs": []interface{}{},
			},
			wantSuccess: true,
		},
		{
			name: "query with service name filter",
			args: map[string]interface{}{
				"service_name": "test-service",
			},
			serverCode: http.StatusOK,
			serverResp: map[string]interface{}{
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
										"severityText":   "INFO",
										"severityNumber": float64(9),
										"body": map[string]interface{}{
											"stringValue": "Test log message",
										},
										"traceId": "abc123",
										"spanId":  "def456",
									},
								},
							},
						},
					},
				},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result *client.ToolResult) {
				data, ok := result.Data.(map[string]interface{})
				if !ok {
					t.Fatal("expected data to be map")
				}
				logs, ok := data["logs"].([]FlatLog)
				if !ok {
					t.Fatal("expected logs to be []FlatLog")
				}
				if len(logs) != 1 {
					t.Errorf("expected 1 log, got %d", len(logs))
				}
				if logs[0].ServiceName != "test-service" {
					t.Errorf("expected service name test-service, got %s", logs[0].ServiceName)
				}
			},
		},
		{
			name: "query with time range",
			args: map[string]interface{}{
				"time_range_minutes": float64(30),
			},
			serverCode: http.StatusOK,
			serverResp: map[string]interface{}{
				"resourceLogs": []interface{}{},
			},
			wantSuccess: true,
		},
		{
			name: "query with max time range",
			args: map[string]interface{}{
				"time_range_minutes": float64(2000), // Should be capped to 1440
			},
			serverCode: http.StatusOK,
			serverResp: map[string]interface{}{
				"resourceLogs": []interface{}{},
			},
			wantSuccess: true,
		},
		{
			name: "query with limit",
			args: map[string]interface{}{
				"limit": float64(50),
			},
			serverCode: http.StatusOK,
			serverResp: map[string]interface{}{
				"resourceLogs": []interface{}{},
			},
			wantSuccess: true,
		},
		{
			name: "query with max limit",
			args: map[string]interface{}{
				"limit": float64(1000), // Should be capped to 500
			},
			serverCode: http.StatusOK,
			serverResp: map[string]interface{}{
				"resourceLogs": []interface{}{},
			},
			wantSuccess: true,
		},
		{
			name: "query with severity filter",
			args: map[string]interface{}{
				"min_severity": "ERROR",
			},
			serverCode: http.StatusOK,
			serverResp: map[string]interface{}{
				"resourceLogs": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{},
						"scopeLogs": []interface{}{
							map[string]interface{}{
								"logRecords": []interface{}{
									map[string]interface{}{
										"timeUnixNano":   "1704067200000000000",
										"severityText":   "INFO",
										"severityNumber": float64(9),
										"body":           map[string]interface{}{"stringValue": "Info message"},
									},
									map[string]interface{}{
										"timeUnixNano":   "1704067200000000000",
										"severityText":   "ERROR",
										"severityNumber": float64(17),
										"body":           map[string]interface{}{"stringValue": "Error message"},
									},
								},
							},
						},
					},
				},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result *client.ToolResult) {
				data, ok := result.Data.(map[string]interface{})
				if !ok {
					t.Fatal("expected data to be map")
				}
				logs, ok := data["logs"].([]FlatLog)
				if !ok {
					t.Fatal("expected logs to be []FlatLog")
				}
				// Should filter out INFO, keep only ERROR
				if len(logs) != 1 {
					t.Errorf("expected 1 log after severity filter, got %d", len(logs))
				}
				if len(logs) > 0 && logs[0].SeverityText != "ERROR" {
					t.Errorf("expected ERROR severity, got %s", logs[0].SeverityText)
				}
			},
		},
		{
			name: "query with body contains filter",
			args: map[string]interface{}{
				"body_contains": "error",
			},
			serverCode: http.StatusOK,
			serverResp: map[string]interface{}{
				"resourceLogs": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{},
						"scopeLogs": []interface{}{
							map[string]interface{}{
								"logRecords": []interface{}{
									map[string]interface{}{
										"timeUnixNano": "1704067200000000000",
										"severityText": "INFO",
										"body":         map[string]interface{}{"stringValue": "Normal message"},
									},
									map[string]interface{}{
										"timeUnixNano": "1704067200000000000",
										"severityText": "ERROR",
										"body":         map[string]interface{}{"stringValue": "An Error occurred"},
									},
								},
							},
						},
					},
				},
			},
			wantSuccess: true,
			checkResult: func(t *testing.T, result *client.ToolResult) {
				data, ok := result.Data.(map[string]interface{})
				if !ok {
					t.Fatal("expected data to be map")
				}
				logs, ok := data["logs"].([]FlatLog)
				if !ok {
					t.Fatal("expected logs to be []FlatLog")
				}
				// Should filter to only message containing "error" (case insensitive)
				if len(logs) != 1 {
					t.Errorf("expected 1 log after body filter, got %d", len(logs))
				}
				if len(logs) > 0 && !strings.Contains(strings.ToLower(logs[0].Body), "error") {
					t.Errorf("expected body to contain 'error', got %s", logs[0].Body)
				}
			},
		},
		{
			name:       "server error",
			args:       map[string]interface{}{},
			serverCode: http.StatusInternalServerError,
			serverResp: map[string]interface{}{
				"error": "internal error",
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverCode)
				json.NewEncoder(w).Encode(tt.serverResp)
			}))
			defer server.Close()

			c := client.NewWithBaseURL(server.URL, "test-token")
			pkg := New(c)

			result := pkg.QueryLogsHandler(context.Background(), tt.args)

			if result.Success != tt.wantSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if tt.checkResult != nil && result.Success {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestQueryLogsHandler_MarkdownOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"resourceLogs": []interface{}{
				map[string]interface{}{
					"resource": map[string]interface{}{
						"attributes": []interface{}{
							map[string]interface{}{
								"key":   "service.name",
								"value": map[string]interface{}{"stringValue": "frontend"},
							},
							map[string]interface{}{
								"key":   "k8s.pod.name",
								"value": map[string]interface{}{"stringValue": "frontend-pod-xyz"},
							},
							map[string]interface{}{
								"key":   "k8s.namespace.name",
								"value": map[string]interface{}{"stringValue": "production"},
							},
							map[string]interface{}{
								"key":   "k8s.container.name",
								"value": map[string]interface{}{"stringValue": "frontend-container"},
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
									"body":           map[string]interface{}{"stringValue": "Failed to connect to database"},
									"traceId":        "trace-abc-123",
									"spanId":         "span-def-456",
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

	result := pkg.QueryLogsHandler(context.Background(), map[string]interface{}{
		"service_name": "frontend",
	})

	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}

	if result.Markdown == "" {
		t.Fatal("expected Markdown to be set")
	}

	md := result.Markdown
	if !strings.Contains(md, "## Log Query Results") {
		t.Error("markdown should contain title")
	}
	if !strings.Contains(md, "**Found 1 logs**") {
		t.Error("markdown should contain count")
	}
	if !strings.Contains(md, "frontend") {
		t.Error("markdown should contain service name")
	}
	if !strings.Contains(md, "ERROR") {
		t.Error("markdown should contain severity")
	}
	if !strings.Contains(md, "Failed to connect") {
		t.Error("markdown should contain log body")
	}
	if !strings.Contains(md, "frontend-pod-xyz") {
		t.Error("markdown should contain pod name")
	}
	if !strings.Contains(md, "service=frontend") {
		t.Error("markdown should contain filter description")
	}

	// Verify K8s fields in structured data
	data := result.Data.(map[string]interface{})
	logs := data["logs"].([]FlatLog)
	if logs[0].K8sNamespace != "production" {
		t.Errorf("K8sNamespace = %s, want production", logs[0].K8sNamespace)
	}
	if logs[0].K8sPodName != "frontend-pod-xyz" {
		t.Errorf("K8sPodName = %s, want frontend-pod-xyz", logs[0].K8sPodName)
	}
	if logs[0].K8sContainerName != "frontend-container" {
		t.Errorf("K8sContainerName = %s, want frontend-container", logs[0].K8sContainerName)
	}
}

func TestFlattenLogsResponse(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		wantLogs int
	}{
		{
			name:     "nil data",
			data:     nil,
			wantLogs: 0,
		},
		{
			name:     "non-map data",
			data:     "string",
			wantLogs: 0,
		},
		{
			name:     "empty resourceLogs",
			data:     map[string]interface{}{"resourceLogs": []interface{}{}},
			wantLogs: 0,
		},
		{
			name:     "invalid resourceLogs type",
			data:     map[string]interface{}{"resourceLogs": "invalid"},
			wantLogs: 0,
		},
		{
			name: "single log record with K8s info",
			data: map[string]interface{}{
				"resourceLogs": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"attributes": []interface{}{
								map[string]interface{}{
									"key":   "service.name",
									"value": map[string]interface{}{"stringValue": "my-service"},
								},
								map[string]interface{}{
									"key":   "k8s.pod.name",
									"value": map[string]interface{}{"stringValue": "pod-abc"},
								},
								map[string]interface{}{
									"key":   "k8s.namespace.name",
									"value": map[string]interface{}{"stringValue": "default"},
								},
								map[string]interface{}{
									"key":   "k8s.container.name",
									"value": map[string]interface{}{"stringValue": "main"},
								},
							},
						},
						"scopeLogs": []interface{}{
							map[string]interface{}{
								"logRecords": []interface{}{
									map[string]interface{}{
										"timeUnixNano":   "1704067200000000000",
										"severityText":   "INFO",
										"severityNumber": float64(9),
										"body":           map[string]interface{}{"stringValue": "Test message"},
										"traceId":        "trace123",
										"spanId":         "span456",
										"attributes": []interface{}{
											map[string]interface{}{
												"key":   "custom.attr",
												"value": map[string]interface{}{"stringValue": "custom-value"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLogs: 1,
		},
		{
			name: "log with observedTimeUnixNano",
			data: map[string]interface{}{
				"resourceLogs": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{},
						"scopeLogs": []interface{}{
							map[string]interface{}{
								"logRecords": []interface{}{
									map[string]interface{}{
										"observedTimeUnixNano": "1704067200000000000",
										"severityText":         "WARN",
										"body":                 map[string]interface{}{"stringValue": "Warning"},
									},
								},
							},
						},
					},
				},
			},
			wantLogs: 1,
		},
		{
			name: "multiple log records",
			data: map[string]interface{}{
				"resourceLogs": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{},
						"scopeLogs": []interface{}{
							map[string]interface{}{
								"logRecords": []interface{}{
									map[string]interface{}{"body": map[string]interface{}{"stringValue": "Log 1"}},
									map[string]interface{}{"body": map[string]interface{}{"stringValue": "Log 2"}},
									map[string]interface{}{"body": map[string]interface{}{"stringValue": "Log 3"}},
								},
							},
						},
					},
				},
			},
			wantLogs: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := flattenLogsResponse(tt.data)
			if len(logs) != tt.wantLogs {
				t.Errorf("got %d logs, want %d", len(logs), tt.wantLogs)
			}
		})
	}
}

func TestFlattenLogsResponse_K8sFields(t *testing.T) {
	data := map[string]interface{}{
		"resourceLogs": []interface{}{
			map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key":   "service.name",
							"value": map[string]interface{}{"stringValue": "test-svc"},
						},
						map[string]interface{}{
							"key":   "k8s.pod.name",
							"value": map[string]interface{}{"stringValue": "my-pod"},
						},
						map[string]interface{}{
							"key":   "k8s.namespace.name",
							"value": map[string]interface{}{"stringValue": "my-ns"},
						},
						map[string]interface{}{
							"key":   "k8s.container.name",
							"value": map[string]interface{}{"stringValue": "my-container"},
						},
					},
				},
				"scopeLogs": []interface{}{
					map[string]interface{}{
						"logRecords": []interface{}{
							map[string]interface{}{
								"body": map[string]interface{}{"stringValue": "test"},
							},
						},
					},
				},
			},
		},
	}

	logs := flattenLogsResponse(data)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	log := logs[0]
	if log.K8sPodName != "my-pod" {
		t.Errorf("K8sPodName = %s, want my-pod", log.K8sPodName)
	}
	if log.K8sNamespace != "my-ns" {
		t.Errorf("K8sNamespace = %s, want my-ns", log.K8sNamespace)
	}
	if log.K8sContainerName != "my-container" {
		t.Errorf("K8sContainerName = %s, want my-container", log.K8sContainerName)
	}
}

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name  string
		rlMap map[string]interface{}
		want  string
	}{
		{
			name:  "nil resource",
			rlMap: map[string]interface{}{},
			want:  "",
		},
		{
			name: "no attributes",
			rlMap: map[string]interface{}{
				"resource": map[string]interface{}{},
			},
			want: "",
		},
		{
			name: "service.name found",
			rlMap: map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key":   "service.name",
							"value": map[string]interface{}{"stringValue": "my-service"},
						},
					},
				},
			},
			want: "my-service",
		},
		{
			name: "service.name not found",
			rlMap: map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key":   "other.attr",
							"value": map[string]interface{}{"stringValue": "value"},
						},
					},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractServiceName(tt.rlMap)
			if got != tt.want {
				t.Errorf("extractServiceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractResourceAttribute(t *testing.T) {
	tests := []struct {
		name  string
		rlMap map[string]interface{}
		key   string
		want  string
	}{
		{
			name:  "empty resource",
			rlMap: map[string]interface{}{},
			key:   "k8s.pod.name",
			want:  "",
		},
		{
			name: "attribute found",
			rlMap: map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key":   "k8s.namespace.name",
							"value": map[string]interface{}{"stringValue": "production"},
						},
					},
				},
			},
			key:  "k8s.namespace.name",
			want: "production",
		},
		{
			name: "attribute not found",
			rlMap: map[string]interface{}{
				"resource": map[string]interface{}{
					"attributes": []interface{}{
						map[string]interface{}{
							"key":   "other.key",
							"value": map[string]interface{}{"stringValue": "val"},
						},
					},
				},
			},
			key:  "k8s.pod.name",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResourceAttribute(tt.rlMap, tt.key)
			if got != tt.want {
				t.Errorf("extractResourceAttribute(%s) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestExtractLogAttributes(t *testing.T) {
	tests := []struct {
		name   string
		logMap map[string]interface{}
		want   map[string]interface{}
	}{
		{
			name:   "no attributes",
			logMap: map[string]interface{}{},
			want:   map[string]interface{}{},
		},
		{
			name: "string attribute",
			logMap: map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key":   "string.attr",
						"value": map[string]interface{}{"stringValue": "test-value"},
					},
				},
			},
			want: map[string]interface{}{"string.attr": "test-value"},
		},
		{
			name: "int attribute",
			logMap: map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key":   "int.attr",
						"value": map[string]interface{}{"intValue": "42"},
					},
				},
			},
			want: map[string]interface{}{"int.attr": int64(42)},
		},
		{
			name: "bool attribute",
			logMap: map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key":   "bool.attr",
						"value": map[string]interface{}{"boolValue": true},
					},
				},
			},
			want: map[string]interface{}{"bool.attr": true},
		},
		{
			name: "multiple attributes",
			logMap: map[string]interface{}{
				"attributes": []interface{}{
					map[string]interface{}{
						"key":   "attr1",
						"value": map[string]interface{}{"stringValue": "value1"},
					},
					map[string]interface{}{
						"key":   "attr2",
						"value": map[string]interface{}{"stringValue": "value2"},
					},
				},
			},
			want: map[string]interface{}{"attr1": "value1", "attr2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLogAttributes(tt.logMap)
			if len(got) != len(tt.want) {
				t.Errorf("got %d attributes, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("attribute %s = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestPostLogs_ToolDefinition(t *testing.T) {
	c := client.NewWithBaseURL("http://example.com", "test-token")
	pkg := New(c)

	tool := pkg.PostLogs()

	if tool.Name != "dash0_logs_send" {
		t.Errorf("tool name = %s, want dash0_logs_send", tool.Name)
	}
	if tool.Description == "" {
		t.Error("tool description should not be empty")
	}
	if tool.InputSchema.Type != "object" {
		t.Errorf("input schema type = %s, want object", tool.InputSchema.Type)
	}
	if len(tool.InputSchema.Required) != 1 || tool.InputSchema.Required[0] != "body" {
		t.Errorf("required = %v, want [body]", tool.InputSchema.Required)
	}
}

func TestQueryLogs_ToolDefinition(t *testing.T) {
	c := client.NewWithBaseURL("http://example.com", "test-token")
	pkg := New(c)

	tool := pkg.QueryLogs()

	if tool.Name != "dash0_logs_query" {
		t.Errorf("tool name = %s, want dash0_logs_query", tool.Name)
	}
	if tool.Description == "" {
		t.Error("tool description should not be empty")
	}
	if tool.InputSchema.Type != "object" {
		t.Errorf("input schema type = %s, want object", tool.InputSchema.Type)
	}

	// Verify all expected properties exist
	expectedProps := []string{"service_name", "time_range_minutes", "min_severity", "body_contains", "limit"}
	for _, prop := range expectedProps {
		if _, exists := tool.InputSchema.Properties[prop]; !exists {
			t.Errorf("expected property %s not found", prop)
		}
	}
}

func TestSeverityOrder(t *testing.T) {
	// Verify severity ordering is correct
	expectedOrder := []struct {
		severity string
		level    int
	}{
		{"TRACE", 1},
		{"DEBUG", 5},
		{"INFO", 9},
		{"WARN", 13},
		{"ERROR", 17},
		{"FATAL", 21},
	}

	for _, tt := range expectedOrder {
		if severityOrder[tt.severity] != tt.level {
			t.Errorf("severityOrder[%s] = %d, want %d", tt.severity, severityOrder[tt.severity], tt.level)
		}
	}

	// Verify ordering relationships
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

func TestQueryLogsHandler_NegativeTimeRange(t *testing.T) {
	c := client.NewWithBaseURL("http://example.com", "test-token")
	pkg := New(c)

	result := pkg.QueryLogsHandler(context.Background(), map[string]interface{}{
		"time_range_minutes": float64(-10),
	})

	if result.Success {
		t.Error("expected error for negative time range")
	}
}

func TestQueryLogsHandler_NegativeLimit(t *testing.T) {
	c := client.NewWithBaseURL("http://example.com", "test-token")
	pkg := New(c)

	result := pkg.QueryLogsHandler(context.Background(), map[string]interface{}{
		"limit": float64(-5),
	})

	if result.Success {
		t.Error("expected error for negative limit")
	}
}

func TestBuildLogStats(t *testing.T) {
	logs := []FlatLog{
		{SeverityText: "ERROR", ServiceName: "svc-a", K8sPodName: "pod-1", TraceID: "t1"},
		{SeverityText: "ERROR", ServiceName: "svc-a", K8sPodName: "pod-1", TraceID: "t2"},
		{SeverityText: "INFO", ServiceName: "svc-b", K8sPodName: "pod-2", TraceID: ""},
		{SeverityText: "WARN", ServiceName: "svc-a", K8sPodName: "", TraceID: "t3"},
	}

	result := buildLogStats(logs)

	if !strings.Contains(result, "**Stats:**") {
		t.Error("should contain Stats header")
	}
	if !strings.Contains(result, "ERROR: 2") {
		t.Error("should show ERROR count")
	}
	if !strings.Contains(result, "WARN: 1") {
		t.Error("should show WARN count")
	}
	if !strings.Contains(result, "INFO: 1") {
		t.Error("should show INFO count")
	}
	if !strings.Contains(result, "Services:") {
		t.Error("should contain services")
	}
	if !strings.Contains(result, "svc-a") {
		t.Error("should contain svc-a")
	}
	if !strings.Contains(result, "With traces: 75%") {
		t.Errorf("should show 75%% trace correlation, got: %s", result)
	}
	if !strings.Contains(result, "Pods:") {
		t.Error("should contain pods")
	}
}

func TestBuildLogStats_EmptyLogs(t *testing.T) {
	logs := []FlatLog{
		{SeverityText: "", ServiceName: "", TraceID: ""},
	}

	result := buildLogStats(logs)

	if !strings.Contains(result, "UNSET: 1") {
		t.Errorf("should show UNSET for empty severity, got: %s", result)
	}
	if !strings.Contains(result, "With traces: 0%") {
		t.Errorf("should show 0%% trace correlation, got: %s", result)
	}
}
