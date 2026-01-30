// Package api provides the unified registry for all Dash0 MCP tools.
package api

import (
	"github.com/ajacobs/dash0-mcp-server/internal/client"
	"github.com/ajacobs/dash0-mcp-server/internal/registry"

	"github.com/ajacobs/dash0-mcp-server/api/alerting"
	"github.com/ajacobs/dash0-mcp-server/api/dashboards"
	"github.com/ajacobs/dash0-mcp-server/api/imports"
	"github.com/ajacobs/dash0-mcp-server/api/logs"
	"github.com/ajacobs/dash0-mcp-server/api/samplingrules"
	"github.com/ajacobs/dash0-mcp-server/api/spans"
	"github.com/ajacobs/dash0-mcp-server/api/syntheticchecks"
	"github.com/ajacobs/dash0-mcp-server/api/views"
)

// RegisterAllTools registers all tool handlers with the registry.
// All handlers are registered, but only enabled tools are exposed.
func RegisterAllTools(reg *registry.Registry, c *client.Client) {
	// Telemetry data ingestion
	logs.Register(reg, c)
	spans.Register(reg, c)

	// Configuration management
	alerting.Register(reg, c)
	dashboards.Register(reg, c)
	views.Register(reg, c)
	syntheticchecks.Register(reg, c)
	samplingrules.Register(reg, c)

	// Migration/import
	imports.Register(reg, c)
}
