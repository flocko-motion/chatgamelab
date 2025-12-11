package functional

import "fmt"

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
	return fmt.Sprintf("%v", v)
}
