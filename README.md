# Dash0 MCP Server

A Model Context Protocol (MCP) server for the Dash0 Observability Platform, enabling AI assistants to interact with Dash0's APIs for monitoring, alerting, and observability management.

## Features

- **Telemetry Query**: Query logs and spans with rich filtering, markdown table output, and summary statistics (P95 latency, error rates, severity distribution)
- **Telemetry Ingestion**: Send OTLP logs and spans to Dash0
- **Dashboard Management**: Create, read, update, and delete Perses dashboards
- **Alerting**: Manage check rules and view active firing/pending alerts
- **Views**: Save and manage query views for logs and traces
- **Synthetic Monitoring**: Configure synthetic checks for proactive monitoring
- **Sampling Rules**: Control data ingestion rates and costs
- **Migration**: Import configurations from other observability platforms
- **Profile-based Tool Control**: Enable/disable tools via YAML configuration
- **LLM-Optimized Output**: All tools return formatted markdown tables with summaries, statistics, and pagination hints — not raw JSON

## Installation

### Build from Source

```bash
cd /path/to/dash0-mcp-server
go build -o dash0-mcp ./cmd/server
```

### Install with Go

```bash
go install github.com/npcomplete777/dash0-mcp/cmd/server@latest
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DASH0_AUTH_TOKEN` | Yes | Bearer token for API authentication |
| `DASH0_REGION` | No | Region: `eu-west-1` (default), `us-east-1`, or `us-west-2` |
| `DASH0_BASE_URL` | No | Custom base URL (overrides region) |
| `DASH0_DATASET` | No | Dataset to use for all API calls (e.g., `otel-demo-gitops`) |
| `DASH0_DEBUG` | No | Enable debug logging (`true`/`false`) |
| `DASH0_MCP_PROFILE` | No | Tool profile: `full`, `demo`, `readonly`, `minimal` |
| `DASH0_MCP_CONFIG_DIR` | No | Path to config directory (default: `./config`) |

### Obtaining an Auth Token

1. Log in to your Dash0 account
2. Navigate to Settings > API Tokens
3. Create a new token with appropriate permissions
4. Set the token as `DASH0_AUTH_TOKEN`

## Tool Profiles

The server supports profile-based tool enablement, allowing you to control which tools are exposed to the AI assistant.

### Available Profiles

| Profile | Tools | Description |
|---------|-------|-------------|
| `full` | 34 | All tools except destructive delete operations |
| `demo` | 19 | Workflow-focused tools for demos and VALIS integration |
| `readonly` | 12 | Read-only operations (list/get only) |
| `minimal` | 8 | Core query operations only |

### Profile Configuration

Profiles are defined in YAML files under `config/profiles/`. Each profile specifies which tools to enable or disable.

**Example: full.yaml**
```yaml
name: full
description: "Full access - all tools except destructive deletes"
enable_all: true
disable:
  - dash0_dashboards_delete
  - dash0_alerting_check_rules_delete
  - dash0_synthetic_checks_delete
  - dash0_sampling_rules_delete
  - dash0_views_delete
```

**Example: minimal.yaml**
```yaml
name: minimal
description: "Minimal - core query operations only"
enable:
  - dash0_logs_query
  - dash0_spans_query
  - dash0_dashboards_list
  - dash0_dashboards_get
  - dash0_alerting_check_rules_list
  - dash0_alerting_check_rules_get
  - dash0_views_list
  - dash0_views_get
disable_unlisted: true
```

### Creating Custom Profiles

1. Create a new YAML file in `config/profiles/`
2. Define enabled/disabled tools using:
   - `enable_all: true` + `disable: [...]` for permissive profiles
   - `enable: [...]` + `disable_unlisted: true` for restrictive profiles
   - `enable: [...]` + `disable: [...]` for explicit overrides

## Usage

### Claude Desktop Configuration

Add to your Claude Desktop configuration (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "dash0": {
      "command": "/path/to/dash0-mcp",
      "env": {
        "DASH0_AUTH_TOKEN": "your-auth-token",
        "DASH0_REGION": "eu-west-1",
        "DASH0_DATASET": "otel-demo-gitops",
        "DASH0_MCP_PROFILE": "full",
        "DASH0_MCP_CONFIG_DIR": "/path/to/dash0-mcp-server/config"
      }
    }
  }
}
```

### Running Manually

```bash
export DASH0_AUTH_TOKEN="your-auth-token"
export DASH0_REGION="eu-west-1"
export DASH0_DATASET="otel-demo-gitops"
export DASH0_MCP_PROFILE="demo"
./dash0-mcp
```

### Switching Profiles

Simply change the `DASH0_MCP_PROFILE` environment variable and restart:

```bash
# Use minimal profile for tight context windows
DASH0_MCP_PROFILE=minimal ./dash0-mcp

