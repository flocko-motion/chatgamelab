package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// The engine never reads a key from disk: it receives a resolved SessionSpec
// and nothing else. Resolution happens here, in the launcher, which is the
// standalone counterpart of the cascade the portal runs.

// A spec's apiKey field either holds the real key, or one of these sentinels
// asking the launcher to resolve one. Anything else — including empty — is
// taken literally, so a session can never run on a key nobody chose.
var resolveSentinels = map[string]bool{"auto": true, "dev": true}

// The placeholder the repo's CLI writes into a fresh config file. It is not a
// key, and must not be mistaken for one.
const placeholderKey = "your-api-key-here"

// Format matches the file the repo's CLI already writes, at
// ~/.chatgamelab/config.yaml.
type cliConfig struct {
	Platforms map[string]struct {
		APIKey string `yaml:"apikey"`
	} `yaml:"platforms"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".chatgamelab", "config.yaml"), nil
}

// apiKeyFromConfig returns the configured key for a platform, or "" when there
// is no usable one. A missing file is not an error at this level; the caller
// decides whether that is fatal.
func apiKeyFromConfig(platform string) (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	var cfg cliConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	if key := cfg.Platforms[platform].APIKey; key != placeholderKey {
		return key, nil
	}
	return "", nil
}
