package status

import (
	"cgl/obj"
	"encoding/json"
)

// FieldNames extracts the ordered field names from a StatusFields JSON string.
func FieldNames(statusFieldsJSON string) []string {
	var fields []obj.StatusField
	if err := json.Unmarshal([]byte(statusFieldsJSON), &fields); err != nil {
		return nil
	}
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}

// MapToFields converts map[string]string back to []StatusField,
// preserving the order defined by fieldNames.
// If a key is missing from statusMap, the value from fallback is used
// (defensive: strict schemas prevent this, but guards against non-strict platforms).
func MapToFields(statusMap map[string]string, fieldNames []string, fallback map[string]string) []obj.StatusField {
	fields := make([]obj.StatusField, 0, len(fieldNames))
	for _, name := range fieldNames {
		value, ok := statusMap[name]
		if !ok && fallback != nil {
			value = fallback[name]
		}
		fields = append(fields, obj.StatusField{
			Name:  name,
			Value: value,
		})
	}
	return fields
}

// FieldsToMap converts []StatusField to map[string]string.
func FieldsToMap(fields []obj.StatusField) map[string]string {
	m := make(map[string]string, len(fields))
	for _, f := range fields {
		m[f.Name] = f.Value
	}
	return m
}

// BuildResponseSchema builds a game-specific JSON schema for LLM responses.
// The status object has fixed keys matching the game's status field names,
// preventing the AI from hallucinating extra fields or dropping existing ones.
func BuildResponseSchema(statusFieldsJSON string) map[string]interface{} {
	fieldNames := FieldNames(statusFieldsJSON)

	// Build status properties with exact field names as keys
	statusProperties := make(map[string]interface{}, len(fieldNames))
	for _, name := range fieldNames {
		statusProperties[name] = map[string]interface{}{"type": "string"}
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "The narrative response to the player's action",
			},
			"status": map[string]interface{}{
				"type":                 "object",
				"properties":           statusProperties,
				"required":             fieldNames,
				"additionalProperties": false,
				"description":          "Updated status fields after the action",
			},
			"imagePrompt": map[string]interface{}{
				"type":        "string",
				"description": "Description for generating an image of the scene",
			},
		},
		"required":             []string{"message", "status", "imagePrompt"},
		"additionalProperties": false,
	}
}
