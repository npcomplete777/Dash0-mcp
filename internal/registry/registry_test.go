package registry

import (
	"context"
	"testing"

	"github.com/ajacobs/dash0-mcp-server/internal/client"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

func TestNew(t *testing.T) {
	t.Run("WithEnabledTools", func(t *testing.T) {
		enabled := map[string]bool{"tool1": true, "tool2": true}
		reg := New(enabled)
		if reg == nil {
			t.Fatal("expected registry, got nil")
		}
		if reg.enabled == nil {
			t.Error("expected enabled map to be set")
		}
	})

	t.Run("WithNilEnabledTools", func(t *testing.T) {
		reg := New(nil)
		if reg == nil {
			t.Fatal("expected registry, got nil")
		}
		if reg.enabled != nil {
			t.Error("expected enabled map to be nil")
		}
	})
}

func TestRegister(t *testing.T) {
	reg := New(nil)

	tool := mcp.Tool{
		Name:        "test_tool",
		Description: "A test tool",
	}
	handler := func(ctx context.Context, args map[string]interface{}) *client.ToolResult {
		return &client.ToolResult{Success: true}
	}

	reg.Register(tool, handler)

	if len(reg.tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(reg.tools))
	}
	if _, ok := reg.tools["test_tool"]; !ok {
		t.Error("expected test_tool to be registered")
	}
}

func TestIsEnabled(t *testing.T) {
	t.Run("NilFilter", func(t *testing.T) {
		reg := New(nil)
		reg.Register(mcp.Tool{Name: "tool1"}, nil)

		if !reg.IsEnabled("tool1") {
			t.Error("expected tool1 to be enabled with nil filter")
		}
		if !reg.IsEnabled("unregistered") {
			t.Error("expected unregistered tool to be 'enabled' with nil filter")
		}
	})

	t.Run("WithFilter", func(t *testing.T) {
		enabled := map[string]bool{"tool1": true}
		reg := New(enabled)

		if !reg.IsEnabled("tool1") {
			t.Error("expected tool1 to be enabled")
		}
		if reg.IsEnabled("tool2") {
			t.Error("expected tool2 to be disabled")
		}
	})
}

func TestGetEnabledTools(t *testing.T) {
	t.Run("NilFilter", func(t *testing.T) {
		reg := New(nil)
		reg.Register(mcp.Tool{Name: "tool1"}, nil)
		reg.Register(mcp.Tool{Name: "tool2"}, nil)

		tools := reg.GetEnabledTools()
		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}
	})

	t.Run("WithFilter", func(t *testing.T) {
		enabled := map[string]bool{"tool1": true}
		reg := New(enabled)
		reg.Register(mcp.Tool{Name: "tool1"}, nil)
		reg.Register(mcp.Tool{Name: "tool2"}, nil)

		tools := reg.GetEnabledTools()
		if len(tools) != 1 {
			t.Errorf("expected 1 enabled tool, got %d", len(tools))
		}
		if tools[0].Name != "tool1" {
			t.Errorf("expected tool1, got %s", tools[0].Name)
		}
	})

	t.Run("SortedOutput", func(t *testing.T) {
		reg := New(nil)
		reg.Register(mcp.Tool{Name: "zebra"}, nil)
		reg.Register(mcp.Tool{Name: "alpha"}, nil)
		reg.Register(mcp.Tool{Name: "beta"}, nil)

		tools := reg.GetEnabledTools()
		if tools[0].Name != "alpha" {
			t.Errorf("expected first tool to be 'alpha', got '%s'", tools[0].Name)
		}
		if tools[1].Name != "beta" {
			t.Errorf("expected second tool to be 'beta', got '%s'", tools[1].Name)
		}
		if tools[2].Name != "zebra" {
			t.Errorf("expected third tool to be 'zebra', got '%s'", tools[2].Name)
		}
	})
}

