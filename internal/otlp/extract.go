package otlp

// ExtractServiceName gets service.name from resource attributes in an OTLP resource map.
func ExtractServiceName(resourceMap map[string]interface{}) string {
	resource, ok := resourceMap["resource"].(map[string]interface{})
	if !ok {
		return ""
	}

	attrs, ok := resource["attributes"].([]interface{})
	if !ok {
		return ""
	}

	for _, attr := range attrs {
		attrMap, ok := attr.(map[string]interface{})
		if !ok {
			continue
		}
		if key, ok := attrMap["key"].(string); ok && key == "service.name" {
			if value, ok := attrMap["value"].(map[string]interface{}); ok {
				if strVal, ok := value["stringValue"].(string); ok {
					return strVal
				}
			}
		}
	}

	return ""
}
