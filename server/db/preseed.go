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

// Well-known UUIDs for dev users (two per role + one participant)
var (
	DevAdmin1UserID      = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	DevAdmin2UserID      = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	DevHead1UserID       = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	DevHead2UserID       = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	DevStaff1UserID      = uuid.MustParse("00000000-0000-0000-0000-000000000005")
	DevStaff2UserID      = uuid.MustParse("00000000-0000-0000-0000-000000000006")
	DevIndividual1UserID = uuid.MustParse("00000000-0000-0000-0000-000000000007")
	DevIndividual2UserID = uuid.MustParse("00000000-0000-0000-0000-000000000008")
	DevParticipantUserID = uuid.MustParse("00000000-0000-0000-0000-000000000009")
	DevInstitutionID     = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	DevWorkshopID        = uuid.MustParse("00000000-0000-0000-0000-000000000011")
)

// DevUserID is the well-known UUID for the dev user (legacy, use DevAdmin1UserID)
var DevUserID = DevAdmin1UserID

// Legacy aliases for backward compatibility
var (
	DevAdminUserID = DevAdmin1UserID
	DevHeadUserID  = DevHead1UserID
	DevStaffUserID = DevStaff1UserID
	DevGuestUserID = DevIndividual1UserID
)

// Preseed ensures required seed data exists in the database
func Preseed(ctx context.Context) {
	log.Debug("running database preseed")

	// Create dev institution first (needed for role assignments)
	preseedDevInstitution(ctx)

	// Create dev workshop (needed for participant role)
	preseedDevWorkshop(ctx)

	// Create dev users (2 per role + 1 participant)
	preseedDevUser(ctx, DevAdmin1UserID, "admin-1", obj.RoleAdmin, nil)
	preseedDevUser(ctx, DevAdmin2UserID, "admin-2", obj.RoleAdmin, nil)
	preseedDevUser(ctx, DevHead1UserID, "head-1", obj.RoleHead, &DevInstitutionID)
	preseedDevUser(ctx, DevHead2UserID, "head-2", obj.RoleHead, &DevInstitutionID)
	preseedDevUser(ctx, DevStaff1UserID, "staff-1", obj.RoleStaff, &DevInstitutionID)
	preseedDevUser(ctx, DevStaff2UserID, "staff-2", obj.RoleStaff, &DevInstitutionID)
	preseedDevUser(ctx, DevIndividual1UserID, "individual-1", obj.RoleIndividual, nil)
	preseedDevUser(ctx, DevIndividual2UserID, "individual-2", obj.RoleIndividual, nil)
	preseedDevUser(ctx, DevParticipantUserID, "participant", obj.RoleParticipant, &DevInstitutionID)

	// Create mock API key for first admin user
	preseedDevApiKey(ctx, DevAdmin1UserID)

	// Create a dummy game for the first admin user
	preseedDevGame(ctx, DevAdmin1UserID)

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

	// Assign role if specified
	if role != "" {
		// If user has an "individual" role (auto-assigned), delete it first
		if user.Role != nil && user.Role.Role == obj.RoleIndividual {
			log.Debug("removing auto-assigned individual role for dev user", "name", name)
			if err := queries().DeleteUserRoles(ctx, userID); err != nil {
				log.Warn("failed to delete individual role", "name", name, "error", err)
				return
			}
		}

		// Only assign if user doesn't have the correct role already
		if user.Role == nil || user.Role.Role != role {
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