# Use full profile for complete access
DASH0_MCP_PROFILE=full ./dash0-mcp
```

## Available Tools

### Telemetry Query

| Tool | Description |
|------|-------------|
| `dash0_logs_query` | Query logs with filtering by service, severity, body text, time range. Returns markdown table with severity distribution, trace correlation %, top services/pods |
| `dash0_spans_query` | Query spans/traces with filtering by service, HTTP method, status code, min duration, errors. Returns markdown table with P95/avg/max latency, error rate, has_children, span kind, K8s pod |

### Telemetry Ingestion

| Tool | Description |
|------|-------------|
| `dash0_logs_send` | Send OTLP log records to Dash0 |
| `dash0_spans_send` | Send OTLP spans to Dash0 |

### Alerting

| Tool | Description |
|------|-------------|
| `dash0_alerting_check_rules_list` | List all check rules with formatted markdown table |
| `dash0_alerting_check_rules_get` | Get a specific check rule |
| `dash0_alerting_check_rules_create` | Create a new check rule |
| `dash0_alerting_check_rules_update` | Update an existing check rule |
| `dash0_alerting_check_rules_delete` | Delete a check rule |
| `dash0_alerting_active_alerts` | List currently firing and pending alerts with severity, duration, and labels |

### Dashboards

| Tool | Description |
|------|-------------|
| `dash0_dashboards_list` | List all dashboards |
| `dash0_dashboards_get` | Get a specific dashboard |
| `dash0_dashboards_create` | Create a new dashboard |
| `dash0_dashboards_update` | Update an existing dashboard |
| `dash0_dashboards_delete` | Delete a dashboard |

### Views

| Tool | Description |
|------|-------------|
| `dash0_views_list` | List all saved views |
| `dash0_views_get` | Get a specific view |
| `dash0_views_create` | Create a new view |
| `dash0_views_update` | Update an existing view |
| `dash0_views_delete` | Delete a view |

### Synthetic Checks

| Tool | Description |
|------|-------------|
| `dash0_synthetic_checks_list` | List all synthetic checks with kind, interval, locations, and URL |
| `dash0_synthetic_checks_get` | Get a specific synthetic check |
| `dash0_synthetic_checks_create` | Create a new synthetic check |
| `dash0_synthetic_checks_update` | Update an existing synthetic check |
| `dash0_synthetic_checks_delete` | Delete a synthetic check |

### Sampling Rules

| Tool | Description |
|------|-------------|
| `dash0_sampling_rules_list` | List all sampling rules |
| `dash0_sampling_rules_get` | Get a specific sampling rule |
| `dash0_sampling_rules_create` | Create a new sampling rule |
| `dash0_sampling_rules_update` | Update an existing sampling rule |
| `dash0_sampling_rules_delete` | Delete a sampling rule |

### Import

| Tool | Description |
|------|-------------|
| `dash0_import_check_rule` | Import a check rule from another platform |
| `dash0_import_dashboard` | Import a dashboard (e.g., from Grafana) |
| `dash0_import_synthetic_check` | Import a synthetic check |
| `dash0_import_view` | Import a saved view |

## Example Interactions

### Query Recent Logs

```
User: Show me error logs from the cart service in the last 15 minutes
Assistant: [Uses dash0_logs_query with service_name="cart", min_severity="ERROR", time_range_minutes=15]
```

### List Dashboards

```
User: Show me all dashboards in Dash0
Assistant: [Uses dash0_dashboards_list]
```

### Create an Alert Rule

```
User: Create an alert rule for high error rates
Assistant: [Uses dash0_alerting_check_rules_create with PromQL expression]
```

### Import a Grafana Dashboard

```
User: Import this Grafana dashboard JSON into Dash0
Assistant: [Uses dash0_import_dashboard with the provided JSON]
```

## API Regions

| Region | Base URL |
|--------|----------|
| EU West 1 (Ireland) | `https://api.eu-west-1.aws.dash0.com` |
| US East 1 (Virginia) | `https://api.us-east-1.aws.dash0.com` |
| US West 2 (Oregon) | `https://api.us-west-2.aws.dash0.com` |

## Output Format

All tools return **LLM-optimized markdown** instead of raw JSON. This dramatically improves AI assistant comprehension and response quality.

### Span Query Output

Span queries return a markdown table with columns: Service, Name, Kind, Duration, Status, HTTP, Pod, Children, Trace ID — plus a stats block:

```
## Span Query Results
**Found 42 spans** | Time: 14:30:00 → 15:30:00 2026-01-15 | Filters: service=cart-service

> **Stats:** Avg: 45.2ms | P95: 210.5ms | Max: 892.0ms | Error rate: 4.8% (2/42) | Services: cart-service (42) | Ops: GET /api/cart (18), POST /api/checkout (12)

| # | Service | Name | Kind | Duration | Status | HTTP | Pod | Children | Trace ID |
|---|---|---|---|---|---|---|---|---|---|
| 1 | cart-service | GET /api/cart | SERVER | 45.2ms | OK | GET 200 /api/cart | cart-pod-abc | yes | abc123def456... |
```

