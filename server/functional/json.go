package functional

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

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
