// package: db / database access and repository layer
// type:    data
// job:     give a dev user the real AI keys the developer keeps in ~/.chatgamelab/config.yaml, and the site its free-use key.
// limits:  does not run in production; a missing file or key adds nothing.
package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cgl/log"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// The platforms a developer may keep a key for, in the order they are seeded:
// the first one seeded becomes the dev user's default key.
var devKeyPlatforms = []string{"openai", "mistral"}

// The placeholder the CLI writes into a fresh config file; it is no key.
const devKeyPlaceholder = "your-api-key-here"

// DevPlatformKey is the developer's key for a platform from
// ~/.chatgamelab/config.yaml, the file the engine's dev tools read, or ""
// when there is none.
func DevPlatformKey(platform string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".chatgamelab", "config.yaml")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var cfg struct {
		Platforms map[string]struct {
			APIKey string `yaml:"apikey"`
		} `yaml:"platforms"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if key := cfg.Platforms[platform].APIKey; key != devKeyPlaceholder {
		return key, nil
	}
	return "", nil
}

// preseedDevPlatformKeys gives the user a key for every platform the
// developer keeps one for and the user has none on yet; the first key added
// becomes their default, the mock key stays for free runs. The first of the
// user's keys in devKeyPlatforms' order becomes the site's free-use key, so
// every dev account plays and draws covers on a real platform; a free-use
// key an admin has set stays.
//
// Never under go test: the integration tests run the dev preseed too, and
// they must not spend the developer's money or find keys a test expects
// absent.
func preseedDevPlatformKeys(ctx context.Context, userID uuid.UUID) {
	if testing.Testing() {
		return
	}
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return
	}
	keys := map[string]uuid.UUID{} // the user's own key on each platform
	for _, share := range user.ApiKeys {
		if share.ApiKey != nil && share.ApiKey.UserID == userID {
			keys[share.ApiKey.Platform] = share.ApiKeyID
		}
	}
	madeDefault := false
	for _, platform := range devKeyPlatforms {
		if _, ok := keys[platform]; ok {
			continue
		}
		key, err := DevPlatformKey(platform)
		if err != nil {
			log.Warn("failed to read the dev config", "error", err)
			return
		}
		if key == "" {
			continue
		}
		log.Debug("creating dev API key from the dev config", "user_id", userID, "platform", platform)
		keyID, err := CreateApiKey(ctx, userID, "Dev "+platform+" Key", platform, key)
		if err != nil {
			log.Warn("failed to create dev API key", "platform", platform, "error", err)
			continue
		}
		keys[platform] = *keyID
		if !madeDefault {
			setDevDefaultKey(ctx, userID, *keyID)
			madeDefault = true
		}
	}
	preseedSiteFreeUseKey(ctx, keys)
}

// preseedSiteFreeUseKey makes the first of keys, in devKeyPlatforms' order,
// the site's free-use key while the site has none.
func preseedSiteFreeUseKey(ctx context.Context, keys map[string]uuid.UUID) {
	settings, err := GetSystemSettings(ctx)
	if err != nil || settings.FreeUseApiKeyID != nil {
		return
	}
	for _, platform := range devKeyPlatforms {
		if keyID, ok := keys[platform]; ok {
			log.Debug("making the dev API key the site's free-use key", "platform", platform)
			if err := queries().UpdateSystemSettingsFreeUseApiKey(ctx, uuid.NullUUID{UUID: keyID, Valid: true}); err != nil {
				log.Warn("failed to set the site's free-use key", "error", err)
			}
			return
		}
	}
}

// setDevDefaultKey makes the user's own key their default.
func setDevDefaultKey(ctx context.Context, userID, keyID uuid.UUID) {
	shares, err := GetApiKeySharesByUser(ctx, userID)
	if err != nil {
		log.Warn("failed to get shares", "error", err)
		return
	}
	for _, share := range shares {
		if share.ApiKeyID == keyID {
			if err := SetUserDefaultApiKeyShare(ctx, userID, &share.ID); err != nil {
				log.Warn("failed to set default API key", "error", err)
			}
			return
		}
	}
}
