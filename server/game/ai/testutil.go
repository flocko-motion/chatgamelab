package ai

import (
	"os"
	"path/filepath"
	"strings"
)

var apiKeyPathOpenai = filepath.Join(os.Getenv("HOME"), ".ai", "openai", "api-keys", "current")
var apiKeyPathMistral = filepath.Join(os.Getenv("HOME"), ".ai", "mistral", "api-keys", "current")

func GetApiKeyOpenAI() string {
	apiKey, err := os.ReadFile(apiKeyPathOpenai)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(apiKey))
}

func GetApiKeyMistral() string {
	apiKey, err := os.ReadFile(apiKeyPathMistral)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(apiKey))
}
