package db

import (
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

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
			fmt.Printf("Warning: failed to auto-upgrade user to admin: %v\n", err)
		}
	}

	return GetUserByID(ctx, id)
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
			fmt.Printf("Warning: failed to auto-upgrade user to admin: %v\n", err)
		}
	}

	return GetUserByID(ctx, id)
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

// GetUserByIDRaw gets the raw user record by ID (includes participant_token field)
func GetUserByIDRaw(ctx context.Context, id uuid.UUID) (db.AppUser, error) {
	return queries().GetUserByID(ctx, id)
}

// RemoveUser deletes a user (checks permissions internally)
func RemoveUser(ctx context.Context, currentUserID uuid.UUID, targetUserID uuid.UUID) error {
	if err := CanDeleteUser(ctx, currentUserID, targetUserID); err != nil {
		return err
	}
	return DeleteUser(ctx, targetUserID)
}

// GetUserByParticipantToken gets a user by their participant token
func GetUserByParticipantToken(ctx context.Context, token string) (*obj.User, error) {
	res, err := queries().GetUserByParticipantToken(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		return nil, err
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

// DeleteUser deletes a user
func DeleteUser(ctx context.Context, id uuid.UUID) error {
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

	fmt.Printf("Auto-upgraded user %s to admin role\n", userID)
	return nil
}

// GetAllUsers returns all users (for admin/CLI use)
func GetAllUsers(ctx context.Context) ([]obj.User, error) {
	rows, err := sqlDb.QueryContext(ctx, `
		SELECT id, name, email, auth0_id, created_at
		FROM app_user
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []obj.User
	for rows.Next() {
		var u obj.User
		var email, auth0ID sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Name, &email, &auth0ID, &createdAt); err != nil {
			return nil, err
		}
		u.Email = sqlNullStringToMaybeString(email)
		u.Auth0Id = sqlNullStringToMaybeString(auth0ID)
		u.Meta.CreatedAt = &createdAt
		users = append(users, u)
	}
	return users, rows.Err()
}

func UpdateUserRole(ctx context.Context, currentUserID uuid.UUID, targetUserID uuid.UUID, role *string, institutionID *uuid.UUID, workshopID *uuid.UUID) error {
	// Check permissions
	currentUser, err := GetUserByID(ctx, currentUserID)
	if err != nil {
		return obj.ErrNotFound("current user not found")
	}

	// Only admin can set roles
	if currentUser.Role == nil || currentUser.Role.Role != obj.RoleAdmin {
		return obj.ErrForbidden("only admins can manage user roles")
	}

	// Validate role name
	if role != nil {
		if _, err := stringToRole(*role); err != nil {
			return err
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

	// No new role? Commit and return
	if role == nil {
		return tx.Commit()
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
