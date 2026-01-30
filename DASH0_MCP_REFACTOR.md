# Dash0 MCP Refactor: Config-Driven Tool Enablement

## Overview

Refactor the Dash0 MCP server to support dynamic tool enablement via YAML configuration. This follows the same pattern implemented for the GitHub MCP server - enabling/disabling tools without code changes.

## Current Dash0 MCP Structure

Based on previous conversations, the Dash0 MCP server has this structure:

```
dash0-mcp-server/
├── cmd/server/
│   └── main.go              # Entry point
├── internal/
│   ├── client/
│   │   └── client.go        # HTTP client for Dash0 API
│   └── config/
│       └── config.go        # Configuration management
├── api/
│   ├── registry.go          # Unified tool registry
│   ├── alerting/            # Check rules tools (5 tools)
│   ├── dashboards/          # Dashboard tools (5 tools)
│   ├── imports/             # Import tools (4 tools)
│   ├── logs/                # Log query/ingestion tools (2 tools)
│   ├── samplingrules/       # Sampling rules tools (5 tools)
│   ├── spans/               # Span query/ingestion tools (2 tools)
│   ├── syntheticchecks/     # Synthetic monitoring tools (5 tools)
│   └── views/               # View tools (5 tools)
├── go.mod
├── go.sum
└── README.md
```

## Current Tools Inventory (~33 tools)

| Category | Tools | Count |
|----------|-------|-------|
| **Dashboards** | list, get, create, update, delete | 5 |
| **Check Rules** | list, get, create, update, delete | 5 |
| **Synthetic Checks** | list, get, create, update, delete | 5 |
| **Sampling Rules** | list, get, create, update, delete | 5 |
| **Views** | list, get, create, update, delete | 5 |
| **Logs** | query, send | 2 |
| **Spans** | query, send | 2 |
| **Import** | dashboard, check_rule, synthetic_check, view | 4 |
| **Total** | | **33** |

## Target Architecture

```
dash0-mcp-server/
├── cmd/server/
│   └── main.go                    # Entry point, loads config
├── internal/
│   ├── client/
│   │   └── client.go              # HTTP client (unchanged)
│   ├── config/
│   │   ├── config.go              # App config (auth, region)
│   │   └── tools.go               # NEW: Tool config loading
│   └── registry/
│       └── registry.go            # NEW: Tool registry with enable/disable
├── api/
│   ├── registry.go                # MODIFY: Use new registry pattern
│   ├── alerting/                  # Unchanged
│   ├── dashboards/                # Unchanged
│   ├── imports/                   # Unchanged
│   ├── logs/                      # Unchanged
│   ├── samplingrules/             # Unchanged
│   ├── spans/                     # Unchanged
│   ├── syntheticchecks/           # Unchanged
│   └── views/                     # Unchanged
├── config/
│   ├── tools.yaml                 # NEW: Master tool definitions
│   └── profiles/
│       ├── full.yaml              # NEW: All tools enabled
│       ├── demo.yaml              # NEW: Demo workflow tools
│       ├── readonly.yaml          # NEW: Read-only tools
│       └── minimal.yaml           # NEW: Bare minimum
├── go.mod
├── go.sum
└── README.md
```

## Implementation Steps

### Step 1: Create Tool Configuration Schema

Create `internal/config/tools.go`:

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

// ToolDef defines a single tool's configuration
type ToolDef struct {
    Enabled     bool   `yaml:"enabled"`
    Description string `yaml:"description"`
    Dangerous   bool   `yaml:"dangerous"`
}

// ToolsConfig holds all tool definitions
type ToolsConfig struct {
    Version        string                         `yaml:"version"`
    DefaultProfile string                         `yaml:"default_profile"`
    Settings       ToolsSettings                  `yaml:"settings"`
    Tools          map[string]map[string]ToolDef  `yaml:"tools"`
}

// ToolsSettings global settings for tool management
type ToolsSettings struct {
    LogEnabledTools bool `yaml:"log_enabled_tools"`
    StrictMode      bool `yaml:"strict_mode"`
}

// Profile defines a tool enablement profile
type Profile struct {
    Name            string   `yaml:"name"`
    Description     string   `yaml:"description"`
    Enable          []string `yaml:"enable"`
    Disable         []string `yaml:"disable"`
    EnableAll       bool     `yaml:"enable_all"`
    DisableUnlisted bool     `yaml:"disable_unlisted"`
}

