// package: db / database access and repository layer
// type:    data
// job:     create, read, update, delete users; participant tokens, default shares, and active workshop.
// limits:  does not manage roles/admin promotion or user listings/stats (-> user_roles.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// CreateUser creates a new user in the database
func CreateUser(ctx context.Context, name string, email *string, auth0ID string, ageGroup *string) (*obj.User, error) {
	emailStr := ""
	if email != nil {
		emailStr = *email
	}
	arg := db.CreateUserParams{
		Name:     name,
		Email:    sql.NullString{String: emailStr, Valid: email != nil},
		Auth0ID:  sql.NullString{String: auth0ID, Valid: auth0ID != ""},
		AgeGroup: stringPtrToNullString(ageGroup),
	}

	id, err := queries().CreateUser(ctx, arg)
	if err != nil {
		return nil, err
	}

	// Auto-upgrade to admin if email is in ADMIN_EMAILS list
	promoted := false
	if email != nil && isAdminEmail(*email) {
		err := autoUpgradeUserToAdmin(ctx, id)
		promoted = err == nil
		switch {
		case errors.Is(err, errAdminBootstrapClosed):
			log.Info("admin email gets the default role: bootstrap closed", "user_id", id)
		case err != nil:
			// Log error but don't fail user creation
			log.Warn("failed to auto-upgrade user to admin", "user_id", id, "error", err)
		}
	}
	if !promoted {
		// Assign default "individual" role to new users
		log.Debug("assigning default individual role to new user", "user_id", id)
		if err := assignDefaultIndividualRole(ctx, id); err != nil {
			// Log error but don't fail user creation
			log.Warn("failed to assign default individual role", "user_id", id, "error", err)
		}
	}

	finalUser, err := GetUserByID(ctx, id)
	if err != nil {
		log.Error("failed to get user after creation", "user_id", id, "error", err)
		return nil, err
	}
	if finalUser.Role == nil {
		log.Warn("user has no role after creation", "user_id", id)
	} else {
		log.Debug("user created successfully", "user_id", id, "role", finalUser.Role.Role)
	}
	return finalUser, nil
}

// CreateUserWithID creates a new user with a specific UUID
func CreateUserWithID(ctx context.Context, id uuid.UUID, name string, email *string, auth0ID string) (*obj.User, error) {
	emailStr := ""
	if email != nil {
		emailStr = *email
	}
	arg := db.CreateUserWithIDParams{
		ID:      id,
		Name:    name,
		Email:   sql.NullString{String: emailStr, Valid: email != nil},
		Auth0ID: sql.NullString{String: auth0ID, Valid: auth0ID != ""},
	}

	_, err := queries().CreateUserWithID(ctx, arg)
	// ON CONFLICT (id) DO NOTHING returns no row: the user is already there and
	// keeps the role it has.
	if errors.Is(err, sql.ErrNoRows) {
		log.Debug("user with this id already exists", "user_id", id)
		return GetUserByID(ctx, id)
	}
	if err != nil {
		return nil, err
	}

	// Auto-upgrade to admin if email is in ADMIN_EMAILS list
	promoted := false
	if email != nil && isAdminEmail(*email) {
		err := autoUpgradeUserToAdmin(ctx, id)
		promoted = err == nil
		switch {
		case errors.Is(err, errAdminBootstrapClosed):
			log.Info("admin email gets the default role: bootstrap closed", "user_id", id)
		case err != nil:
			// Log error but don't fail user creation
			log.Warn("failed to auto-upgrade user to admin", "user_id", id, "error", err)
		}
	}
	if !promoted {
		// Assign default "individual" role to new users
		log.Debug("assigning default individual role to new user", "user_id", id)
		if err := assignDefaultIndividualRole(ctx, id); err != nil {
			// Log error but don't fail user creation
			log.Warn("failed to assign default individual role", "user_id", id, "error", err)
		}
	}

	finalUser, err := GetUserByID(ctx, id)
	if err != nil {
		log.Error("failed to get user after creation", "user_id", id, "error", err)
		return nil, err
	}
	if finalUser.Role == nil {
		log.Warn("user has no role after creation", "user_id", id)
	} else {
		log.Debug("user created successfully", "user_id", id, "role", finalUser.Role.Role)
	}
	return finalUser, nil
}

