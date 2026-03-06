// Package otlp provides shared types for OTLP log and span queries.
package otlp

// AttributeFilter represents a filter condition for OTLP queries.
type AttributeFilter struct {
	Key      string                `json:"key"`
	Operator string                `json:"operator"`
	Value    *AttributeFilterValue `json:"value,omitempty"`
}

// AttributeFilterValue represents the value in a filter condition.
type AttributeFilterValue struct {
	StringValue *string `json:"stringValue,omitempty"`
	IntValue    *string `json:"intValue,omitempty"`
	BoolValue   *bool   `json:"boolValue,omitempty"`
}

// TimeRange represents a time range for queries.
type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Pagination represents pagination settings.
type Pagination struct {
	Limit int `json:"limit,omitempty"`
}