// LoadToolsConfig loads tools.yaml and profile
func LoadToolsConfig(configDir, profileName string) (*ToolsConfig, *Profile, error) {
    // Load master tools.yaml
    toolsPath := filepath.Join(configDir, "tools.yaml")
    toolsData, err := os.ReadFile(toolsPath)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to read tools.yaml: %w", err)
    }

    var toolsConfig ToolsConfig
    if err := yaml.Unmarshal(toolsData, &toolsConfig); err != nil {
        return nil, nil, fmt.Errorf("failed to parse tools.yaml: %w", err)
    }

    // Determine profile
    if profileName == "" {
        profileName = os.Getenv("DASH0_MCP_PROFILE")
    }
    if profileName == "" {
        profileName = toolsConfig.DefaultProfile
    }
    if profileName == "" {
        profileName = "full"
    }

    // Load profile
    profilePath := filepath.Join(configDir, "profiles", profileName+".yaml")
    var profile Profile

    if profileData, err := os.ReadFile(profilePath); err == nil {
        if err := yaml.Unmarshal(profileData, &profile); err != nil {
            return nil, nil, fmt.Errorf("failed to parse profile %s: %w", profileName, err)
        }
    }

    return &toolsConfig, &profile, nil
}

// GetEnabledTools returns list of tool names that should be registered
func GetEnabledTools(tc *ToolsConfig, p *Profile) map[string]bool {
    enabled := make(map[string]bool)

    // Build profile override sets
    profileEnabled := make(map[string]bool)
    profileDisabled := make(map[string]bool)

    for _, t := range p.Enable {
        profileEnabled[t] = true
    }
    for _, t := range p.Disable {
        profileDisabled[t] = true
    }

    // Evaluate each tool
    for _, tools := range tc.Tools {
        for toolName, toolDef := range tools {
            shouldEnable := false

            if p.EnableAll {
                shouldEnable = !profileDisabled[toolName]
            } else if p.DisableUnlisted {
                shouldEnable = profileEnabled[toolName]
            } else {
                shouldEnable = toolDef.Enabled
                if profileEnabled[toolName] {
                    shouldEnable = true
                }
                if profileDisabled[toolName] {
                    shouldEnable = false
                }
            }

            if shouldEnable {
                enabled[toolName] = true
            }
        }
    }

    return enabled
}
```

### Step 2: Create Tool Registry

Create `internal/registry/registry.go`:

```go
package registry

import (
    "context"
    "fmt"
    "log"
    "sync"

    "github.com/mark3labs/mcp-go/mcp"
)

// Handler is the function signature for tool handlers
type Handler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// ToolDef contains metadata and handler for a tool
type ToolDef struct {
    Name        string
    Description string
    InputSchema map[string]interface{}
    Handler     Handler
}

// Registry manages tool registration and enablement
type Registry struct {
    mu       sync.RWMutex
    tools    map[string]ToolDef
    enabled  map[string]bool
}

// New creates a new Registry
func New(enabledTools map[string]bool) *Registry {
    return &Registry{
        tools:   make(map[string]ToolDef),
        enabled: enabledTools,
    }
}

// Register adds a tool to the registry
func (r *Registry) Register(def ToolDef) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[def.Name] = def
}

// IsEnabled checks if a tool is enabled
func (r *Registry) IsEnabled(name string) bool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.enabled[name]
}

// GetEnabledTools returns all enabled tool definitions for MCP listing
func (r *Registry) GetEnabledTools() []mcp.Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var tools []mcp.Tool
    for name, def := range r.tools {
        if r.enabled[name] {
            tools = append(tools, mcp.Tool{
                Name:        def.Name,
                Description: def.Description,
                InputSchema: mcp.ToolInputSchema{
                    Type:       "object",
                    Properties: def.InputSchema,
                },
            })
        }
    }
    return tools
}

// Call executes a tool if enabled
func (r *Registry) Call(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
    r.mu.RLock()
    def, exists := r.tools[name]
    enabled := r.enabled[name]
    r.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("tool %s not found", name)
    }
    if !enabled {
        return nil, fmt.Errorf("tool %s is not enabled in current profile", name)
    }

    return def.Handler(ctx, args)
}