// UpdateUserDetails updates a user's name and email.
func UpdateUserDetails(ctx context.Context, id uuid.UUID, name string, email *string) error {
	emailStr := ""
	if email != nil {
		emailStr = *email
	}
	arg := db.UpdateUserParams{
		ID:    id,
		Name:  name,
		Email: sql.NullString{String: emailStr, Valid: email != nil},
	}
	return queries().UpdateUser(ctx, arg)
}

// UpdateUserAgeGroup updates a user's age group.
func UpdateUserAgeGroup(ctx context.Context, id uuid.UUID, ageGroup *string) error {
	arg := db.UpdateUserAgeGroupParams{
		ID:       id,
		AgeGroup: stringPtrToNullString(ageGroup),
	}
	return queries().UpdateUserAgeGroup(ctx, arg)
}

// UpdateUserAiQualityTier updates a user's AI quality tier.
func UpdateUserAiQualityTier(ctx context.Context, id uuid.UUID, tier *string) error {
	arg := db.UpdateUserAiQualityTierParams{
		ID:            id,
		AiQualityTier: stringPtrToNullString(tier),
	}
	return queries().UpdateUserAiQualityTier(ctx, arg)
}

// UpdateUserLanguage updates a user's preferred language (users may only update their own).
func UpdateUserLanguage(ctx context.Context, currentUserID uuid.UUID, targetUserID uuid.UUID, language string) error {
	// Check permissions - users can only update their own language
	if err := canAccessUser(ctx, currentUserID, OpUpdate, targetUserID); err != nil {
		return err
	}

	arg := db.UpdateUserLanguageParams{
		ID:       targetUserID,
		Language: language,
	}
	return queries().UpdateUserLanguage(ctx, arg)
}

// GetUserByIDRaw gets the raw user record by ID (includes participant_token field)
func GetUserByIDRaw(ctx context.Context, id uuid.UUID) (db.AppUser, error) {
	return queries().GetUserByID(ctx, id)
}

// RemoveUser deletes a user (checks permissions internally)
func RemoveUser(ctx context.Context, currentUserID uuid.UUID, targetUserID uuid.UUID) error {
	if err := CanDeleteUser(ctx, currentUserID, targetUserID); err != nil {
		return err
	}

	// Last-head protection: if the target is a head, ensure the institution has another head
	targetUser, err := GetUserByID(ctx, targetUserID)
	if err != nil {
		return obj.ErrNotFound("target user not found")
	}
	if targetUser.Role != nil && targetUser.Role.Role == obj.RoleHead && targetUser.Role.Institution != nil {
		instID := uuid.NullUUID{UUID: targetUser.Role.Institution.ID, Valid: true}
		headCount, err := queries().CountHeadsByInstitution(ctx, instID)
		if err != nil {
			return obj.ErrServerError("failed to check institution heads")
		}
		if headCount <= 1 {
			return obj.NewAppError(obj.ErrCodeLastHead, "cannot delete the last head of an institution")
		}
	}

	return DeleteUser(ctx, targetUserID)
}

// ParticipantAuthError represents a specific error during participant authentication
type ParticipantAuthError struct {
	Code    string // "invalid_token", "workshop_inactive"
	Message string
}

// Error returns the participant authentication error message.
func (e *ParticipantAuthError) Error() string {
	return e.Message
}

// GetUserByParticipantToken gets a user by their participant token
// Returns specific error codes for different failure scenarios
func GetUserByParticipantToken(ctx context.Context, token string) (*obj.User, error) {
	res, err := queries().GetUserByParticipantToken(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		// Check if the token exists but workshop is inactive
		status, statusErr := queries().CheckParticipantTokenStatus(ctx, sql.NullString{String: token, Valid: true})
		if statusErr == nil && status.TokenExists {
			// Token exists but query failed - likely inactive workshop
			workshopActive, ok := status.WorkshopActive.(bool)
			if ok && !workshopActive {
				return nil, &ParticipantAuthError{
					Code:    "workshop_inactive",
					Message: "Workshop is inactive",
				}
			}
		}
		// Token doesn't exist or other error
		return nil, &ParticipantAuthError{
			Code:    "invalid_token",
			Message: "Invalid participant token",
		}
	}
	// Get full user details with role
	return GetUserByID(ctx, res.ID)
}

