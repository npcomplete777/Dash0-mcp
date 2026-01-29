# Dash0 MCP Server

A Model Context Protocol (MCP) server for the Dash0 Observability Platform, enabling AI assistants to interact with Dash0's APIs for monitoring, alerting, and observability management.

## Features

- **Telemetry Ingestion**: Send OTLP logs and spans to Dash0
- **Dashboard Management**: Create, read, update, and delete dashboards
- **Alerting**: Manage check rules (Prometheus-style alert rules)
- **Views**: Save and manage query views for logs and traces
- **Synthetic Monitoring**: Configure synthetic checks for proactive monitoring
- **Sampling Rules**: Control data ingestion rates and costs
- **Migration**: Import configurations from other observability platforms

## Installation

### Build from Source

```bash
cd /path/to/dash0-mcp-server
go build -o dash0-mcp ./cmd/server
```

### Install with Go

```bash
go install github.com/ajacobs/dash0-mcp-server/cmd/server@latest
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DASH0_AUTH_TOKEN` | Yes | Bearer token for API authentication |
| `DASH0_REGION` | No | Region: `eu-west-1` (default) or `us-east-1` |
| `DASH0_BASE_URL` | No | Custom base URL (overrides region) |
| `DASH0_DEBUG` | No | Enable debug logging (`true`/`false`) |

### Obtaining an Auth Token

1. Log in to your Dash0 account
2. Navigate to Settings > API Tokens
3. Create a new token with appropriate permissions
4. Set the token as `DASH0_AUTH_TOKEN`

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
        "DASH0_REGION": "eu-west-1"
      }
    }
  }
}
```

### Running Manually

```bash
export DASH0_AUTH_TOKEN="your-auth-token"
export DASH0_REGION="eu-west-1"
./dash0-mcp
```

## Available Tools

### Telemetry

| Tool | Description |
|------|-------------|
| `dash0_logs_send` | Send OTLP log records to Dash0 |
| `dash0_spans_send` | Send OTLP spans to Dash0 |

### Alerting

| Tool | Description |
|------|-------------|
| `dash0_alerting_check_rules_list` | List all check rules |
| `dash0_alerting_check_rules_get` | Get a specific check rule |
| `dash0_alerting_check_rules_create` | Create a new check rule |
| `dash0_alerting_check_rules_update` | Update an existing check rule |
| `dash0_alerting_check_rules_delete` | Delete a check rule |

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
| `dash0_synthetic_checks_list` | List all synthetic checks |
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

## Development

### Project Structure

```
dash0-mcp-server/
├── cmd/server/           # Main entry point
│   └── main.go
├── internal/
│   ├── client/           # HTTP client for Dash0 API
│   │   └── client.go
│   └── config/           # Configuration management
│       └── config.go
├── api/                  # MCP tool packages
│   ├── registry.go       # Unified tool registry
│   ├── alerting/         # Check rules tools
│   ├── dashboards/       # Dashboard tools
│   ├── imports/          # Import tools
│   ├── logs/             # Log ingestion tools
│   ├── samplingrules/    # Sampling rules tools
│   ├── spans/            # Span ingestion tools
│   ├── syntheticchecks/  # Synthetic monitoring tools
│   └── views/            # View tools
├── go.mod
├── go.sum
└── README.md
```

### Adding New Tools

1. Create a new package under `api/` or add to an existing one
2. Implement the `toolsProvider` interface:
   - `Tools() []mcp.Tool` - Return tool definitions
   - `Handlers() map[string]func(...) *client.ToolResult` - Return handlers
3. Register the package in `api/registry.go`

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

#### Current Coverage

| Package | Coverage |
|---------|----------|
| `api/alerting` | 100.0% |
| `api/dashboards` | 100.0% |
| `api/imports` | 100.0% |
| `api/logs` | 93.0% |
| `api/samplingrules` | 100.0% |
| `api/spans` | 94.0% |
| `api/syntheticchecks` | 100.0% |
| `api/views` | 100.0% |
| `internal/client` | 96.2% |
| `internal/config` | 91.9% |

### Building

```bash
go build -o dash0-mcp ./cmd/server
```

## License

MIT License - See LICENSE file for details.

## Contributing

Contributions are welcome! Please read the contributing guidelines before submitting pull requests.

## Support

- [Dash0 Documentation](https://www.dash0.com/docs)
- [Dash0 API Reference](https://api-docs.dash0.com)
- [GitHub Issues](https://github.com/ajacobs/dash0-mcp-server/issues)