// LogEnabled logs which tools are enabled at startup
func (r *Registry) LogEnabled() {
    r.mu.RLock()
    defer r.mu.RUnlock()

    count := 0
    for name := range r.tools {
        if r.enabled[name] {
            count++
            log.Printf("  ✓ %s", name)
        }
    }
    log.Printf("Enabled %d/%d tools", count, len(r.tools))
}
```

### Step 3: Modify API Registry

Update `api/registry.go` to use the new pattern:

```go
package api

import (
    "github.com/dash0/dash0-mcp-server/api/alerting"
    "github.com/dash0/dash0-mcp-server/api/dashboards"
    "github.com/dash0/dash0-mcp-server/api/imports"
    "github.com/dash0/dash0-mcp-server/api/logs"
    "github.com/dash0/dash0-mcp-server/api/samplingrules"
    "github.com/dash0/dash0-mcp-server/api/spans"
    "github.com/dash0/dash0-mcp-server/api/syntheticchecks"
    "github.com/dash0/dash0-mcp-server/api/views"
    "github.com/dash0/dash0-mcp-server/internal/client"
    "github.com/dash0/dash0-mcp-server/internal/registry"
)

// RegisterAllTools registers all tool handlers with the registry
// All handlers are registered, but only enabled tools are exposed
func RegisterAllTools(reg *registry.Registry, c *client.Client) {
    // Dashboard tools
    dashboards.Register(reg, c)
    
    // Alerting/Check Rules tools
    alerting.Register(reg, c)
    
    // Synthetic Checks tools
    syntheticchecks.Register(reg, c)
    
    // Sampling Rules tools
    samplingrules.Register(reg, c)
    
    // Views tools
    views.Register(reg, c)
    
    // Logs tools
    logs.Register(reg, c)
    
    // Spans tools
    spans.Register(reg, c)
    
    // Import tools
    imports.Register(reg, c)
}
```

### Step 4: Update Each API Package

Each package under `api/` needs a `Register` function. Example for `api/dashboards/register.go`:

```go
package dashboards

import (
    "github.com/dash0/dash0-mcp-server/internal/client"
    "github.com/dash0/dash0-mcp-server/internal/registry"
)

// Register registers all dashboard tools with the registry
func Register(reg *registry.Registry, c *client.Client) {
    p := NewPackage(c)

    reg.Register(registry.ToolDef{
        Name:        "dash0_dashboards_list",
        Description: "List all dashboards in Dash0",
        InputSchema: map[string]interface{}{},
        Handler:     p.List,
    })

    reg.Register(registry.ToolDef{
        Name:        "dash0_dashboards_get",
        Description: "Get a specific dashboard by ID",
        InputSchema: map[string]interface{}{
            "origin_or_id": map[string]interface{}{
                "type":        "string",
                "description": "Dashboard ID or origin reference",
            },
        },
        Handler: p.Get,
    })

    reg.Register(registry.ToolDef{
        Name:        "dash0_dashboards_create",
        Description: "Create a new Perses dashboard",
        InputSchema: map[string]interface{}{
            "body": map[string]interface{}{
                "type":        "object",
                "description": "Perses dashboard JSON specification",
            },
        },
        Handler: p.Create,
    })

    reg.Register(registry.ToolDef{
        Name:        "dash0_dashboards_update",
        Description: "Update an existing dashboard",
        InputSchema: map[string]interface{}{
            "origin_or_id": map[string]interface{}{
                "type":        "string",
                "description": "Dashboard ID or origin reference",
            },
            "body": map[string]interface{}{
                "type":        "object",
                "description": "Updated Perses dashboard JSON",
            },
        },
        Handler: p.Update,
    })

    reg.Register(registry.ToolDef{
        Name:        "dash0_dashboards_delete",
        Description: "Delete a dashboard",
        InputSchema: map[string]interface{}{
            "origin_or_id": map[string]interface{}{
                "type":        "string",
                "description": "Dashboard ID or origin reference",
            },
        },
        Handler: p.Delete,
    })
}
```

### Step 5: Update Main Entry Point

Update `cmd/server/main.go`:

```go
package main