// GetUserByID gets a user by ID
func GetUserByID(ctx context.Context, id uuid.UUID) (*obj.User, error) {
	log.Debug("getting user details by id", "user_id", id)
	res, err := queries().GetUserDetailsByID(ctx, id)
	if err != nil {
		log.Error("failed to get user details by id", "user_id", id, "error", err)
		return nil, err
	}
	log.Debug("got user details", "user_id", id, "name", res.Name, "email", res.Email.String)
	user := obj.User{
		ID: res.ID,
		Meta: obj.Meta{
			CreatedBy:  res.CreatedBy,
			CreatedAt:  &res.CreatedAt,
			ModifiedBy: res.ModifiedBy,
			ModifiedAt: &res.CreatedAt,
		},
		Name:          res.Name,
		Email:         sqlNullStringToMaybeString(res.Email),
		DeletedAt:     &res.DeletedAt.Time,
		Auth0Id:       sqlNullStringToMaybeString(res.Auth0ID),
		AiQualityTier: sqlNullStringToMaybeString(res.AiQualityTier),
		Language:      res.Language,
		AgeGroup:      sqlNullStringToMaybeString(res.AgeGroup),
	}
	if res.RoleID.Valid {
		role, err := stringToRole(res.Role.String)
		if err != nil {
			return nil, err
		}
		user.Role = &obj.UserRole{
			ID:   res.RoleID.UUID,
			Role: role,
		}
		if res.InstitutionID.Valid {
			inst := &obj.Institution{
				ID:   res.InstitutionID.UUID,
				Name: res.InstitutionName.String,
			}
			if res.InstitutionFreeUseApiKeyShareID.Valid {
				inst.FreeUseApiKeyShareID = &res.InstitutionFreeUseApiKeyShareID.UUID
			}
			if res.InstitutionPromptConstraints.Valid {
				inst.PromptConstraints = &res.InstitutionPromptConstraints.String
			}
			user.Role.Institution = inst
		}
		// For participants: use workshop_id (their assigned workshop)
		// For head/staff/individual: use active_workshop_id (workshop mode) or workshop_id as fallback
		if res.WorkshopID.Valid {
			var aiQualityTier *string
			if res.WorkshopAiQualityTier.Valid {
				aiQualityTier = &res.WorkshopAiQualityTier.String
			}
			var promptConstraints *string
			if res.WorkshopPromptConstraints.Valid {
				promptConstraints = &res.WorkshopPromptConstraints.String
			}
			user.Role.Workshop = &obj.Workshop{
				ID:                         res.WorkshopID.UUID,
				Name:                       res.WorkshopName.String,
				ShowPublicGames:            res.WorkshopShowPublicGames.Bool,
				ShowOtherParticipantsGames: res.WorkshopShowOtherParticipantsGames.Bool,
				DesignEditingEnabled:       res.WorkshopDesignEditingEnabled.Bool,
				IsPaused:                   res.WorkshopIsPaused.Bool,
				AllowGameSharing:           res.WorkshopAllowGameSharing.Bool,
				AiQualityTier:              aiQualityTier,
				PromptConstraints:          promptConstraints,
			}
		} else if res.ActiveWorkshopID.Valid {
			// Head/staff/individual in workshop mode - use active workshop
			var aiQualityTier *string
			if res.ActiveWorkshopAiQualityTier.Valid {
				aiQualityTier = &res.ActiveWorkshopAiQualityTier.String
			}
			var promptConstraints *string
			if res.ActiveWorkshopPromptConstraints.Valid {
				promptConstraints = &res.ActiveWorkshopPromptConstraints.String
			}
			user.Role.Workshop = &obj.Workshop{
				ID:                         res.ActiveWorkshopID.UUID,
				Name:                       res.ActiveWorkshopName.String,
				ShowPublicGames:            res.ActiveWorkshopShowPublicGames.Bool,
				ShowOtherParticipantsGames: res.ActiveWorkshopShowOtherParticipantsGames.Bool,
				DesignEditingEnabled:       res.ActiveWorkshopDesignEditingEnabled.Bool,
				IsPaused:                   res.ActiveWorkshopIsPaused.Bool,
				AllowGameSharing:           res.ActiveWorkshopAllowGameSharing.Bool,
				AiQualityTier:              aiQualityTier,
				PromptConstraints:          promptConstraints,
			}
		}
	}
	log.Debug("fetching api key shares for user", "user_id", id)
	user.ApiKeys, err = GetApiKeySharesByUser(ctx, id)
	if err != nil {
		log.Error("failed to get api key shares for user", "user_id", id, "error", err)
		return nil, err
	}
	log.Debug("successfully loaded user", "user_id", id, "name", user.Name, "api_keys_count", len(user.ApiKeys))

	return &user, nil
}

