package views

import (
	"context"
	"fmt"
	"net/url"

	"github.com/npcomplete777/dash0-mcp/internal/client"
	"github.com/npcomplete777/dash0-mcp/internal/formatter"
	"github.com/npcomplete777/dash0-mcp/internal/registry"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

const (
	basePath = "/api/views"
)

// Compile-time interface check.
var _ registry.ToolProvider = (*Tools)(nil)

// Tools provides MCP tools for Views API operations.
type Tools struct {
	client *client.Client
}

// New creates a new Views tools instance.
func New(c *client.Client) *Tools {
	return &Tools{client: c}
}

// Tools returns all MCP tools in this package.
func (p *Tools) Tools() []mcp.Tool {
	return []mcp.Tool{
		p.ListViews(),
		p.GetView(),
		p.CreateView(),
		p.UpdateView(),
		p.DeleteView(),
	}
}

// Handlers returns a map of tool name to handler function.
func (p *Tools) Handlers() map[string]func(context.Context, map[string]interface{}) *client.ToolResult {
	return map[string]func(context.Context, map[string]interface{}) *client.ToolResult{
		"dash0_views_list":   p.ListViewsHandler,
		"dash0_views_get":    p.GetViewHandler,
		"dash0_views_create": p.CreateViewHandler,
		"dash0_views_update": p.UpdateViewHandler,
		"dash0_views_delete": p.DeleteViewHandler,
	}
}

// ListViews returns the dash0_views_list tool definition.
func (p *Tools) ListViews() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_views_list",
		Description: "List all saved views in Dash0. Views are saved queries and filters for logs, traces, and metrics exploration.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
}

// ListViewsHandler handles the dash0_views_list tool.
func (p *Tools) ListViewsHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	result := p.client.Get(ctx, basePath)
	if result.Success {
		result.Markdown = formatter.FormatListResponse("Views", result.Data)
	}
	return result
}

// GetView returns the dash0_views_get tool definition.
func (p *Tools) GetView() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_views_get",
		Description: "Get a specific view by its origin or ID, including query configuration and filters.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the view to retrieve.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// GetViewHandler handles the dash0_views_get tool.
func (p *Tools) GetViewHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf(basePath+"/%s", url.PathEscape(originOrID))
	return p.client.Get(ctx, path)
}

// CreateView returns the dash0_views_create tool definition.
func (p *Tools) CreateView() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_views_create",
		Description: `Create a new saved view in Dash0 for quick access to commonly used queries and filters.

IMPORTANT: Views use Kubernetes CRD format (Dash0View).

Required structure:
- kind: Must be "Dash0View"
- metadata.name: View identifier (lowercase, alphanumeric, hyphens)
- spec.type: Must be "resources" (currently the only supported type)

Example body:
{
  "kind": "Dash0View",
  "metadata": {"name": "production-services"},
  "spec": {"type": "resources"}
}

Another example:
{
  "kind": "Dash0View",
  "metadata": {"name": "error-traces"},
  "spec": {"type": "resources"}
}`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The view configuration in Dash0View CRD format.",
					"properties": map[string]interface{}{
						"kind": map[string]interface{}{
							"type":        "string",
							"description": "Must be 'Dash0View'",
							"enum":        []string{"Dash0View"},
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "View metadata",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type":        "string",
									"description": "View identifier (lowercase, alphanumeric, hyphens)",
								},
							},
							"required": []interface{}{"name"},
						},
						"spec": map[string]interface{}{
							"type":        "object",
							"description": "View specification",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type":        "string",
									"description": "View type (currently only 'resources' is supported)",
									"enum":        []string{"resources"},
								},
							},
							"required": []interface{}{"type"},
						},
					},
					"required": []interface{}{"kind", "metadata", "spec"},
				},
			},
			Required: []string{"body"},
		},
	}
}

// CreateViewHandler handles the dash0_views_create tool.
func (p *Tools) CreateViewHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	return p.client.Post(ctx, basePath, body)
}

// UpdateView returns the dash0_views_update tool definition.
func (p *Tools) UpdateView() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_views_update",
		Description: `Update an existing view by its origin or ID.

The body should follow the same Dash0View CRD format as create:
{
  "kind": "Dash0View",
  "metadata": {"name": "updated-view"},
  "spec": {"type": "resources"}
}`,
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the view to update.",
				},
				"body": map[string]interface{}{
					"type":        "object",
					"description": "The updated view configuration in Dash0View CRD format.",
					"properties": map[string]interface{}{
						"kind": map[string]interface{}{
							"type":        "string",
							"description": "Must be 'Dash0View'",
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "View metadata with name",
						},
						"spec": map[string]interface{}{
							"type":        "object",
							"description": "View specification with type",
						},
					},
					"required": []interface{}{"kind", "metadata", "spec"},
				},
			},
			Required: []string{"origin_or_id", "body"},
		},
	}
}

// UpdateViewHandler handles the dash0_views_update tool.
func (p *Tools) UpdateViewHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	body, ok := args["body"]
	if !ok {
		return client.ErrorResult(400, "body is required")
	}

	path := fmt.Sprintf(basePath+"/%s", url.PathEscape(originOrID))
	return p.client.Put(ctx, path, body)
}

// DeleteView returns the dash0_views_delete tool definition.
func (p *Tools) DeleteView() mcp.Tool {
	return mcp.Tool{
		Name:        "dash0_views_delete",
		Description: "Delete a view by its origin or ID.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"origin_or_id": map[string]interface{}{
					"type":        "string",
					"description": "The origin or ID of the view to delete.",
				},
			},
			Required: []string{"origin_or_id"},
		},
	}
}

// DeleteViewHandler handles the dash0_views_delete tool.
func (p *Tools) DeleteViewHandler(ctx context.Context, args map[string]interface{}) *client.ToolResult {
	originOrID, ok := args["origin_or_id"].(string)
	if !ok || originOrID == "" {
		return client.ErrorResult(400, "origin_or_id is required")
	}

	path := fmt.Sprintf(basePath+"/%s", url.PathEscape(originOrID))
	return p.client.Delete(ctx, path)
}

// Register registers all views tools with the registry.
func Register(reg *registry.Registry, c *client.Client) {
	p := New(c)
	for _, tool := range p.Tools() {
		handler := p.Handlers()[tool.Name]
		reg.Register(tool, handler)
	}
}