func TestGetHandler(t *testing.T) {
	reg := New(nil)

	handler := func(ctx context.Context, args map[string]interface{}) *client.ToolResult {
		return &client.ToolResult{Success: true, Data: "test"}
	}
	reg.Register(mcp.Tool{Name: "tool1"}, handler)

	t.Run("ExistingTool", func(t *testing.T) {
		h := reg.GetHandler("tool1")
		if h == nil {
			t.Fatal("expected handler, got nil")
		}
		result := h(context.Background(), nil)
		if !result.Success {
			t.Error("expected success")
		}
	})

	t.Run("NonexistentTool", func(t *testing.T) {
		h := reg.GetHandler("nonexistent")
		if h != nil {
			t.Error("expected nil handler for nonexistent tool")
		}
	})
}

func TestCall(t *testing.T) {
	enabled := map[string]bool{"tool1": true}
	reg := New(enabled)

	handler := func(ctx context.Context, args map[string]interface{}) *client.ToolResult {
		return &client.ToolResult{Success: true, Data: args["input"]}
	}
	reg.Register(mcp.Tool{Name: "tool1"}, handler)
	reg.Register(mcp.Tool{Name: "tool2"}, handler)

	t.Run("EnabledTool", func(t *testing.T) {
		result := reg.Call(context.Background(), "tool1", map[string]interface{}{"input": "test"})
		if !result.Success {
			t.Error("expected success")
		}
		if result.Data != "test" {
			t.Errorf("expected 'test', got %v", result.Data)
		}
	})

	t.Run("DisabledTool", func(t *testing.T) {
		result := reg.Call(context.Background(), "tool2", nil)
		if result.Success {
			t.Error("expected failure for disabled tool")
		}
		if result.Error == nil {
			t.Error("expected error for disabled tool")
		}
	})

	t.Run("NonexistentTool", func(t *testing.T) {
		result := reg.Call(context.Background(), "nonexistent", nil)
		if result.Success {
			t.Error("expected failure for nonexistent tool")
		}
	})
}

func TestToolCount(t *testing.T) {
	reg := New(nil)
	reg.Register(mcp.Tool{Name: "tool1"}, nil)
	reg.Register(mcp.Tool{Name: "tool2"}, nil)

	if reg.ToolCount() != 2 {
		t.Errorf("expected 2 tools, got %d", reg.ToolCount())
	}
}

func TestEnabledCount(t *testing.T) {
	t.Run("NilFilter", func(t *testing.T) {
		reg := New(nil)
		reg.Register(mcp.Tool{Name: "tool1"}, nil)
		reg.Register(mcp.Tool{Name: "tool2"}, nil)

		if reg.EnabledCount() != 2 {
			t.Errorf("expected 2 enabled, got %d", reg.EnabledCount())
		}
	})

	t.Run("WithFilter", func(t *testing.T) {
		enabled := map[string]bool{"tool1": true}
		reg := New(enabled)
		reg.Register(mcp.Tool{Name: "tool1"}, nil)
		reg.Register(mcp.Tool{Name: "tool2"}, nil)

		if reg.EnabledCount() != 1 {
			t.Errorf("expected 1 enabled, got %d", reg.EnabledCount())
		}
	})
}

func TestEnabledToolNames(t *testing.T) {
	enabled := map[string]bool{"tool1": true, "tool3": true}
	reg := New(enabled)
	reg.Register(mcp.Tool{Name: "tool1"}, nil)
	reg.Register(mcp.Tool{Name: "tool2"}, nil)
	reg.Register(mcp.Tool{Name: "tool3"}, nil)

	names := reg.EnabledToolNames()
	if len(names) != 2 {
		t.Errorf("expected 2 enabled names, got %d", len(names))
	}
	// Should be sorted
	if names[0] != "tool1" {
		t.Errorf("expected first to be 'tool1', got '%s'", names[0])
	}
	if names[1] != "tool3" {
		t.Errorf("expected second to be 'tool3', got '%s'", names[1])
	}
}

func TestAllToolNames(t *testing.T) {
	reg := New(nil)
	reg.Register(mcp.Tool{Name: "zebra"}, nil)
	reg.Register(mcp.Tool{Name: "alpha"}, nil)

	names := reg.AllToolNames()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
	// Should be sorted
	if names[0] != "alpha" {
		t.Errorf("expected first to be 'alpha', got '%s'", names[0])
	}
	if names[1] != "zebra" {
		t.Errorf("expected second to be 'zebra', got '%s'", names[1])
	}
}
