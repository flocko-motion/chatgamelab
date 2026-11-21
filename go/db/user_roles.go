// package: db / database access and repository layer
// type:    data
// job:     role assignment, admin auto-promotion, and admin-facing user listing and stats.
// limits:  does not handle basic user CRUD (-> user.go).
package db

import (
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"
	"errors"
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

// errAdminBootstrapClosed is returned when ADMIN_EMAILS would elevate someone
// after the first admin already exists.
var errAdminBootstrapClosed = errors.New("admin bootstrap closed: an admin already exists")

// adminBootstrapOpen reports whether ADMIN_EMAILS may still elevate anyone.
//
// The list exists to create the first admin of an empty deployment, so it stops
// granting anything once one exists — otherwise editing the environment (or
// registering an address that was left in the list) hands out admin on a live
// system for as long as the deployment lives. Later admins are appointed by an
// existing admin instead.
//
// A failed count closes the gate: refusing a legitimate bootstrap costs one
// restart, granting a wrong one costs the deployment.
func adminBootstrapOpen(ctx context.Context) bool {
	count, err := queries().CountAdmins(ctx)
	if err != nil {
		log.Warn("failed to count admins, refusing admin bootstrap", "error", err)
		return false
	}
	return count == 0
}

// autoUpgradeUserToAdmin creates an admin role for the user
func autoUpgradeUserToAdmin(ctx context.Context, userID uuid.UUID) error {
	if !adminBootstrapOpen(ctx) {
		return errAdminBootstrapClosed
	}

	// Create admin role for the user
	arg := CreateUserRoleParams{
		UserID:        userID,
		Role:          string(obj.RoleAdmin),
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
	arg := CreateUserRoleParams{
		UserID:        userID,
		Role:          string(obj.RoleIndividual),
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

	// Checked before the roles are cleared below, so a closed gate leaves the
	// user's own role intact.
	if !adminBootstrapOpen(ctx) {
		log.Debug("skipping login admin promotion: an admin already exists", "user_id", user.ID)
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

// PromoteAdminEmails promotes the users whose email is in ADMIN_EMAILS to admin,
// for as long as the deployment has no admin yet. Called on server startup.
func PromoteAdminEmails(ctx context.Context) {
	adminEmails := os.Getenv("ADMIN_EMAILS")
	if adminEmails == "" {
		log.Debug("ADMIN_EMAILS not set, skipping admin promotion check")
		return
	}

	if !adminBootstrapOpen(ctx) {
		log.Debug("admin already exists, skipping ADMIN_EMAILS bootstrap")
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
		if user.Role != nil && user.Role.Role == obj.RoleAdmin {
			log.Debug("admin email user is already an admin", "user_id", user.ID, "email", email)
			continue
		}
		if user.Role == nil || user.Role.Role != obj.RoleIndividual {
			log.Warn("skipping admin promotion: role must be individual", "user_id", user.ID, "email", email, "role", user.Role)
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
			Name:      res.Name.String,
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

// GetUserStats retrieves statistics for a user
func GetUserStats(ctx context.Context, userID uuid.UUID) (*obj.UserStats, error) {
	gamesCount, err := queries().CountUserGames(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to count user games: %w", err)
	}

	playCount, err := queries().SumPlayCountOfUserGames(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to sum play count: %w", err)
	}

	sessionsCount, err := queries().CountUserGameSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count user sessions: %w", err)
	}

	messagesCount, err := queries().SumUserGameSessionMessages(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count user messages: %w", err)
	}

	return &obj.UserStats{
		GamesPlayed:       int(sessionsCount),
		GamesCreated:      int(gamesCount),
		MessagesSent:      int(messagesCount),
		TotalPlaysOnGames: int(playCount),
	}, nil
}
