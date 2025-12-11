package functional

import (
	"fmt"
	"os"
	"time"
)

func Shorten(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-2] + ".."
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
