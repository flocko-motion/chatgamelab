package db

import (
	"context"

	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// DevUserID is the well-known UUID for the dev user
var DevUserID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// Preseed ensures required seed data exists in the database
func Preseed(ctx context.Context) {
	log.Debug("running database preseed")

	// Ensure dev user exists
	user, err := GetUserByID(ctx, DevUserID)
	if err != nil {
		log.Debug("creating dev user", "user_id", DevUserID)
		user, err = CreateUserWithID(ctx, DevUserID, "dev", nil, "")
		if err != nil {
			log.Warn("failed to create dev user", "error", err)
			return
		}
	}

	// Ensure dev user has admin role
	if user.Role == nil {
		log.Debug("assigning admin role to dev user")
		if err := autoUpgradeUserToAdmin(ctx, DevUserID); err != nil {
			log.Warn("failed to assign admin role to dev user", "error", err)
			return
		}
	}

	// Ensure dev user has a mock API key
	if len(user.ApiKeys) == 0 {
		log.Debug("creating mock API key for dev user")
		keyID, err := CreateApiKey(ctx, DevUserID, "Dev Mock Key", "mock", "mock-api-key-for-testing")
		if err != nil {
			log.Warn("failed to create mock API key", "error", err)
			return
		}

		// Set it as the default
		shares, err := GetApiKeySharesByUser(ctx, DevUserID)
		if err != nil {
			log.Warn("failed to get shares", "error", err)
			return
		}
		for _, share := range shares {
			if share.ApiKeyID == *keyID {
				if err := SetUserDefaultApiKeyShare(ctx, DevUserID, &share.ID); err != nil {
					log.Warn("failed to set default API key", "error", err)
				}
				break
			}
		}
	}

	// Ensure dev user has a dummy game
	games, err := GetGames(ctx, &DevUserID, nil)
	if err != nil {
		log.Warn("failed to get games", "error", err)
		return
	}
	if len(games) == 0 {
		log.Debug("creating dummy game for dev user")
		game := &obj.Game{
			Name:                   "Dev Test Game",
			Description:            "A simple test game for development",
			Public:                 false,
			SystemMessageScenario:  `An example game for testing purposes. Full of stereotypical characters and situations. Perfect for demonstrating basic gameplay mechanics.`,
			SystemMessageGameStart: "Welcome to the tavern! What would you like to do? I heard there's a dragon nearby...",
			ImageStyle:             "fantasy pixel art, 16-bit style",
			StatusFields:           `[{"name": "Health", "value": "100"}, {"name": "Gold", "value": "5"}, {"name": "XP", "value": "0"}, {"name": "Level", "value": "1"}]`,
		}
		if err := CreateGame(ctx, DevUserID, game); err != nil {
			log.Warn("failed to create dummy game", "error", err)
		}
	}

	log.Debug("database preseed completed")
}
