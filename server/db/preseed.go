package db

import (
	"cgl/obj"
	"context"
	"log"

	"github.com/google/uuid"
)

// DevUserID is the well-known UUID for the dev user
var DevUserID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// Preseed ensures required seed data exists in the database
func Preseed(ctx context.Context) {
	// Ensure dev user exists
	user, err := GetUserByID(ctx, DevUserID)
	if err != nil {
		log.Printf("Creating dev user with ID %s", DevUserID)
		user, err = CreateUserWithID(ctx, DevUserID, "dev", nil, "")
		if err != nil {
			log.Printf("Warning: failed to create dev user: %v", err)
			return
		}
	}

	// Ensure dev user has a mock API key
	if len(user.ApiKeys) == 0 {
		log.Printf("Creating mock API key for dev user")
		keyID, err := CreateApiKey(ctx, DevUserID, "Dev Mock Key", "mock", "mock-api-key-for-testing")
		if err != nil {
			log.Printf("Warning: failed to create mock API key: %v", err)
			return
		}

		// Set it as the default
		shares, err := GetApiKeySharesByUser(ctx, DevUserID)
		if err != nil {
			log.Printf("Warning: failed to get shares: %v", err)
			return
		}
		for _, share := range shares {
			if share.ApiKeyID == *keyID {
				if err := SetUserDefaultApiKeyShare(ctx, DevUserID, &share.ID); err != nil {
					log.Printf("Warning: failed to set default API key: %v", err)
				}
				break
			}
		}
	}

	// Ensure dev user has a dummy game
	games, err := GetGames(ctx, &DevUserID, nil)
	if err != nil {
		log.Printf("Warning: failed to get games: %v", err)
		return
	}
	if len(games) == 0 {
		log.Printf("Creating dummy game for dev user")
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
			log.Printf("Warning: failed to create dummy game: %v", err)
		}
	}
}
