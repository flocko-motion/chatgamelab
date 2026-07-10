// package: db / database access and repository layer
// type:    data
// job:     role assignment, admin auto-promotion, and admin-facing user listing and stats.
// limits:  does not handle basic user CRUD (-> user.go).
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

// UpdateUserRole updates a target user's role and its institution/workshop scope.
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
