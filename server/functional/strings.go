package functional

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"
)

func Shorten(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-2] + ".."
}

// ShortenLeft returns a shortened version of a string for display (showing right part)
func ShortenLeft(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return ".." + s[len(s)-max+2:]
}

func MaybeFieldToString(m map[string]any, key, ifNotExisting, ifNil string) string {
	if val, exists := m[key]; exists {
		return MaybeToString(val, ifNil)
	}
	return ifNotExisting
}

func MaybeToString(v any, ifNil string) string {
	if v == nil {
		return ifNil
	}

	// Handle pointers by dereferencing them
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ifNil
		}
		return fmt.Sprintf("%v", val.Elem().Interface())
	}

	return fmt.Sprintf("%v", v)
}

func BoolToString(b bool, ifTrue, ifFalse string) string {
	if b {
		return ifTrue
	}
	return ifFalse
}

func RequireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		fmt.Printf("missing env %s - did you source the .env file?\n", name)
		os.Exit(1)
	}
	return v
}

func EnvOrDefault(name, defaultValue string) string {
	v := os.Getenv(name)
	if v == "" {
		return defaultValue
	}
	return v
}

func HumanizeDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

// NormalizeJson unmarshals JSON into the given struct type and re-marshals it to normalize the format.
// If o is not a pointer, a pointer to a new instance of its type is created automatically.
func NormalizeJson(in string, o any) string {
	target := EnsurePointer(o)
	_ = json.Unmarshal([]byte(in), target)
	normalized, _ := json.Marshal(target)
	return string(normalized)
}

func MustAnyToJson(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// EnsurePointer returns a pointer to o if o is not already a pointer.
func EnsurePointer(o any) any {
	v := reflect.ValueOf(o)
	if v.Kind() == reflect.Ptr {
		return o
	}
	// Create a new pointer to the same type and return it
	ptr := reflect.New(v.Type())
	return ptr.Interface()
}

// EndsWithDigits checks if a dash-separated string ends with a segment of exactly n digits.
// Useful for filtering dated/versioned model IDs like "gpt-4-0613" or "mistral-small-2402".
func EndsWithDigits(s string, n int) bool {
	parts := SplitDash(s)
	if len(parts) < 2 {
		return false
	}
	last := parts[len(parts)-1]
	if len(last) != n {
		return false
	}
	for _, ch := range last {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// EndsWithDatePattern checks if a dash-separated string ends with a YYYY-MM-DD pattern.
// Example: "gpt-4-2024-01-25" returns true.
func EndsWithDatePattern(s string) bool {
	parts := SplitDash(s)
	if len(parts) < 4 {
		return false
	}
	lastThree := parts[len(parts)-3:]
	if len(lastThree[0]) != 4 || len(lastThree[1]) != 2 || len(lastThree[2]) != 2 {
		return false
	}
	for _, part := range lastThree {
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

// SplitDash splits a string by dashes. Convenience wrapper for model ID parsing.
func SplitDash(s string) []string {
	return splitBy(s, '-')
}

func splitBy(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// NormalizeYaml unmarshals YAML into the given struct type and re-marshals it to normalize the format.
// If o is not a pointer, a pointer to a new instance of its type is created automatically.
func NormalizeYaml(in string, o any) string {
	target := EnsurePointer(o)
	_ = yaml.Unmarshal([]byte(in), target)
	normalized, _ := yaml.Marshal(target)
	return string(normalized)
}
