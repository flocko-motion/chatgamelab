package db

import (
	db "cgl/db/sqlc"
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// CreateUser creates a new user in the database
func CreateUser(ctx context.Context, name string, email *string, auth0ID string) (*obj.User, error) {
	emailStr := ""
	if email != nil {
		emailStr = *email
	}
	arg := db.CreateUserParams{
		Name:    name,
		Email:   sql.NullString{String: emailStr, Valid: email != nil},
		Auth0ID: sql.NullString{String: auth0ID, Valid: auth0ID != ""},
	}

	id, err := queries().CreateUser(ctx, arg)
	if err != nil {
		return nil, err
	}

	// Auto-upgrade to admin if email is in ADMIN_EMAILS list
	if email != nil && isAdminEmail(*email) {
		if err := autoUpgradeUserToAdmin(ctx, id); err != nil {
			// Log error but don't fail user creation
			log.Warn("failed to auto-upgrade user to admin", "user_id", id, "error", err)
		}
	} else {
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
	if err != nil {
		return nil, err
	}

	// Auto-upgrade to admin if email is in ADMIN_EMAILS list
	if email != nil && isAdminEmail(*email) {
		if err := autoUpgradeUserToAdmin(ctx, id); err != nil {
			// Log error but don't fail user creation
			log.Warn("failed to auto-upgrade user to admin", "user_id", id, "error", err)
		}
	} else {
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

func UpdateUserAiQualityTier(ctx context.Context, id uuid.UUID, tier *string) error {
	arg := db.UpdateUserAiQualityTierParams{
		ID:            id,
		AiQualityTier: stringPtrToNullString(tier),
	}
	return queries().UpdateUserAiQualityTier(ctx, arg)
}

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
	res, err := queries().GetUserDetailsByID(ctx, id)
	if err != nil {
		return nil, err
	}
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
			user.Role.Institution = inst
		}
		// For participants: use workshop_id (their assigned workshop)
		// For head/staff/individual: use active_workshop_id (workshop mode) or workshop_id as fallback
		if res.WorkshopID.Valid {
			var aiQualityTier *string
			if res.WorkshopAiQualityTier.Valid {
				aiQualityTier = &res.WorkshopAiQualityTier.String
			}
			user.Role.Workshop = &obj.Workshop{
				ID:                         res.WorkshopID.UUID,
				Name:                       res.WorkshopName.String,
				ShowPublicGames:            res.WorkshopShowPublicGames.Bool,
				ShowOtherParticipantsGames: res.WorkshopShowOtherParticipantsGames.Bool,
				AiQualityTier:              aiQualityTier,
			}
		} else if res.ActiveWorkshopID.Valid {
			// Head/staff/individual in workshop mode - use active workshop
			var aiQualityTier *string
			if res.ActiveWorkshopAiQualityTier.Valid {
				aiQualityTier = &res.ActiveWorkshopAiQualityTier.String
			}
			user.Role.Workshop = &obj.Workshop{
				ID:                         res.ActiveWorkshopID.UUID,
				Name:                       res.ActiveWorkshopName.String,
				ShowPublicGames:            res.ActiveWorkshopShowPublicGames.Bool,
				ShowOtherParticipantsGames: res.ActiveWorkshopShowOtherParticipantsGames.Bool,
				AiQualityTier:              aiQualityTier,
			}
		}
	}
	user.ApiKeys, err = GetApiKeySharesByUser(ctx, id)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByAuth0ID gets a user by Auth0 ID
func GetUserByAuth0ID(ctx context.Context, auth0ID string) (*obj.User, error) {
	id, err := queries().GetUserIDByAuth0ID(ctx, sql.NullString{String: auth0ID, Valid: true})
	if err != nil {
		return nil, err
	}
	return GetUserByID(ctx, id)
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

// DeleteUser soft-deletes a user and cleans up all their data:
// sessions, API keys (with cascade), shares, roles, favourites, workshop participant records.
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
		_ = queries().ClearPrivateShareGameIDByGameID(ctx, uuid.NullUUID{UUID: gameID, Valid: true})
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
		// Clean up private share guest data
		privateGames, _ := queries().GetGamesWithPrivateShareByApiKeyID(ctx, keyID)
		for _, g := range privateGames {
			_ = DeleteGuestDataByGameID(ctx, g.ID)
			_ = queries().ClearGamePrivateShare(ctx, g.ID)
		}
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

	// 8. Soft-delete the user
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

// isAdminEmail checks if the given email is in the ADMIN_EMAILS environment variable
func isAdminEmail(email string) bool {
	adminEmails := os.Getenv("ADMIN_EMAILS")
	if adminEmails == "" {
		return false
	}

	// Split by comma and trim whitespace
	emails := strings.Split(adminEmails, ",")
	for _, adminEmail := range emails {
		if strings.TrimSpace(adminEmail) == "" {
			continue
		}
		if strings.TrimSpace(adminEmail) == strings.TrimSpace(email) {
			return true
		}
	}
	return false
}

// autoUpgradeUserToAdmin creates an admin role for the user
func autoUpgradeUserToAdmin(ctx context.Context, userID uuid.UUID) error {
	// Create admin role for the user
	arg := db.CreateUserRoleParams{
		UserID:        userID,
		Role:          sql.NullString{String: string(obj.RoleAdmin), Valid: true},
		InstitutionID: uuid.NullUUID{}, // Admin role has no institution
		WorkshopID:    uuid.NullUUID{}, // Admin role has no workshop
	}

	_, err := queries().CreateUserRole(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to create admin role: %w", err)
	}

	log.Info("auto-upgraded user to admin role", "user_id", userID)
	return nil
}

// assignDefaultIndividualRole creates an "individual" role for the user
func assignDefaultIndividualRole(ctx context.Context, userID uuid.UUID) error {
	// Create individual role for the user
	arg := db.CreateUserRoleParams{
		UserID:        userID,
		Role:          sql.NullString{String: string(obj.RoleIndividual), Valid: true},
		InstitutionID: uuid.NullUUID{}, // Individual role has no institution
		WorkshopID:    uuid.NullUUID{}, // Individual role has no workshop
	}

	_, err := queries().CreateUserRole(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to create individual role: %w", err)
	}

	log.Debug("assigned default individual role to user", "user_id", userID)
	return nil
}

// CheckAndPromoteAdmin checks if a single user should be promoted to admin based on ADMIN_EMAILS.
// Called at login time to ensure admin promotion happens immediately, not just on server restart.
// Returns the (possibly updated) user.
func CheckAndPromoteAdmin(ctx context.Context, user *obj.User) *obj.User {
	if user.Email == nil || !isAdminEmail(*user.Email) {
		return user
	}

	// Already admin?
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		log.Debug("user already has admin role", "user_id", user.ID, "email", *user.Email)
		return user
	}

	// Only individual users can be promoted
	if user.Role == nil || user.Role.Role != obj.RoleIndividual {
		log.Warn("skipping login admin promotion: user does not have individual role", "user_id", user.ID, "email", *user.Email)
		return user
	}

	log.Info("promoting user to admin at login", "user_id", user.ID, "email", *user.Email)

	if err := queries().DeleteUserRoles(ctx, user.ID); err != nil {
		log.Warn("failed to delete existing roles for login admin promotion", "user_id", user.ID, "error", err)
		return user
	}

	if err := autoUpgradeUserToAdmin(ctx, user.ID); err != nil {
		log.Warn("failed to promote user to admin at login", "user_id", user.ID, "error", err)
		return user
	}

	// Reload user to get updated role
	updated, err := GetUserByID(ctx, user.ID)
	if err != nil {
		log.Warn("failed to reload user after login admin promotion", "user_id", user.ID, "error", err)
		return user
	}
	return updated
}

// PromoteAdminEmails checks users whose email is in ADMIN_EMAILS and promotes
// them to admin role. Called on server startup.
func PromoteAdminEmails(ctx context.Context) {
	adminEmails := os.Getenv("ADMIN_EMAILS")
	if adminEmails == "" {
		log.Debug("ADMIN_EMAILS not set, skipping admin promotion check")
		return
	}

	log.Info("checking for admin email promotions", "admin_emails", adminEmails)

	for _, email := range strings.Split(adminEmails, ",") {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}

		raw, err := queries().GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
		if err != nil {
			log.Debug("admin email user not found, skipping", "email", email)
			continue
		}

		user, err := GetUserByID(ctx, raw.ID)
		if err != nil {
			log.Warn("failed to load user for admin promotion", "email", email, "error", err)
			continue
		}

		// Only individual users can be promoted to admin
		if user.Role == nil || user.Role.Role != obj.RoleIndividual {
			log.Warn("skipping admin promotion: user does not have individual role", "user_id", user.ID, "email", *user.Email, "role", user.Role)
			continue
		}

		CheckAndPromoteAdmin(ctx, user)
	}
}

// GetAllUsers returns all users with their roles (for admin/CLI use)
func GetAllUsers(ctx context.Context) ([]obj.User, error) {
	rows, err := queries().GetAllUsersWithDetails(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]obj.User, 0, len(rows))
	for _, res := range rows {
		user := obj.User{
			ID: res.ID,
			Meta: obj.Meta{
				CreatedBy:  res.CreatedBy,
				CreatedAt:  &res.CreatedAt,
				ModifiedBy: res.ModifiedBy,
				ModifiedAt: &res.ModifiedAt,
			},
			Name:      res.Name,
			Email:     sqlNullStringToMaybeString(res.Email),
			DeletedAt: &res.DeletedAt.Time,
			Auth0Id:   sqlNullStringToMaybeString(res.Auth0ID),
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
				user.Role.Institution = &obj.Institution{
					ID:   res.InstitutionID.UUID,
					Name: res.InstitutionName.String,
				}
			}
			if res.WorkshopID.Valid {
				user.Role.Workshop = &obj.Workshop{
					ID:   res.WorkshopID.UUID,
					Name: res.WorkshopName.String,
				}
			}
		}
		users = append(users, user)
	}
	return users, nil
}