### Log Query Output

Log queries include severity distribution, trace correlation, and top pods:

```
> **Stats:** ERROR: 12 | WARN: 28 | INFO: 156 | Services: cart-service (120), api-gateway (76) | With traces: 85% | Pods: cart-pod-1 (80), cart-pod-2 (40)
```

### List Endpoints

All list endpoints (dashboards, views, sampling rules, etc.) return formatted tables with name, kind, and origin extracted from Kubernetes CRD metadata.

## Architecture

### Key Design Decisions

- **Markdown-first output**: Tools set `ToolResult.Markdown` for pre-formatted text; the MCP handler serves it directly, bypassing JSON marshaling. Write operations still return JSON.
- **OTLP flattening with field enrichment**: Span and log queries flatten nested OTLP structures and extract K8s metadata (pod name, namespace, container), span kind, event/link counts, and parent-child relationships
- **Summary statistics**: Span queries compute avg/P95/max duration, error rate, and top services/operations. Log queries compute severity distribution, trace correlation %, and top pods.
- **Shared OTLP types**: Common telemetry query types (`AttributeFilter`, `TimeRange`, `Pagination`) are defined once in `internal/otlp/` and shared by logs and spans packages
- **ToolProvider interface**: All 8 domain packages implement `registry.ToolProvider` with compile-time verification (`var _ registry.ToolProvider = (*Tools)(nil)`)
- **HTTP retry logic**: The client automatically retries on HTTP 429 (rate limit) and 503 (service unavailable) with exponential backoff and `Retry-After` header support
- **Structured logging**: Uses `log/slog` for leveled, structured log output (controlled by `DASH0_DEBUG`)
- **Graceful shutdown**: Handles `SIGINT`/`SIGTERM` for clean process termination
- **Input validation**: Query tools validate parameters (reject negative time ranges/limits, trim whitespace from string filters)
- **Dataset handling**: Dataset is passed as a query parameter on all API requests, and additionally in the request body for telemetry query endpoints

## Development

### Project Structure

```
dash0-mcp/
├── cmd/server/           # Main entry point
│   └── main.go           # Server bootstrap, slog setup, signal handling
├── internal/
│   ├── client/           # HTTP client for Dash0 API
│   │   └── client.go     # Request execution, retry logic, dataset handling
│   ├── config/           # Configuration management
│   │   ├── config.go     # Auth/region config + validation
│   │   └── tools.go      # Tool profile config
│   ├── formatter/        # Markdown output formatting
│   │   └── markdown.go   # Table rendering, duration formatting, list formatting
│   ├── otlp/             # Shared OpenTelemetry types
│   │   ├── types.go      # AttributeFilter, TimeRange, Pagination
│   │   └── extract.go    # ExtractServiceName helper
│   └── registry/         # Tool registry with filtering
│       └── registry.go   # Registry, ToolProvider interface
├── api/                  # MCP tool packages
│   ├── registry.go       # Unified tool registration
│   ├── provider.go       # ToolProvider type alias
│   ├── alerting/         # Check rules tools
│   ├── dashboards/       # Dashboard tools
│   ├── imports/          # Import tools
│   ├── logs/             # Log query/ingestion tools
│   ├── samplingrules/    # Sampling rules tools
│   ├── spans/            # Span query/ingestion tools
│   ├── syntheticchecks/  # Synthetic monitoring tools
│   └── views/            # View tools
├── config/               # Tool configuration
│   ├── tools.yaml        # Master tool definitions
│   └── profiles/         # Profile definitions
│       ├── full.yaml
│       ├── demo.yaml
│       ├── readonly.yaml
│       └── minimal.yaml
├── LICENSE
├── go.mod
├── go.sum
└── README.md
```

### Adding New Tools

1. Create a new package under `api/` or add to an existing one
2. Implement the package structure:
   - `Tools` struct with `client *client.Client`
   - `New(c *client.Client) *Tools` constructor
   - `Tools() []mcp.Tool` - Return tool definitions
   - `Handlers() map[string]func(...) *client.ToolResult` - Return handlers
   - `Register(reg *registry.Registry, c *client.Client)` - Register with registry
   - `var _ registry.ToolProvider = (*Tools)(nil)` - Compile-time interface check
3. Define a `basePath` constant for API endpoints
4. Call the Register function in `api/registry.go`
5. Add tool definitions to `config/tools.yaml`
6. Update profiles as needed

### Running Tests

```bash
go test ./...
```

### Test Coverage

Run tests with coverage report:

```bash
go test -cover ./...
```

Generate detailed HTML coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Building

```bash
go build -o dash0-mcp ./cmd/server
```

## License

MIT License - See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please read the contributing guidelines before submitting pull requests.

## Support

- [Dash0 Documentation](https://www.dash0.com/docs)
- [Dash0 API Reference](https://api-docs.dash0.com)
- [GitHub Issues](https://github.com/npcomplete777/Dash0-mcp/issues)
