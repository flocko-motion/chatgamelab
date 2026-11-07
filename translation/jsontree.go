package main

// The locale files are trees of objects with text at the leaves. A leaf is
// named by its dotted path, such as "auth.ageGroup.label".

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

func parse(s string) (any, error) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return v, nil
}

func mustIndent(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b) + "\n"
}

func indentJSON(s string) (string, error) {
	v, err := parse(s)
	if err != nil {
		return "", err
	}
	return mustIndent(v), nil
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// leaves maps every leaf's path to its value as text.
func leaves(s string) (map[string]string, error) {
	v, err := parse(s)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	var walk func(v any, path string)
	walk = func(v any, path string) {
		switch val := v.(type) {
		case map[string]any:
			for _, k := range sortedKeys(val) {
				walk(val[k], join(path, k))
			}
		case []any:
			for i, child := range val {
				walk(child, fmt.Sprintf("%s[%d]", path, i))
			}
		case string:
			out[path] = val
		default:
			out[path] = fmt.Sprintf("%v", val)
		}
	}
	walk(v, "")
	return out, nil
}

// fieldHashes hashes each field's text across all sources, so a field's hash
// changes when its text changes in any of them.
func fieldHashes(sources ...string) (map[string]string, error) {
	all := make([]map[string]string, len(sources))
	paths := map[string]bool{}
	for i, s := range sources {
		l, err := leaves(s)
		if err != nil {
			return nil, err
		}
		all[i] = l
		for p := range l {
			paths[p] = true
		}
	}
	out := map[string]string{}
	for p := range paths {
		h := sha256.New()
		for _, l := range all {
			if v, ok := l[p]; ok {
				h.Write([]byte(v))
			}
		}
		out[p] = hex.EncodeToString(h.Sum(nil))[:16]
	}
	return out, nil
}

// extractPaths keeps only the leaves whose paths are in keep.
func extractPaths(s string, keep map[string]bool) (string, error) {
	v, err := parse(s)
	if err != nil {
		return "", err
	}
	var walk func(v any, path string) any
	walk = func(v any, path string) any {
		switch val := v.(type) {
		case map[string]any:
			out := map[string]any{}
			for k, child := range val {
				if c := walk(child, join(path, k)); c != nil {
					out[k] = c
				}
			}
			if len(out) == 0 {
				return nil
			}
			return out
		case []any:
			var out []any
			for i, child := range val {
				if c := walk(child, fmt.Sprintf("%s[%d]", path, i)); c != nil {
					out = append(out, c)
				}
			}
			if len(out) == 0 {
				return nil
			}
			return out
		default:
			if keep[path] {
				return val
			}
			return nil
		}
	}
	out := walk(v, "")
	if out == nil {
		out = map[string]any{}
	}
	return mustIndent(out), nil
}

// mergeJSON writes the leaves of src over dst.
func mergeJSON(dst, src string) (string, error) {
	d, err := parse(dst)
	if err != nil {
		return "", err
	}
	s, err := parse(src)
	if err != nil {
		return "", err
	}
	var merge func(d, s any)
	merge = func(d, s any) {
		dm, ok1 := d.(map[string]any)
		sm, ok2 := s.(map[string]any)
		if !ok1 || !ok2 {
			return
		}
		for k, sv := range sm {
			if dv, ok := dm[k].(map[string]any); ok {
				if _, ok := sv.(map[string]any); ok {
					merge(dv, sv)
					continue
				}
			}
			dm[k] = sv
		}
	}
	merge(d, s)
	return mustIndent(d), nil
}

// syncStructures adds to each tree the fields only the other has, set to the
// placeholder, and names the fields it added to each.
func syncStructures(a, b, placeholder string) (patchedA, patchedB string, addedA, addedB []string, err error) {
	va, err := parse(a)
	if err != nil {
		return "", "", nil, nil, err
	}
	vb, err := parse(b)
	if err != nil {
		return "", "", nil, nil, err
	}
	var fill func(v any) any
	fill = func(v any) any {
		switch val := v.(type) {
		case map[string]any:
			out := map[string]any{}
			for k, c := range val {
				out[k] = fill(c)
			}
			return out
		case []any:
			out := make([]any, len(val))
			for i, c := range val {
				out[i] = fill(c)
			}
			return out
		default:
			return placeholder
		}
	}
	var sync func(a, b any, path string)
	sync = func(a, b any, path string) {
		ma, ok1 := a.(map[string]any)
		mb, ok2 := b.(map[string]any)
		if !ok1 || !ok2 {
			return
		}
		for _, k := range sortedKeys(mb) {
			if _, ok := ma[k]; !ok {
				ma[k] = fill(mb[k])
				addedA = append(addedA, join(path, k))
			}
		}
		for _, k := range sortedKeys(ma) {
			if _, ok := mb[k]; !ok {
				mb[k] = fill(ma[k])
				addedB = append(addedB, join(path, k))
			} else {
				sync(ma[k], mb[k], join(path, k))
			}
		}
	}
	sync(va, vb, "")
	return mustIndent(va), mustIndent(vb), addedA, addedB, nil
}

// findPlaceholders names the leaves whose text is the placeholder.
func findPlaceholders(s, placeholder string) []string {
	l, err := leaves(s)
	if err != nil {
		return nil
	}
	var out []string
	for p, v := range l {
		if v == placeholder {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
