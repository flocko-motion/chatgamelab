// Package devconfig reads the keys a developer keeps for running the engine
// by hand, as the standalone launcher and the probe do.
//
// The engine never reads a key from disk: it receives a resolved SessionSpec
// and nothing else. Resolution happens in a launcher, the standalone
// counterpart of the cascade the platform runs.
package devconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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

// Path is where the config file lives.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".chatgamelab", "config.yaml"), nil
}

// APIKey returns the configured key for a platform, or "" when there is no
// usable one. A missing file is not an error at this level; the caller
// decides whether that is fatal.
func APIKey(platform string) (string, error) {
	path, err := Path()
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
