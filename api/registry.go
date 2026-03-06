// Package api provides the unified registry for all Dash0 MCP tools.
package api

import (
	"github.com/npcomplete777/dash0-mcp/internal/client"
	"github.com/npcomplete777/dash0-mcp/internal/registry"

	"github.com/npcomplete777/dash0-mcp/api/alerting"
	"github.com/npcomplete777/dash0-mcp/api/dashboards"
	"github.com/npcomplete777/dash0-mcp/api/imports"
	"github.com/npcomplete777/dash0-mcp/api/logs"
	"github.com/npcomplete777/dash0-mcp/api/samplingrules"
	"github.com/npcomplete777/dash0-mcp/api/spans"
	"github.com/npcomplete777/dash0-mcp/api/syntheticchecks"
	"github.com/npcomplete777/dash0-mcp/api/views"
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
