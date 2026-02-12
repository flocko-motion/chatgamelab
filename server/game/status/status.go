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
