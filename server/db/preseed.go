package db

import (
	"context"
	"database/sql"
	"time"

	db "cgl/db/sqlc"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// Well-known UUIDs for dev users (one per role)
var (
	DevAdminUserID       = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	DevHeadUserID        = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	DevStaffUserID       = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	DevParticipantUserID = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	DevGuestUserID       = uuid.MustParse("00000000-0000-0000-0000-000000000005")
	DevInstitutionID     = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	DevWorkshopID        = uuid.MustParse("00000000-0000-0000-0000-000000000011")
)

// DevUserID is the well-known UUID for the dev user (legacy, use DevAdminUserID)
var DevUserID = DevAdminUserID

// Preseed ensures required seed data exists in the database
func Preseed(ctx context.Context) {
	log.Debug("running database preseed")

	// Create dev institution first (needed for role assignments)
	preseedDevInstitution(ctx)

	// Create dev workshop (needed for participant role)
	preseedDevWorkshop(ctx)

	// Create dev users for each role
	preseedDevUser(ctx, DevAdminUserID, "admin", obj.RoleAdmin, nil)
	preseedDevUser(ctx, DevHeadUserID, "head", obj.RoleHead, &DevInstitutionID)
	preseedDevUser(ctx, DevStaffUserID, "staff", obj.RoleStaff, &DevInstitutionID)
	preseedDevUser(ctx, DevParticipantUserID, "participant", obj.RoleParticipant, &DevInstitutionID)
	preseedDevUser(ctx, DevGuestUserID, "guest", "", nil) // No role = guest

	// Create mock API key for admin user
	preseedDevApiKey(ctx, DevAdminUserID)

	// Create a dummy game for the admin user
	preseedDevGame(ctx, DevAdminUserID)

	log.Debug("database preseed completed")
}

// preseedDevInstitution creates the dev institution if it doesn't exist
func preseedDevInstitution(ctx context.Context) {
	_, err := queries().GetInstitutionByID(ctx, DevInstitutionID)
	if err != nil {
		log.Debug("creating dev institution", "id", DevInstitutionID)
		now := time.Now()
		arg := db.CreateInstitutionParams{
			ID:         DevInstitutionID,
			CreatedBy:  uuid.NullUUID{},
			CreatedAt:  now,
			ModifiedBy: uuid.NullUUID{},
			ModifiedAt: now,
			Name:       "Dev Organization",
		}
		if _, err := queries().CreateInstitution(ctx, arg); err != nil {
			log.Warn("failed to create dev institution", "error", err)
		}
	}
}

// preseedDevWorkshop creates the dev workshop if it doesn't exist
func preseedDevWorkshop(ctx context.Context) {
	_, err := queries().GetWorkshopByID(ctx, DevWorkshopID)
	if err != nil {
		log.Debug("creating dev workshop", "id", DevWorkshopID)
		now := time.Now()
		arg := db.CreateWorkshopParams{
			ID:            DevWorkshopID,
			CreatedBy:     uuid.NullUUID{},
			CreatedAt:     now,
			ModifiedBy:    uuid.NullUUID{},
			ModifiedAt:    now,
			Name:          "Dev Workshop",
			InstitutionID: DevInstitutionID,
			Active:        true,
			Public:        false,
		}
		if _, err := queries().CreateWorkshop(ctx, arg); err != nil {
			log.Warn("failed to create dev workshop", "error", err)
		}
	}
}

// preseedDevUser creates a dev user with the given role if they don't exist
func preseedDevUser(ctx context.Context, userID uuid.UUID, name string, role obj.Role, institutionID *uuid.UUID) {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		log.Debug("creating dev user", "user_id", userID, "name", name, "role", role)
		email := name + "@dev.local"
		user, err = CreateUserWithID(ctx, userID, name, &email, "")
		if err != nil {
			log.Warn("failed to create dev user", "name", name, "error", err)
			return
		}
	}

	// Assign role if specified and user doesn't have one
	if role != "" && user.Role == nil {
		log.Debug("assigning role to dev user", "name", name, "role", role)
		var workshopID uuid.NullUUID
		if role == obj.RoleParticipant {
			workshopID = uuid.NullUUID{UUID: DevWorkshopID, Valid: true}
		}
		arg := db.CreateUserRoleParams{
			UserID:        userID,
			Role:          sql.NullString{String: string(role), Valid: true},
			InstitutionID: uuid.NullUUID{UUID: uuidPtrToUUID(institutionID), Valid: institutionID != nil},
			WorkshopID:    workshopID,
		}
		if _, err := queries().CreateUserRole(ctx, arg); err != nil {
			log.Warn("failed to assign role to dev user", "name", name, "role", role, "error", err)
		}
	}
}

// preseedDevApiKey creates a mock API key for the given user if they don't have one
func preseedDevApiKey(ctx context.Context, userID uuid.UUID) {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return
	}

	if len(user.ApiKeys) == 0 {
		log.Debug("creating mock API key for dev user", "user_id", userID)
		keyID, err := CreateApiKey(ctx, userID, "Dev Mock Key", "mock", "mock-api-key-for-testing")
		if err != nil {
			log.Warn("failed to create mock API key", "error", err)
			return
		}

		// Set it as the default
		shares, err := GetApiKeySharesByUser(ctx, userID)
		if err != nil {
			log.Warn("failed to get shares", "error", err)
			return
		}
		for _, share := range shares {
			if share.ApiKeyID == *keyID {
				if err := SetUserDefaultApiKeyShare(ctx, userID, &share.ID); err != nil {
					log.Warn("failed to set default API key", "error", err)
				}
				break
			}
		}
	}
}

// preseedDevGame creates a dummy game for the given user if they don't have one
func preseedDevGame(ctx context.Context, userID uuid.UUID) {
	games, err := GetGames(ctx, &userID, nil)
	if err != nil {
		log.Warn("failed to get games", "error", err)
		return
	}
	if len(games) == 0 {
		log.Debug("creating dummy game for dev user", "user_id", userID)
		game := &obj.Game{
			Name:                   "Dev Test Game",
			Description:            "A simple test game for development",
			Public:                 false,
			SystemMessageScenario:  `An example game for testing purposes. Full of stereotypical characters and situations. Perfect for demonstrating basic gameplay mechanics.`,
			SystemMessageGameStart: "Welcome to the tavern! What would you like to do? I heard there's a dragon nearby...",
			ImageStyle:             "fantasy pixel art, 16-bit style",
			StatusFields:           `[{"name": "Health", "value": "100"}, {"name": "Gold", "value": "5"}, {"name": "XP", "value": "0"}, {"name": "Level", "value": "1"}]`,
		}
		if err := CreateGame(ctx, userID, game); err != nil {
			log.Warn("failed to create dummy game", "error", err)
		}
	}
}
