package functional

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// ComputeHash computes a short SHA256 hex hash from the concatenation of the given strings.
func ComputeHash(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// InjectJsonField injects a top-level field into a JSON string and returns the updated JSON.
func InjectJsonField(jsonStr, key, value string) (string, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}
	obj[key] = value
	out, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(out) + "\n", nil
}

// ReadJsonField reads a top-level string field from a JSON string.
// Returns empty string and no error if the field doesn't exist.
func ReadJsonField(jsonStr, key string) (string, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}
	val, ok := obj[key]
	if !ok {
		return "", nil
	}
	str, ok := val.(string)
	if !ok {
		return "", nil
	}
	return str, nil
}

// IsSameJsonStructure compares two JSON strings and returns an error if they have different structures.
// It checks that both JSONs have the exact same fields (keys) at all nesting levels.
// Values are ignored - only the structure (field names and types) is compared.
// Returns nil if structures match, otherwise returns an error describing the difference.
func IsSameJsonStructure(json1, json2 string) error {
	var obj1, obj2 interface{}

	if err := json.Unmarshal([]byte(json1), &obj1); err != nil {
		return fmt.Errorf("failed to parse first JSON: %w", err)
	}

	if err := json.Unmarshal([]byte(json2), &obj2); err != nil {
		return fmt.Errorf("failed to parse second JSON: %w", err)
	}

	return compareStructure(obj1, obj2, "")
}

// compareStructure recursively compares the structure of two JSON objects
func compareStructure(obj1, obj2 interface{}, path string) error {
	type1 := reflect.TypeOf(obj1)
	type2 := reflect.TypeOf(obj2)

	// Check if types match
	if type1 != type2 {
		return fmt.Errorf("type mismatch at %s: %v vs %v", pathOrRoot(path), type1, type2)
	}

	switch v1 := obj1.(type) {
	case map[string]interface{}:
		v2 := obj2.(map[string]interface{})

		// Get sorted keys from both maps
		keys1 := getSortedKeys(v1)
		keys2 := getSortedKeys(v2)

		// Check if key sets match
		if !equalStringSlices(keys1, keys2) {
			missing1 := difference(keys2, keys1)
			missing2 := difference(keys1, keys2)

			if len(missing1) > 0 {
				return fmt.Errorf("missing fields in first JSON at %s: %v", pathOrRoot(path), missing1)
			}
			if len(missing2) > 0 {
				return fmt.Errorf("missing fields in second JSON at %s: %v", pathOrRoot(path), missing2)
			}
		}

		// Recursively compare each field
		for _, key := range keys1 {
			newPath := key
			if path != "" {
				newPath = path + "." + key
			}
			if err := compareStructure(v1[key], v2[key], newPath); err != nil {
				return err
			}
		}

	case []interface{}:
		v2 := obj2.([]interface{})

		// For arrays, compare structure of first element if both are non-empty
		if len(v1) > 0 && len(v2) > 0 {
			newPath := path + "[0]"
			if err := compareStructure(v1[0], v2[0], newPath); err != nil {
				return err
			}
		} else if len(v1) != len(v2) {
			// If one is empty and the other isn't, we can't verify structure
			// But we allow this case as it's still structurally compatible
		}

		// For primitive types (string, number, bool, null), structure is just the type
		// which we already checked above
	}

	return nil
}

// pathOrRoot returns the path or "root" if path is empty
func pathOrRoot(path string) string {
	if path == "" {
		return "root"
	}
	return path
}

// getSortedKeys returns sorted keys from a map
func getSortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// equalStringSlices checks if two string slices are equal
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// difference returns elements in a that are not in b
func difference(a, b []string) []string {
	mb := make(map[string]bool, len(b))
	for _, x := range b {
		mb[x] = true
	}
	var diff []string
	for _, x := range a {
		if !mb[x] {
			diff = append(diff, x)
		}
	}
	return diff
}

// SyncJsonStructures compares two JSON strings and adds missing keys to each
// using a placeholder value. It returns the patched JSON strings and a list of
// fields that were added to each. If both are already in sync, the added lists
// will be empty.
func SyncJsonStructures(json1, json2 string, placeholder string) (patched1, patched2 string, added1, added2 []string, err error) {
	var obj1, obj2 interface{}

	if err := json.Unmarshal([]byte(json1), &obj1); err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to parse first JSON: %w", err)
	}

	if err := json.Unmarshal([]byte(json2), &obj2); err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to parse second JSON: %w", err)
	}

	syncStructure(obj1, obj2, "", placeholder, &added1, &added2)

	out1, err := json.MarshalIndent(obj1, "", "  ")
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to marshal first JSON: %w", err)
	}

	out2, err := json.MarshalIndent(obj2, "", "  ")
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to marshal second JSON: %w", err)
	}

	return string(out1) + "\n", string(out2) + "\n", added1, added2, nil
}

// syncStructure recursively syncs two JSON objects by adding missing keys from each other.
// Missing leaf values get the placeholder string; missing sub-trees are deep-copied with
// all leaf values replaced by the placeholder.
func syncStructure(obj1, obj2 interface{}, path, placeholder string, added1, added2 *[]string) {
	m1, ok1 := obj1.(map[string]interface{})
	m2, ok2 := obj2.(map[string]interface{})
	if !ok1 || !ok2 {
		return
	}

	// Keys in m2 missing from m1
	for _, key := range getSortedKeys(m2) {
		newPath := key
		if path != "" {
			newPath = path + "." + key
		}
		if _, exists := m1[key]; !exists {
			m1[key] = deepCopyWithPlaceholder(m2[key], placeholder)
			*added1 = append(*added1, newPath)
		}
	}

	// Keys in m1 missing from m2
	for _, key := range getSortedKeys(m1) {
		newPath := key
		if path != "" {
			newPath = path + "." + key
		}
		if _, exists := m2[key]; !exists {
			m2[key] = deepCopyWithPlaceholder(m1[key], placeholder)
			*added2 = append(*added2, newPath)
		}
	}

	// Recurse into shared keys
	for _, key := range getSortedKeys(m1) {
		if _, exists := m2[key]; exists {
			newPath := key
			if path != "" {
				newPath = path + "." + key
			}
			syncStructure(m1[key], m2[key], newPath, placeholder, added1, added2)
		}
	}
}

// FindPlaceholders scans a JSON string for leaf values that match the placeholder
// and returns a list of dotted field paths where the placeholder was found.
func FindPlaceholders(jsonStr string, placeholder string) ([]string, error) {
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	var results []string
	findPlaceholders(obj, "", placeholder, &results)
	return results, nil
}

func findPlaceholders(v interface{}, path, placeholder string, results *[]string) {
	switch val := v.(type) {
	case map[string]interface{}:
		for _, key := range getSortedKeys(val) {
			newPath := key
			if path != "" {
				newPath = path + "." + key
			}
			findPlaceholders(val[key], newPath, placeholder, results)
		}
	case []interface{}:
		for i, child := range val {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			findPlaceholders(child, newPath, placeholder, results)
		}
	case string:
		if val == placeholder {
			*results = append(*results, path)
		}
	}
}

// deepCopyWithPlaceholder creates a deep copy of a JSON value, replacing all
// leaf string values with the placeholder.
func deepCopyWithPlaceholder(v interface{}, placeholder string) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, child := range val {
			result[k] = deepCopyWithPlaceholder(child, placeholder)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, child := range val {
			result[i] = deepCopyWithPlaceholder(child, placeholder)
		}
		return result
	default:
		return placeholder
	}
}
