package config

import (
	"fmt"
	"os"
	"path/filepath"

	"cgl/game/ai"

	"gopkg.in/yaml.v2"
)

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".chatgamelab")
	configFile := filepath.Join(configDir, "config.yaml")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create config file if it doesn't exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := createInitialConfig(configFile); err != nil {
			return "", fmt.Errorf("failed to create initial config: %w", err)
		}
	}

	return configFile, nil
}

// createInitialConfig creates a nicely formatted initial config file
func createInitialConfig(path string) error {
	// Get available platforms from AI module
	platformInfos := ai.GetAiPlatformInfos()

	// Generate YAML content
	content := "# ChatGameLab Configuration\n"
	content += "# Replace 'your-api-key-here' with your actual API key\n\n"
	content += "platforms:\n"

	for _, platform := range platformInfos {
		content += fmt.Sprintf("  %s:\n", platform.ID)
		content += "    apikey: your-api-key-here\n"
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// GetApiKey retrieves the API key for a given platform from the config file.
// If apiKeyFlag is provided (non-empty), it returns that directly.
// Otherwise, it reads from the config file at ~/.chatgamelab/config.yaml.
// Returns an error if no valid API key is found.
func GetApiKey(platform string, apiKeyFlag string) (string, error) {
	// If API key provided via flag, use it directly
	if apiKeyFlag != "" {
		return apiKeyFlag, nil
	}

	// Read from config file
	configPath, err := GetConfigPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		Platforms map[string]struct {
			APIKey string `yaml:"apikey"`
		} `yaml:"platforms"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse config file: %w", err)
	}

	platformConfig, exists := config.Platforms[platform]
	if !exists {
		return "", fmt.Errorf("platform %s not found in config file %s", platform, configPath)
	}

	apiKey := platformConfig.APIKey
	if apiKey == "" || apiKey == "your-api-key-here" {
		return "", fmt.Errorf("no API key configured for platform %s. Use --api-key or edit %s", platform, configPath)
	}

	return apiKey, nil
}
