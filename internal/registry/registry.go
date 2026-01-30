// Package registry provides a tool registry with enable/disable filtering.
package registry

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

// Handler is the function signature for tool handlers.
type Handler func(ctx context.Context, args map[string]interface{}) *client.ToolResult

// ToolDef contains the complete definition of a tool.
type ToolDef struct {
	Tool    mcp.Tool
	Handler Handler
}

// Registry manages tool registration and enablement filtering.
type Registry struct {
	mu      sync.RWMutex
	tools   map[string]ToolDef
	enabled map[string]bool
}

// New creates a new Registry with the given enabled tools filter.
// If enabledTools is nil, all registered tools will be enabled.
func New(enabledTools map[string]bool) *Registry {
	return &Registry{
		tools:   make(map[string]ToolDef),
		enabled: enabledTools,
	}
}

// Register adds a tool to the registry.
// The tool will only be exposed if it's in the enabled set (or if no filter is set).
func (r *Registry) Register(tool mcp.Tool, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = ToolDef{
		Tool:    tool,
		Handler: handler,
	}
}

// IsEnabled checks if a tool is enabled.
func (r *Registry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If no filter set, all tools are enabled
	if r.enabled == nil {
		return true
	}
	return r.enabled[name]
}

// GetEnabledTools returns all enabled tool definitions for MCP listing.
func (r *Registry) GetEnabledTools() []mcp.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []mcp.Tool
	for name, def := range r.tools {
		if r.enabled == nil || r.enabled[name] {
			tools = append(tools, def.Tool)
		}
	}

	// Sort for consistent ordering
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools
}

// GetHandler returns the handler for a tool, or nil if not found.
func (r *Registry) GetHandler(name string) Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, exists := r.tools[name]
	if !exists {
		return nil
	}
	return def.Handler
}

// Call executes a tool handler if the tool exists and is enabled.
func (r *Registry) Call(ctx context.Context, name string, args map[string]interface{}) *client.ToolResult {
	r.mu.RLock()
	def, exists := r.tools[name]
	enabled := r.enabled == nil || r.enabled[name]
	r.mu.RUnlock()

	if !exists {
		return client.ErrorResult(404, fmt.Sprintf("tool %s not found", name))
	}
	if !enabled {
		return client.ErrorResult(403, fmt.Sprintf("tool %s is not enabled in current profile", name))
	}

	return def.Handler(ctx, args)
}

// ToolCount returns the total number of registered tools.
func (r *Registry) ToolCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// EnabledCount returns the number of enabled tools.
func (r *Registry) EnabledCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.enabled == nil {
		return len(r.tools)
	}

	count := 0
	for name := range r.tools {
		if r.enabled[name] {
			count++
		}
	}
	return count
}

// EnabledToolNames returns a sorted list of enabled tool names.
func (r *Registry) EnabledToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.tools {
		if r.enabled == nil || r.enabled[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// AllToolNames returns a sorted list of all registered tool names.
func (r *Registry) AllToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
