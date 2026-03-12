// Package formatter provides markdown table formatting for MCP tool responses.
package formatter

import (
	"fmt"
	"strings"
)

// Table renders a markdown table from headers and rows with an optional summary and footer.
func Table(title, summary string, headers []string, rows [][]string, footer string) string {
	var b strings.Builder

	if title != "" {
		b.WriteString("## ")
		b.WriteString(title)
		b.WriteString("\n\n")
	}

	if summary != "" {
		b.WriteString(summary)
		b.WriteString("\n\n")
	}

	if len(headers) == 0 {
		return b.String()
	}

	// Header row
	b.WriteString("| ")
	b.WriteString(strings.Join(headers, " | "))
	b.WriteString(" |\n")

	// Separator row
	b.WriteString("|")
	for range headers {
		b.WriteString("---|")
	}
	b.WriteString("\n")

	// Data rows
	for _, row := range rows {
		b.WriteString("| ")
		// Pad row to match header count
		padded := make([]string, len(headers))
		for i := range padded {
			if i < len(row) {
				padded[i] = escapePipe(row[i])
			}
		}
		b.WriteString(strings.Join(padded, " | "))
		b.WriteString(" |\n")
	}

	if footer != "" {
		b.WriteString("\n")
		b.WriteString(footer)
		b.WriteString("\n")
	}

	return b.String()
}

// Truncate shortens a string to maxLen, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// FormatDuration formats milliseconds into a human-readable string.
func FormatDuration(ms float64) string {
	if ms < 1 {
		return fmt.Sprintf("%.0fµs", ms*1000)
	}
	if ms < 1000 {
		return fmt.Sprintf("%.1fms", ms)
	}
	return fmt.Sprintf("%.2fs", ms/1000)
}

// SpanKindName converts an OTLP span kind number to a human-readable name.
func SpanKindName(kind int) string {
	switch kind {
	case 1:
		return "INTERNAL"
	case 2:
		return "SERVER"
	case 3:
		return "CLIENT"
	case 4:
		return "PRODUCER"
	case 5:
		return "CONSUMER"
	default:
		return "UNSPECIFIED"
	}
}

// StatusName converts an OTLP status code to a human-readable name.
func StatusName(code int) string {
	switch code {
	case 0:
		return "UNSET"
	case 1:
		return "OK"
	case 2:
		return "ERROR"
	default:
		return fmt.Sprintf("%d", code)
	}
}

// FormatListResponse formats a generic API list response as a markdown table.
// It attempts to extract items from common response shapes (array or object with items).
func FormatListResponse(resourceType string, data interface{}) string {
	items := extractItems(data)
	if items == nil {
		// Can't parse, return empty
		return fmt.Sprintf("## %s\n\nNo items found.\n", resourceType)
	}

	if len(items) == 0 {
		return fmt.Sprintf("## %s\n\nNo items found.\n", resourceType)
	}

	// Extract common fields from CRD-style items
	headers := []string{"#", "Name", "Kind", "Origin"}
	var rows [][]string

	for i, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name := extractNestedString(m, "metadata", "name")
		kind := stringVal(m["kind"])
		origin := stringVal(m["origin"])

		if name == "" {
			name = stringVal(m["name"])
		}
		if origin == "" {
			origin = extractNestedString(m, "metadata", "origin")
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			name,
			kind,
			Truncate(origin, 40),
		})
	}

	summary := fmt.Sprintf("**Found %d %s**", len(rows), strings.ToLower(resourceType))
	return Table(resourceType, summary, headers, rows, "")
}

// extractItems tries to get a slice of items from various response shapes.
func extractItems(data interface{}) []interface{} {
	if data == nil {
		return nil
	}

	// Direct array
	if arr, ok := data.([]interface{}); ok {
		return arr
	}

	// Object with known list fields
	if m, ok := data.(map[string]interface{}); ok {
		for _, key := range []string{"items", "data", "results", "rules"} {
			if arr, ok := m[key].([]interface{}); ok {
				return arr
			}
		}
	}

	return nil
}

func extractNestedString(m map[string]interface{}, keys ...string) string {
	current := m
	for i, key := range keys {
		if i == len(keys)-1 {
			return stringVal(current[key])
		}
		next, ok := current[key].(map[string]interface{})
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}

func stringVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func escapePipe(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
