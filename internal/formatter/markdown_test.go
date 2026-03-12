package formatter

import (
	"strings"
	"testing"
)

func TestTable_Basic(t *testing.T) {
	result := Table("Test Title", "**Found 3 items**",
		[]string{"#", "Name", "Value"},
		[][]string{
			{"1", "foo", "bar"},
			{"2", "baz", "qux"},
		},
		"_Footer text_",
	)

	if !strings.Contains(result, "## Test Title") {
		t.Error("missing title")
	}
	if !strings.Contains(result, "**Found 3 items**") {
		t.Error("missing summary")
	}
	if !strings.Contains(result, "| # | Name | Value |") {
		t.Error("missing header row")
	}
	if !strings.Contains(result, "|---|---|---|") {
		t.Error("missing separator row")
	}
	if !strings.Contains(result, "| 1 | foo | bar |") {
		t.Error("missing first data row")
	}
	if !strings.Contains(result, "| 2 | baz | qux |") {
		t.Error("missing second data row")
	}
	if !strings.Contains(result, "_Footer text_") {
		t.Error("missing footer")
	}
}

func TestTable_EmptyHeaders(t *testing.T) {
	result := Table("Title", "Summary", nil, nil, "")
	if !strings.Contains(result, "## Title") {
		t.Error("missing title")
	}
	// Should not contain table markup
	if strings.Contains(result, "|---|") {
		t.Error("should not have separator for empty headers")
	}
}

func TestTable_NoTitleOrSummary(t *testing.T) {
	result := Table("", "", []string{"A", "B"}, [][]string{{"1", "2"}}, "")
	if strings.Contains(result, "##") {
		t.Error("should not have title")
	}
	if !strings.Contains(result, "| A | B |") {
		t.Error("missing header")
	}
}

func TestTable_PipeEscaping(t *testing.T) {
	result := Table("", "", []string{"Col"},
		[][]string{{"value|with|pipes"}}, "")
	if strings.Contains(result, "value|with") {
		t.Error("pipes in values should be escaped")
	}
	if !strings.Contains(result, `value\|with\|pipes`) {
		t.Error("pipes should be escaped with backslash")
	}
}

func TestTable_ShortRowPadding(t *testing.T) {
	result := Table("", "", []string{"A", "B", "C"},
		[][]string{{"1"}}, "")
	// Row should be padded to match header count
	if !strings.Contains(result, "| 1 |  |  |") {
		t.Errorf("short row should be padded, got: %s", result)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := Truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms       float64
		expected string
	}{
		{0.5, "500µs"},
		{0.001, "1µs"},
		{1.0, "1.0ms"},
		{123.456, "123.5ms"},
		{999.9, "999.9ms"},
		{1000, "1.00s"},
		{1500, "1.50s"},
		{60000, "60.00s"},
	}

	for _, tt := range tests {
		result := FormatDuration(tt.ms)
		if result != tt.expected {
			t.Errorf("FormatDuration(%f) = %q, want %q", tt.ms, result, tt.expected)
		}
	}
}

func TestSpanKindName(t *testing.T) {
	tests := []struct {
		kind     int
		expected string
	}{
		{0, "UNSPECIFIED"},
		{1, "INTERNAL"},
		{2, "SERVER"},
		{3, "CLIENT"},
		{4, "PRODUCER"},
		{5, "CONSUMER"},
		{99, "UNSPECIFIED"},
	}

	for _, tt := range tests {
		result := SpanKindName(tt.kind)
		if result != tt.expected {
			t.Errorf("SpanKindName(%d) = %q, want %q", tt.kind, result, tt.expected)
		}
	}
}

func TestStatusName(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{0, "UNSET"},
		{1, "OK"},
		{2, "ERROR"},
		{99, "99"},
	}

	for _, tt := range tests {
		result := StatusName(tt.code)
		if result != tt.expected {
			t.Errorf("StatusName(%d) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}

func TestFormatListResponse_Array(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{
			"kind": "PersesDashboard",
			"metadata": map[string]interface{}{
				"name":   "my-dashboard",
				"origin": "org123/dash456",
			},
		},
		map[string]interface{}{
			"kind": "PersesDashboard",
			"metadata": map[string]interface{}{
				"name": "another-dashboard",
			},
		},
	}

	result := FormatListResponse("Dashboards", data)

	if !strings.Contains(result, "## Dashboards") {
		t.Error("missing title")
	}
	if !strings.Contains(result, "**Found 2 dashboards**") {
		t.Error("missing summary")
	}
	if !strings.Contains(result, "my-dashboard") {
		t.Error("missing first dashboard name")
	}
	if !strings.Contains(result, "another-dashboard") {
		t.Error("missing second dashboard name")
	}
	if !strings.Contains(result, "PersesDashboard") {
		t.Error("missing kind")
	}
}

func TestFormatListResponse_NestedItems(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"kind": "Dash0View",
				"metadata": map[string]interface{}{
					"name": "test-view",
				},
			},
		},
	}

	result := FormatListResponse("Views", data)
	if !strings.Contains(result, "test-view") {
		t.Error("should extract items from nested 'items' key")
	}
}

func TestFormatListResponse_Empty(t *testing.T) {
	result := FormatListResponse("Things", []interface{}{})
	if !strings.Contains(result, "No items found") {
		t.Error("should show empty message")
	}
}

func TestFormatListResponse_Nil(t *testing.T) {
	result := FormatListResponse("Things", nil)
	if !strings.Contains(result, "No items found") {
		t.Error("should show empty message for nil")
	}
}

func TestFormatListResponse_NonArray(t *testing.T) {
	result := FormatListResponse("Things", "not an array")
	if !strings.Contains(result, "No items found") {
		t.Error("should show empty message for non-array")
	}
}

func TestEscapePipe(t *testing.T) {
	if escapePipe("no pipes") != "no pipes" {
		t.Error("should not modify strings without pipes")
	}
	if escapePipe("a|b|c") != `a\|b\|c` {
		t.Error("should escape pipes")
	}
}

func TestExtractNestedString(t *testing.T) {
	m := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "test",
		},
		"top": "level",
	}

	if extractNestedString(m, "metadata", "name") != "test" {
		t.Error("should extract nested string")
	}
	if extractNestedString(m, "top") != "level" {
		t.Error("should extract top-level string")
	}
	if extractNestedString(m, "missing", "key") != "" {
		t.Error("should return empty for missing keys")
	}
}
