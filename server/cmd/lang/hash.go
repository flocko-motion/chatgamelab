package lang

import (
	"cgl/functional"
	"encoding/json"
	"fmt"
)

// ComputeFieldHashes computes a hash for each leaf field in the JSON structures.
// It takes multiple JSON strings (e.g., en.json and de.json) and computes a hash
// for each field path based on the concatenated values from all sources.
// Returns a map from field path to hash.
func ComputeFieldHashes(jsonStrs ...string) (map[string]string, error) {
	if len(jsonStrs) == 0 {
		return nil, fmt.Errorf("no JSON strings provided")
	}

	// Collect field values from all sources
	allFieldValues := make([]map[string]string, len(jsonStrs))
	for i, jsonStr := range jsonStrs {
		fieldValues, err := functional.CollectFieldValues(jsonStr)
		if err != nil {
			return nil, fmt.Errorf("failed to collect field values from JSON %d: %w", i, err)
		}
		allFieldValues[i] = fieldValues
	}

	// Compute hash for each field path by concatenating values from all sources
	result := make(map[string]string)
	
	// Get all unique paths
	allPaths := make(map[string]bool)
	for _, fieldValues := range allFieldValues {
		for path := range fieldValues {
			allPaths[path] = true
		}
	}

	// For each path, collect values from all sources and compute hash
	for path := range allPaths {
		var values []string
		for _, fieldValues := range allFieldValues {
			if val, exists := fieldValues[path]; exists {
				values = append(values, val)
			}
		}
		result[path] = functional.ComputeHash(values...)
	}

	return result, nil
}

// ExtractChangedFields compares old and new hash maps and returns a JSON string
// containing only the fields that have changed (different hash) or are new.
// The structure is preserved, but only changed leaf values are included.
func ExtractChangedFields(fullJSON string, oldHashes, newHashes map[string]string) (string, error) {
	// Build a set of changed paths
	changedPaths := make(map[string]bool)
	for path, newHash := range newHashes {
		oldHash, exists := oldHashes[path]
		if !exists || oldHash != newHash {
			changedPaths[path] = true
		}
	}

	if len(changedPaths) == 0 {
		return "{}", nil
	}

	// Extract only changed fields using the generic functional utility
	return functional.ExtractFieldsByPaths(fullJSON, changedPaths)
}

// MergeTranslations merges a partial translation (containing only changed fields)
// into a full existing translation. Fields in the partial translation overwrite
// corresponding fields in the full translation.
func MergeTranslations(fullJSON, partialJSON string) (string, error) {
	return functional.MergeJSON(fullJSON, partialJSON)
}

// SaveHashFile saves the hash map to a JSON file
func SaveHashFile(hashes map[string]string) ([]byte, error) {
	hashJSON, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hash map: %w", err)
	}
	return append(hashJSON, '\n'), nil
}
