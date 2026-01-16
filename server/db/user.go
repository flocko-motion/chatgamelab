package db

import (
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"
	"database/sql"
	"fmt"
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

func UpdateUserSettings(ctx context.Context, id uuid.UUID, showAiModelSelector bool) error {
	arg := db.UpdateUserSettingsParams{
		ID:                  id,
		ShowAiModelSelector: showAiModelSelector,
	}
	return queries().UpdateUserSettings(ctx, arg)
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
		Name:                res.Name,
		Email:               sqlNullStringToMaybeString(res.Email),
		DeletedAt:           &res.DeletedAt.Time,
		Auth0Id:             sqlNullStringToMaybeString(res.Auth0ID),
		ShowAiModelSelector: res.ShowAiModelSelector,
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

func UpdateUserRole(ctx context.Context, userID uuid.UUID, role *string, institutionID *uuid.UUID) error {
	// Validate role name
	if role != nil {
		if _, err := stringToRole(*role); err != nil {
			return err
		}
	}

	// Delete existing roles for this user
	if err := queries().DeleteUserRoles(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete existing roles: %w", err)
	}

	// No new role?
	if role == nil {
		return nil
	}

	// Create the new role
	arg := db.CreateUserRoleParams{
		UserID:        userID,
		Role:          sql.NullString{String: *role, Valid: *role != ""},
		InstitutionID: uuid.NullUUID{UUID: uuidPtrToUUID(institutionID), Valid: institutionID != nil},
	}
	_, err := queries().CreateUserRole(ctx, arg)
	return err
}
