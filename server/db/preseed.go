// package: db / database access and repository layer
// type:    data
// job:     seed well-known dev users, institutions, and workshops for local development.
// limits:  does not define SQL queries (-> db/sqlc) or run in production.
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
	preseedDevUsers(ctx)

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

// devUserSlot is one well-known dev account: a fixed ID with the name and role
// it is meant to carry.
type devUserSlot struct {
	id            uuid.UUID
	name          string
	role          obj.Role
	institutionID *uuid.UUID
}

// devUserSlots lists the dev accounts preseed maintains (2 per role + 1 participant).
func devUserSlots() []devUserSlot {
	return []devUserSlot{
		{DevAdmin1UserID, "admin-1", obj.RoleAdmin, nil},
		{DevAdmin2UserID, "admin-2", obj.RoleAdmin, nil},
		{DevHead1UserID, "head-1", obj.RoleHead, &DevInstitutionID},
		{DevHead2UserID, "head-2", obj.RoleHead, &DevInstitutionID},
		{DevStaff1UserID, "staff-1", obj.RoleStaff, &DevInstitutionID},
		{DevStaff2UserID, "staff-2", obj.RoleStaff, &DevInstitutionID},
		{DevIndividual1UserID, "individual-1", obj.RoleIndividual, nil},
		{DevIndividual2UserID, "individual-2", obj.RoleIndividual, nil},
		{DevParticipantUserID, "participant", obj.RoleParticipant, &DevInstitutionID},
	}
}

// preseedDevUsers reconciles the well-known dev accounts with devUserSlots.
//
// Misnamed accounts are parked under a temporary name and email before the
// intended ones are applied: the slots have been renumbered before, leaving a
// name wanted by one slot held by the account of another, and name and email
// are both unique columns.
func preseedDevUsers(ctx context.Context) {
	slots := devUserSlots()
	exists := make([]bool, len(slots))
	misnamed := make([]bool, len(slots))

	for i, slot := range slots {
		user, err := GetUserByIDRaw(ctx, slot.id)
		if err != nil {
			continue
		}
		exists[i] = true
		misnamed[i] = user.Name != slot.name
		if !misnamed[i] {
			continue
		}
		log.Debug("parking dev user name", "user_id", slot.id, "from", user.Name, "to", slot.name)
		parked := "preseed-parked-" + slot.id.String()
		if err := renameDevUser(ctx, slot.id, parked); err != nil {
			log.Warn("failed to park dev user name", "user_id", slot.id, "from", user.Name, "error", err)
			misnamed[i] = false
		}
	}

	for i, slot := range slots {
		switch {
		case misnamed[i]:
			log.Debug("renaming dev user", "user_id", slot.id, "name", slot.name)
			if err := renameDevUser(ctx, slot.id, slot.name); err != nil {
				log.Warn("failed to rename dev user", "user_id", slot.id, "name", slot.name, "error", err)
				continue
			}
		case !exists[i]:
			log.Debug("creating dev user", "user_id", slot.id, "name", slot.name, "role", slot.role)
			email := devUserEmail(slot.name)
			if _, err := CreateUserWithID(ctx, slot.id, slot.name, &email, ""); err != nil {
				log.Warn("failed to create dev user", "name", slot.name, "error", err)
				continue
			}
		}
		setDevUserRole(ctx, slot)
	}
}

// renameDevUser sets a dev user's name and its matching @dev.local email.
func renameDevUser(ctx context.Context, userID uuid.UUID, name string) error {
	return queries().UpdateUser(ctx, db.UpdateUserParams{
		ID:    userID,
		Name:  name,
		Email: sql.NullString{String: devUserEmail(name), Valid: true},
	})
}

func devUserEmail(name string) string {
	return name + "@dev.local"
}

// setDevUserRole leaves the dev user holding exactly the slot's role. The roles
// are dropped and recreated rather than compared, because GetUserByID surfaces
// only one of several role rows and a stale second role stays invisible to a
// comparison. Nothing references user_role.id, so deleting is safe.
func setDevUserRole(ctx context.Context, slot devUserSlot) {
	if slot.role == "" {
		return
	}
	if err := queries().DeleteUserRoles(ctx, slot.id); err != nil {
		log.Warn("failed to clear dev user roles", "name", slot.name, "error", err)
		return
	}

	log.Debug("assigning role to dev user", "name", slot.name, "role", slot.role)
	var workshopID uuid.NullUUID
	if slot.role == obj.RoleParticipant {
		workshopID = uuid.NullUUID{UUID: DevWorkshopID, Valid: true}
	}
	arg := db.CreateUserRoleParams{
		UserID:        slot.id,
		Role:          sql.NullString{String: string(slot.role), Valid: true},
		InstitutionID: uuid.NullUUID{UUID: uuidPtrToUUID(slot.institutionID), Valid: slot.institutionID != nil},
		WorkshopID:    workshopID,
	}
	if _, err := queries().CreateUserRole(ctx, arg); err != nil {
		log.Warn("failed to assign role to dev user", "name", slot.name, "role", slot.role, "error", err)
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