import (
    "log"
    "os"
    "path/filepath"

    "github.com/dash0/dash0-mcp-server/api"
    "github.com/dash0/dash0-mcp-server/internal/client"
    "github.com/dash0/dash0-mcp-server/internal/config"
    "github.com/dash0/dash0-mcp-server/internal/registry"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    // Load app config (auth, region)
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Determine config directory for tools
    configDir := os.Getenv("DASH0_MCP_CONFIG_DIR")
    if configDir == "" {
        exe, _ := os.Executable()
        configDir = filepath.Join(filepath.Dir(exe), "config")
    }

    // Load tools config and profile
    profileName := os.Getenv("DASH0_MCP_PROFILE")
    toolsConfig, profile, err := config.LoadToolsConfig(configDir, profileName)
    if err != nil {
        log.Printf("Warning: Could not load tools config: %v", err)
        log.Printf("Using default: all tools enabled")
    }

    // Determine enabled tools
    var enabledTools map[string]bool
    if toolsConfig != nil && profile != nil {
        enabledTools = config.GetEnabledTools(toolsConfig, profile)
        log.Printf("Loaded profile: %s (%s)", profile.Name, profile.Description)
    } else {
        // Default: enable all tools
        enabledTools = make(map[string]bool)
        // Will be populated by registry with all tools enabled
    }

    // Create HTTP client
    httpClient := client.New(cfg)

    // Create registry with enabled tools filter
    reg := registry.New(enabledTools)

    // Register ALL tool handlers (registry filters by enabled)
    api.RegisterAllTools(reg, httpClient)

    // Log enabled tools
    if toolsConfig != nil && toolsConfig.Settings.LogEnabledTools {
        log.Println("Enabled tools:")
        reg.LogEnabled()
    }

    // Create MCP server
    mcpServer := server.NewMCPServer(
        "dash0-mcp",
        "1.0.0",
        server.WithToolCapabilities(true),
    )

    // Register enabled tools with MCP
    for _, tool := range reg.GetEnabledTools() {
        mcpServer.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            result, err := reg.Call(ctx, req.Params.Name, req.Params.Arguments)
            if err != nil {
                return mcp.NewToolResultError(err.Error()), nil
            }
            return mcp.NewToolResultText(formatJSON(result)), nil
        })
    }

    // Start stdio server
    if err := server.ServeStdio(mcpServer); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

---

## Configuration Files

### config/tools.yaml

