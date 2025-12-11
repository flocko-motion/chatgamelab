package functional

import (
	"fmt"
	"os"
)

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
