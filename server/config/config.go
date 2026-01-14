package config

import (
	"fmt"
	"os"
	"path/filepath"

	"cgl/game/ai"

	"gopkg.in/yaml.v2"
)

// Config represents the full configuration structure
type Config struct {
	Server       ServerConfig              `yaml:"server"`
	KnownServers []KnownServer             `yaml:"known_servers,omitempty"`
	Platforms    map[string]PlatformConfig `yaml:"platforms"`
}

// ServerConfig holds server connection details
type ServerConfig struct {
	URL string `yaml:"url"`
	JWT string `yaml:"jwt"`
}

// KnownServer represents a server that has been used before
type KnownServer struct {
	Alias string `yaml:"alias"`
	URL   string `yaml:"url"`
	JWT   string `yaml:"jwt,omitempty"`
}

// PlatformConfig holds platform-specific API keys
type PlatformConfig struct {
	APIKey string `yaml:"apikey"`
}

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
	content += "# Use 'user login' command to configure server connection\n\n"
	content += "server:\n"
	content += "  url: \"\"\n"
	content += "  jwt: \"\"\n\n"
	content += "# Replace 'your-api-key-here' with your actual API key\n"
	content += "platforms:\n"

	for _, platform := range platformInfos {
		content += fmt.Sprintf("  %s:\n", platform.ID)
		content += "    apikey: your-api-key-here\n"
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// LoadConfig reads and parses the config file
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig writes the config to the config file
func SaveConfig(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetServerURL returns the configured server URL
func GetServerURL() (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}

	if config.Server.URL == "" {
		return "", fmt.Errorf("no server configured. Use 'user login' to configure")
	}

	return config.Server.URL, nil
}

// GetJWT returns the configured JWT token
func GetJWT() (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}

	return config.Server.JWT, nil
}

// SetServerConfig updates the server configuration and optionally adds to known servers
func SetServerConfig(url, jwt string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	config.Server.URL = url
	config.Server.JWT = jwt

	return SaveConfig(config)
}

// SetServerConfigWithAlias updates the server configuration and adds/updates it in known servers
func SetServerConfigWithAlias(url, jwt, alias string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	config.Server.URL = url
	config.Server.JWT = jwt

	// Add or update in known servers
	AddOrUpdateKnownServer(config, alias, url, jwt)

	return SaveConfig(config)
}

// GetKnownServerByAlias retrieves a known server by its alias
func GetKnownServerByAlias(alias string) (*KnownServer, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	for _, server := range config.KnownServers {
		if server.Alias == alias {
			return &server, nil
		}
	}

	return nil, fmt.Errorf("no server found with alias '%s'", alias)
}

// AddOrUpdateKnownServer adds or updates a server in the known servers list
func AddOrUpdateKnownServer(config *Config, alias, url, jwt string) {
	// Check if alias already exists
	for i, server := range config.KnownServers {
		if server.Alias == alias {
			// Update existing
			config.KnownServers[i].URL = url
			config.KnownServers[i].JWT = jwt
			return
		}
	}

	// Add new server
	config.KnownServers = append(config.KnownServers, KnownServer{
		Alias: alias,
		URL:   url,
		JWT:   jwt,
	})
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
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}

	platformConfig, exists := config.Platforms[platform]
	if !exists {
		configPath, _ := GetConfigPath()
		return "", fmt.Errorf("platform %s not found in config file %s", platform, configPath)
	}

	apiKey := platformConfig.APIKey
	if apiKey == "" || apiKey == "your-api-key-here" {
		configPath, _ := GetConfigPath()
		return "", fmt.Errorf("no API key configured for platform %s. Use --api-key or edit %s", platform, configPath)
	}

	return apiKey, nil
}