```yaml
# Dash0 MCP Tool Configuration
version: "1.0"
default_profile: full

settings:
  log_enabled_tools: true
  strict_mode: true

tools:
  #############################################################################
  # DASHBOARD OPERATIONS
  #############################################################################
  dashboards:
    dash0_dashboards_list:
      enabled: true
      description: "List all dashboards in Dash0"
      dangerous: false

    dash0_dashboards_get:
      enabled: true
      description: "Get a specific dashboard by ID"
      dangerous: false

    dash0_dashboards_create:
      enabled: true
      description: "Create a new Perses dashboard"
      dangerous: false

    dash0_dashboards_update:
      enabled: true
      description: "Update an existing dashboard"
      dangerous: false

    dash0_dashboards_delete:
      enabled: false
      description: "Delete a dashboard (DESTRUCTIVE)"
      dangerous: true

  #############################################################################
  # CHECK RULES (ALERTING)
  #############################################################################
  alerting:
    dash0_alerting_check_rules_list:
      enabled: true
      description: "List all Prometheus-style check rules"
      dangerous: false

    dash0_alerting_check_rules_get:
      enabled: true
      description: "Get a specific check rule by ID"
      dangerous: false

    dash0_alerting_check_rules_create:
      enabled: true
      description: "Create a new check rule (PromQL-based alert)"
      dangerous: false

    dash0_alerting_check_rules_update:
      enabled: true
      description: "Update an existing check rule"
      dangerous: false

    dash0_alerting_check_rules_delete:
      enabled: false
      description: "Delete a check rule (DESTRUCTIVE)"
      dangerous: true

  #############################################################################
  # SYNTHETIC CHECKS
  #############################################################################
  syntheticchecks:
    dash0_synthetic_checks_list:
      enabled: true
      description: "List all synthetic monitoring checks"
      dangerous: false

    dash0_synthetic_checks_get:
      enabled: true
      description: "Get a specific synthetic check"
      dangerous: false

    dash0_synthetic_checks_create:
      enabled: true
      description: "Create a new synthetic check (HTTP, browser, etc.)"
      dangerous: false

    dash0_synthetic_checks_update:
      enabled: true
      description: "Update a synthetic check"
      dangerous: false

    dash0_synthetic_checks_delete:
      enabled: false
      description: "Delete a synthetic check (DESTRUCTIVE)"
      dangerous: true

  #############################################################################
  # SAMPLING RULES
  #############################################################################
  samplingrules:
    dash0_sampling_rules_list:
      enabled: true
      description: "List all sampling rules"
      dangerous: false

    dash0_sampling_rules_get:
      enabled: true
      description: "Get a specific sampling rule"
      dangerous: false

    dash0_sampling_rules_create:
      enabled: true
      description: "Create a sampling rule to control data ingestion"
      dangerous: false

    dash0_sampling_rules_update:
      enabled: true
      description: "Update a sampling rule"
      dangerous: false

    dash0_sampling_rules_delete:
      enabled: false
      description: "Delete a sampling rule (DESTRUCTIVE)"
      dangerous: true

  #############################################################################
  # VIEWS (SAVED QUERIES)
  #############################################################################
  views:
    dash0_views_list:
      enabled: true
      description: "List all saved views"
      dangerous: false

    dash0_views_get:
      enabled: true
      description: "Get a specific saved view"
      dangerous: false

    dash0_views_create:
      enabled: true
      description: "Create a saved view with filters and columns"
      dangerous: false

    dash0_views_update:
      enabled: true
      description: "Update a saved view"
      dangerous: false

    dash0_views_delete:
      enabled: false
      description: "Delete a saved view (DESTRUCTIVE)"
      dangerous: true

  #############################################################################
  # TELEMETRY - LOGS
  #############################################################################
  logs:
    dash0_logs_query:
      enabled: true
      description: "Query logs from Dash0 with filtering"
      dangerous: false

    dash0_logs_send:
      enabled: true
      description: "Send OTLP logs to Dash0"
      dangerous: false

  #############################################################################
  # TELEMETRY - SPANS
  #############################################################################
  spans:
    dash0_spans_query:
      enabled: true
      description: "Query spans/traces from Dash0 with filtering"
      dangerous: false

    dash0_spans_send:
      enabled: true
      description: "Send OTLP spans to Dash0"
      dangerous: false

  #############################################################################
  # IMPORT TOOLS
  #############################################################################
  imports:
    dash0_import_dashboard:
      enabled: true
      description: "Import a Grafana dashboard into Dash0"
      dangerous: false

    dash0_import_check_rule:
      enabled: true
      description: "Import a Prometheus alerting rule"
      dangerous: false

    dash0_import_synthetic_check:
      enabled: true
      description: "Import a synthetic check configuration"
      dangerous: false

    dash0_import_view:
      enabled: true
      description: "Import a saved view configuration"
      dangerous: false
```

### config/profiles/full.yaml

```yaml
# Full Profile: All Dash0 Tools
name: full
description: "Full Dash0 API coverage (33 tools)"

enable_all: true

# Disable only destructive operations by default
disable:
  - dash0_dashboards_delete
  - dash0_alerting_check_rules_delete
  - dash0_synthetic_checks_delete
  - dash0_sampling_rules_delete
  - dash0_views_delete
```

### config/profiles/demo.yaml

```yaml
# Demo Profile: Observability Workflow
#
# Enables tools for demonstrating:
# - Dashboard creation and viewing
# - Alert rule configuration
# - Synthetic check setup
# - Trace/log querying
#
# Tool count: 20

name: demo
description: "Observability demo workflow"

enable:
  # Dashboards - full CRUD except delete
  - dash0_dashboards_list
  - dash0_dashboards_get
  - dash0_dashboards_create
  - dash0_dashboards_update

  # Alerting - full CRUD except delete
  - dash0_alerting_check_rules_list
  - dash0_alerting_check_rules_get
  - dash0_alerting_check_rules_create
  - dash0_alerting_check_rules_update

  # Synthetic Checks - full CRUD except delete
  - dash0_synthetic_checks_list
  - dash0_synthetic_checks_get
  - dash0_synthetic_checks_create
  - dash0_synthetic_checks_update

  # Telemetry Query
  - dash0_logs_query
  - dash0_spans_query

  # Import (for Grafana migration demos)
  - dash0_import_dashboard
  - dash0_import_check_rule

  # Views - just list and get
  - dash0_views_list
  - dash0_views_get
  - dash0_views_create

disable_unlisted: true
```

