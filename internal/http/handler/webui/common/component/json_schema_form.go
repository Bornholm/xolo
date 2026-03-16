package component

import "encoding/json"

func getTitle(schema map[string]any, fallback string) string {
	if t, ok := schema["title"].(string); ok && t != "" {
		return t
	}
	return fallback
}

// isArrayOfObjects returns true if schema describes an array of objects.
func isArrayOfObjects(schema map[string]any) bool {
	if t, _ := schema["type"].(string); t != "array" {
		return false
	}
	items, _ := schema["items"].(map[string]any)
	if items == nil {
		return false
	}
	itemType, _ := items["type"].(string)
	return itemType == "object"
}

// marshalJSON serialises v to a JSON string (empty string on error).
func marshalJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