func UpdateUserRole(ctx context.Context, currentUserID uuid.UUID, targetUserID uuid.UUID, role *string, institutionID *uuid.UUID, workshopID *uuid.UUID) error {
	// Check permissions
	currentUser, err := GetUserByID(ctx, currentUserID)
	if err != nil {
		return obj.ErrNotFound("current user not found")
	}

	// Admin can do anything
	isAdmin := currentUser.Role != nil && currentUser.Role.Role == obj.RoleAdmin

	// Head can promote staff to head within their own institution
	isHead := currentUser.Role != nil && currentUser.Role.Role == obj.RoleHead && currentUser.Role.Institution != nil

	if !isAdmin && !isHead {
		return obj.ErrForbidden("only admins or heads can manage user roles")
	}

	// If head (not admin), validate the operation
	if isHead && !isAdmin {
		// Head can only set head or staff roles
		if role != nil && *role != string(obj.RoleHead) && *role != string(obj.RoleStaff) {
			return obj.ErrForbidden("heads can only assign head or staff roles")
		}

		// Head can only operate within their own institution
		if institutionID == nil || *institutionID != currentUser.Role.Institution.ID {
			return obj.ErrForbidden("heads can only manage members within their own institution")
		}

		// Verify target user is in the same institution
		targetUser, err := GetUserByID(ctx, targetUserID)
		if err != nil {
			return obj.ErrNotFound("target user not found")
		}
		if targetUser.Role == nil || targetUser.Role.Institution == nil || targetUser.Role.Institution.ID != currentUser.Role.Institution.ID {
			return obj.ErrForbidden("target user is not in your institution")
		}
	}

	// Validate role name
	if role != nil {
		if _, err := stringToRole(*role); err != nil {
			return err
		}
	}

	// Only individual users can be promoted to admin
	if role != nil && *role == string(obj.RoleAdmin) {
		targetUser, err := GetUserByID(ctx, targetUserID)
		if err != nil {
			return obj.ErrNotFound("target user not found")
		}
		if targetUser.Role == nil || targetUser.Role.Role != obj.RoleIndividual {
			return obj.ErrForbidden("only users with 'individual' role can be promoted to admin")
		}
	}

	// If the target user is losing their admin role, clear the system free-use API key
	// if it references one of their keys (non-admins should not have keys in system settings)
	{
		targetUser, lookupErr := GetUserByID(ctx, targetUserID)
		if lookupErr == nil && targetUser.Role != nil && targetUser.Role.Role == obj.RoleAdmin {
			if role == nil || *role != string(obj.RoleAdmin) {
				_ = ClearSystemSettingsFreeUseApiKeyByOwner(ctx, targetUserID)
			}
		}
	}

	// If removing role (role == nil) or changing institution, clean up API key shares
	// with the user's current institution before the role change
	if role == nil || institutionID != nil {
		targetUser, lookupErr := GetUserByID(ctx, targetUserID)
		if lookupErr == nil && targetUser.Role != nil && targetUser.Role.Institution != nil {
			oldInstID := targetUser.Role.Institution.ID
			// Only clean up if actually leaving the institution (removing role or moving to different institution)
			if role == nil || (institutionID != nil && *institutionID != oldInstID) {
				_ = queries().DeleteApiKeySharesByOwnerForInstitution(ctx, db.DeleteApiKeySharesByOwnerForInstitutionParams{
					UserID:        targetUserID,
					InstitutionID: uuid.NullUUID{UUID: oldInstID, Valid: true},
				})
				_ = queries().DeleteApiKeySharesByOwnerForInstitutionWorkshops(ctx, db.DeleteApiKeySharesByOwnerForInstitutionWorkshopsParams{
					UserID:        targetUserID,
					InstitutionID: oldInstID,
				})
			}
		}
	}

	// Use a transaction to ensure atomicity
	tx, err := sqlDb.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	txQueries := queries().WithTx(tx)

	// Delete existing roles for this user
	if err := txQueries.DeleteUserRoles(ctx, targetUserID); err != nil {
		return fmt.Errorf("failed to delete existing roles: %w", err)
	}

	// No new role specified? Assign "individual" as the default fallback role.
	// Admin/head/staff who lose their role become individual users.
	if role == nil {
		individualRole := string(obj.RoleIndividual)
		role = &individualRole
	}

	// Create the new role
	arg := db.CreateUserRoleParams{
		UserID:        targetUserID,
		Role:          sql.NullString{String: *role, Valid: *role != ""},
		InstitutionID: uuid.NullUUID{UUID: uuidPtrToUUID(institutionID), Valid: institutionID != nil},
		WorkshopID:    uuid.NullUUID{UUID: uuidPtrToUUID(workshopID), Valid: workshopID != nil},
	}
	if _, err := txQueries.CreateUserRole(ctx, arg); err != nil {
		return fmt.Errorf("failed to create user role: %w", err)
	}

	// Commit the transaction
	return tx.Commit()
}

// GetUserStats retrieves statistics for a user
func GetUserStats(ctx context.Context, userID uuid.UUID) (*obj.UserStats, error) {
	sessionsCount, err := queries().CountUserSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count user sessions: %w", err)
	}

	gamesCount, err := queries().CountUserGames(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to count user games: %w", err)
	}

	messagesCount, err := queries().CountUserPlayerMessages(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count user messages: %w", err)
	}

	playCount, err := queries().SumPlayCountOfUserGames(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to sum play count: %w", err)
	}

	return &obj.UserStats{
		GamesPlayed:       int(sessionsCount),
		GamesCreated:      int(gamesCount),
		MessagesSent:      int(messagesCount),
		TotalPlaysOnGames: int(playCount),
	}, nil
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

// ClearActiveWorkshop clears the active workshop for a user (leave workshop mode)
func ClearActiveWorkshop(ctx context.Context, userID uuid.UUID) error {
	err := queries().ClearUserActiveWorkshop(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to clear active workshop: %w", err)
	}
	return nil
}
