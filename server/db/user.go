package db

import (
	"context"
	"database/sql"
	"fmt"
	db "webapp-server/db/sqlc"
	"webapp-server/obj"

	"github.com/google/uuid"
)

// CreateUser creates a new user in the database
func CreateUser(ctx context.Context, name string, email *string, auth0ID string) (*obj.User, error) {
	arg := db.CreateUserParams{
		Name:    name,
		Email:   sql.NullString{String: stringPtrToString(email), Valid: email != nil},
		Auth0ID: sql.NullString{String: auth0ID, Valid: auth0ID != ""},
	}

	id, err := queries().CreateUser(ctx, arg)
	if err != nil {
		return nil, err
	}

	return GetUserByID(ctx, id)
}

func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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

// DeleteUser deletes a user
func DeleteUser(ctx context.Context, id uuid.UUID) error {
	return queries().DeleteUser(ctx, id)
}

func shortenOpenaiKey(key string) string {
	if len(key) == 0 {
		return "none"
	} else if len(key) < 12 {
		return "invalid"
	}
	return "sk-..." + key[len(key)-8:]
}

func UpdateUserDetails(ctx context.Context, id uuid.UUID, name string, email *string) error {
	arg := db.UpdateUserParams{
		ID:    id,
		Name:  name,
		Email: sql.NullString{String: stringPtrToString(email), Valid: email != nil},
	}
	return queries().UpdateUser(ctx, arg)
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
		Role:          sql.NullString{String: *role, Valid: role != nil && *role != ""},
		InstitutionID: uuid.NullUUID{UUID: uuidPtrToUUID(institutionID), Valid: institutionID != nil},
	}
	_, err := queries().CreateUserRole(ctx, arg)
	return err
}

func uuidPtrToUUID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.UUID{}
	}
	return *id
}