// GetUserByAuth0ID gets a user by Auth0 ID
func GetUserByAuth0ID(ctx context.Context, auth0ID string) (*obj.User, error) {
	log.Debug("searching for user by auth0_id", "auth0_id", auth0ID, "auth0_id_length", len(auth0ID))
	id, err := queries().GetUserIDByAuth0ID(ctx, sql.NullString{String: auth0ID, Valid: true})
	if err != nil {
		log.Debug("user not found by auth0_id", "auth0_id", auth0ID, "error", err)
		return nil, err
	}
	log.Debug("found user by auth0_id", "auth0_id", auth0ID, "user_id", id)
	return GetUserByID(ctx, id)
}

// GetUserByEmail gets a user by email address
func GetUserByEmail(ctx context.Context, email string) (*obj.User, error) {
	raw, err := queries().GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
	if err != nil {
		return nil, err
	}
	return GetUserByID(ctx, raw.ID)
}

// IsNameTaken checks if a username is already taken
func IsNameTaken(ctx context.Context, name string) (bool, error) {
	return queries().IsNameTaken(ctx, name)
}

// IsNameTakenByOther checks if a username is taken by another user (for updates)
func IsNameTakenByOther(ctx context.Context, name string, excludeUserID uuid.UUID) (bool, error) {
	return queries().IsNameTakenByOther(ctx, db.IsNameTakenByOtherParams{
		Name: name,
		ID:   excludeUserID,
	})
}

// IsEmailTakenByOther checks if an email is taken by another user (for updates)
func IsEmailTakenByOther(ctx context.Context, email string, excludeUserID uuid.UUID) (bool, error) {
	return queries().IsEmailTakenByOther(ctx, db.IsEmailTakenByOtherParams{
		Email: sql.NullString{String: email, Valid: true},
		ID:    excludeUserID,
	})
}

// DeleteUser hard-deletes a user and cleans up all their data:
// sessions, API keys (with cascade), shares, roles, favourites, workshop participant records,
// invites, and originally_created_by references.
func DeleteUser(ctx context.Context, id uuid.UUID) error {
	// 1. Delete session messages and sessions
	if err := queries().DeleteGameSessionMessagesByUserID(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user session messages: %w", err)
	}
	if err := queries().DeleteAllUserSessions(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	// 2. Clean up games created by this user
	gameIDs, _ := queries().GetGameIDsByCreator(ctx, uuid.NullUUID{UUID: id, Valid: true})
	for _, gameID := range gameIDs {
		_ = queries().DeleteGameTagsByGameID(ctx, gameID)
		_ = queries().DeleteGameSessionMessagesByGameID(ctx, gameID)
		_ = queries().DeleteGameSessionsByGameID(ctx, gameID)
		_ = queries().DeleteFavouritesByGameID(ctx, gameID)
		_ = queries().DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: gameID, Valid: true})
		_ = queries().DeleteGameSharesByGameID(ctx, gameID)           // clean up game shares
		_ = queries().ClearGameSponsoredApiKeyByApiKeyID(ctx, gameID) // clear sponsored refs
		_ = queries().HardDeleteGame(ctx, gameID)
	}

	// 3. Clean up API keys owned by this user (cascade: clears shares, workshop refs, game sponsors, etc.)
	keyIDs, _ := queries().GetApiKeyIDsByUser(ctx, id)
	for _, keyID := range keyIDs {
		// Clear session api_key_id references
		_ = queries().ClearSessionApiKeyID(ctx, uuid.NullUUID{UUID: keyID, Valid: true})
		// Clear user default_api_key_share_id references
		_ = queries().ClearUserDefaultApiKeyShareByApiKeyID(ctx, keyID)
		// Clear workshop default_api_key_share_id references
		_ = queries().ClearWorkshopDefaultApiKeyShareByApiKeyID(ctx, keyID)
		// Clear game sponsored API key references
		_ = queries().ClearGameSponsoredApiKeyByApiKeyID(ctx, keyID)
		// Clear system free-use key reference
		_ = queries().ClearSystemSettingsFreeUseApiKey(ctx, uuid.NullUUID{UUID: keyID, Valid: true})
		// Delete all shares for this key
		_ = queries().DeleteApiKeySharesByApiKeyID(ctx, keyID)
	}
	// Delete all API keys owned by this user
	_ = queries().DeleteAllApiKeysByUser(ctx, id)

	// 3. Clear user's default_api_key_share_id (in case it references someone else's share)
	_ = queries().SetUserDefaultApiKeyShare(ctx, db.SetUserDefaultApiKeyShareParams{
		ID:                   id,
		DefaultApiKeyShareID: uuid.NullUUID{},
	})

	// 4. Delete workshop participant records for this user
	_ = queries().DeleteWorkshopParticipantsByUserID(ctx, id)

	// 5. Delete favourites
	_ = queries().DeleteUserFavourites(ctx, id)

	// 6. Delete user roles
	_ = queries().DeleteUserRoles(ctx, id)

	// 7. Clear system free-use API key if it references this user's keys
	_ = ClearSystemSettingsFreeUseApiKeyByOwner(ctx, id)

	// 8. Clean up invite references (FK constraints block hard delete)
	nullID := uuid.NullUUID{UUID: id, Valid: true}
	_ = queries().DeleteInvitesForUser(ctx, nullID)
	_ = queries().ClearInviteAcceptedByUser(ctx, nullID)

	// 9. Clear originally_created_by on cloned games owned by others
	_ = queries().ClearGameOriginalCreator(ctx, nullID)

	// 10. Hard-delete the user
	return queries().DeleteUser(ctx, id)
}