### config/profiles/readonly.yaml

```yaml
# Read-Only Profile: Safe Exploration
name: readonly
description: "Read-only Dash0 access (13 tools)"

enable:
  - dash0_dashboards_list
  - dash0_dashboards_get
  - dash0_alerting_check_rules_list
  - dash0_alerting_check_rules_get
  - dash0_synthetic_checks_list
  - dash0_synthetic_checks_get
  - dash0_sampling_rules_list
  - dash0_sampling_rules_get
  - dash0_views_list
  - dash0_views_get
  - dash0_logs_query
  - dash0_spans_query

disable_unlisted: true
```

### config/profiles/minimal.yaml

```yaml
# Minimal Profile: Core Operations Only
name: minimal
description: "Minimal Dash0 tools for tight context (8 tools)"

enable:
  # Query only
  - dash0_logs_query
  - dash0_spans_query
  
  # Dashboards - list and view
  - dash0_dashboards_list
  - dash0_dashboards_get
  
  # Alerts - list and view
  - dash0_alerting_check_rules_list
  - dash0_alerting_check_rules_get
  
  # Synthetics - list and view
  - dash0_synthetic_checks_list
  - dash0_synthetic_checks_get

disable_unlisted: true
```

---

## Claude Desktop Configuration

Update `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "dash0": {
      "command": "/Users/aaronjacobs/mcp/src/domains/observability/dash0-mcp-server/dash0-mcp",
      "env": {
        "DASH0_AUTH_TOKEN": "auth_xxx",
        "DASH0_REGION": "us-west-2",
        "DASH0_MCP_PROFILE": "demo",
        "DASH0_MCP_CONFIG_DIR": "/Users/aaronjacobs/mcp/src/domains/observability/dash0-mcp-server/config"
      }
    }
  }
}
```

---

## Testing

```bash
# Build
cd ~/mcp/src/domains/observability/dash0-mcp-server
go build -o dash0-mcp ./cmd/server

# Test tool listing with demo profile
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
  DASH0_AUTH_TOKEN=xxx DASH0_REGION=us-west-2 DASH0_MCP_PROFILE=demo ./dash0-mcp | \
  jq '.result.tools | length'
# Expected: 20

# Test with full profile
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
  DASH0_AUTH_TOKEN=xxx DASH0_REGION=us-west-2 DASH0_MCP_PROFILE=full ./dash0-mcp | \
  jq '.result.tools | length'
# Expected: ~28 (full minus dangerous deletes)

# Test with readonly profile
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
  DASH0_AUTH_TOKEN=xxx DASH0_REGION=us-west-2 DASH0_MCP_PROFILE=readonly ./dash0-mcp | \
  jq '.result.tools | length'
# Expected: 13
```

---

## Implementation Checklist for Claude Code

1. [ ] Read existing codebase structure in `~/mcp/src/domains/observability/dash0-mcp-server/`
2. [ ] Create `internal/config/tools.go` with YAML loading logic
3. [ ] Create `internal/registry/registry.go` with enable/disable filtering
4. [ ] Update each `api/*` package to use `Register(reg, client)` pattern
5. [ ] Update `api/registry.go` to call all package Register functions
6. [ ] Update `cmd/server/main.go` to load config and filter tools
7. [ ] Create `config/tools.yaml` master configuration
8. [ ] Create `config/profiles/*.yaml` profile files
9. [ ] Build and test with different profiles
10. [ ] Update README.md with profile documentation

---

## Benefits

1. **No rebuild to change tools** - edit YAML, restart MCP
2. **Profile switching** - `DASH0_MCP_PROFILE=demo` vs `full` vs `readonly`
3. **Safe defaults** - dangerous delete tools disabled by default
4. **Self-documenting** - YAML shows all available tools and their purpose
5. **Consistent pattern** - matches GitHub MCP refactor for unified approach
6. **Context optimization** - use `minimal` profile when context is tight