// SetUserDefaultApiKeyShare sets the default API key share for a user.
// Pass nil to clear the default.
func SetUserDefaultApiKeyShare(ctx context.Context, userID uuid.UUID, shareID *uuid.UUID) error {
	arg := db.SetUserDefaultApiKeyShareParams{
		ID:                   userID,
		DefaultApiKeyShareID: uuid.NullUUID{UUID: uuidPtrToUUID(shareID), Valid: shareID != nil},
	}
	return queries().SetUserDefaultApiKeyShare(ctx, arg)
}

// GetUserDefaultApiKeyShare returns the default API key share ID for a user, or nil if not set.
func GetUserDefaultApiKeyShare(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	result, err := queries().GetUserDefaultApiKeyShare(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !result.Valid {
		return nil, nil
	}
	return &result.UUID, nil
}

func uuidPtrToUUID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.UUID{}
	}
	return *id
}

// SetActiveWorkshop sets the active workshop for a head/staff/individual user (workshop mode)
// Validates that the user has the right role and the workshop belongs to their institution
func SetActiveWorkshop(ctx context.Context, userID uuid.UUID, workshopID uuid.UUID) error {
	// Get user to verify role and institution
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	if user.Role == nil {
		return obj.ErrForbidden("user has no role")
	}

	// Only head, staff, and individual can set active workshop
	if user.Role.Role != obj.RoleHead && user.Role.Role != obj.RoleStaff && user.Role.Role != obj.RoleIndividual {
		return obj.ErrForbidden("only head, staff, and individual users can enter workshop mode")
	}

	// Get workshop to validate it exists and check institution
	workshop, err := queries().GetWorkshopByID(ctx, workshopID)
	if err != nil {
		return obj.ErrNotFound("workshop not found")
	}

	// For head/staff: workshop must belong to their institution
	if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
		if user.Role.Institution == nil || user.Role.Institution.ID != workshop.InstitutionID {
			return obj.ErrForbidden("workshop does not belong to your institution")
		}
	}

	// For individual: they can enter any active workshop (they don't have an institution)
	// But the workshop must be active
	if !workshop.Active {
		return obj.ErrForbidden("workshop is not active")
	}

	// Set the active workshop
	err = queries().SetUserActiveWorkshop(ctx, db.SetUserActiveWorkshopParams{
		UserID:           userID,
		ActiveWorkshopID: uuid.NullUUID{UUID: workshopID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to set active workshop: %w", err)
	}

	return nil
}

// ClearActiveWorkshop clears the active workshop for a user (leave workshop mode).
// Also cleans up any orphaned participant role for the given workshop.
func ClearActiveWorkshop(ctx context.Context, userID uuid.UUID, workshopID ...uuid.UUID) error {
	err := queries().ClearUserActiveWorkshop(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to clear active workshop: %w", err)
	}
	// Clean up orphaned participant role if workshop ID is provided
	if len(workshopID) > 0 {
		_ = queries().DeleteUserParticipantRole(ctx, db.DeleteUserParticipantRoleParams{
			UserID:     userID,
			WorkshopID: uuid.NullUUID{UUID: workshopID[0], Valid: true},
		})
	}
	return nil
}
