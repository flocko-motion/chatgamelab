package db

import (
	"cgl/ai"
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

func userIsAllowedToUseApiKey(ctx context.Context, userID, apiKeyID uuid.UUID) error {
	// Validate that user is allowed to use this API key
	apiKeys, err := GetUserApiKeys(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user api keys: %w", err)
	}
	var foundKey *obj.ApiKeyShare
	for i := range apiKeys {
		if apiKeys[i].ApiKey != nil && apiKeys[i].ApiKey.ID == apiKeyID {
			foundKey = &apiKeys[i]
			break
		}
	}
	if foundKey == nil {
		return errors.New("access denied: user does not have access to this API key")
	}
	return nil
}

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
	user.ApiKeys, err = GetUserApiKeys(ctx, id)
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

func CreateUserApiKey(ctx context.Context, userID uuid.UUID, platform, key string) (*uuid.UUID, error) {
	if !slices.Contains(ai.ApiKeyPlatforms, platform) {
		return nil, errors.New("unknown platform: " + platform)
	}

	now := time.Now()
	arg := db.CreateApiKeyParams{
		ID:         uuid.New(),
		CreatedBy:  uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:  now,
		ModifiedBy: uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt: now,
		UserID:     userID,
		Platform:   platform,
		Key:        key,
	}
	result, err := queries().CreateApiKey(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &result.ID, nil
}

func DeleteUserApiKey(ctx context.Context, userID uuid.UUID, apiKeyID string) error {
	id, err := uuid.Parse(apiKeyID)
	if err != nil {
		return errors.New("invalid api key id")
	}

	// Verify the key belongs to the user
	key, err := queries().GetApiKeyByID(ctx, id)
	if err != nil {
		return err
	}
	if key.UserID != userID {
		return errors.New("api key does not belong to user")
	}

	return queries().DeleteApiKey(ctx, id)
}

func GetUserApiKeys(ctx context.Context, userID uuid.UUID) ([]obj.ApiKeyShare, error) {
	var result []obj.ApiKeyShare

	// 1. Get user's own API keys (treated as "shared with self")
	ownKeys, err := queries().GetUserApiKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get own api keys: %w", err)
	}
	for _, k := range ownKeys {
		result = append(result, obj.ApiKeyShare{
			ID: k.ID,
			Meta: obj.Meta{
				CreatedBy:  k.CreatedBy,
				CreatedAt:  &k.CreatedAt,
				ModifiedBy: k.ModifiedBy,
				ModifiedAt: &k.ModifiedAt,
			},
			ApiKey: &obj.ApiKey{
				ID:           k.ID,
				UserID:       k.UserID,
				Platform:     k.Platform,
				Key:          k.Key,
				KeyShortened: shortenApiKey(k.Key),
			},
			UserID:                    &userID,
			AllowPublicSponsoredPlays: true, // own keys have full access
		})
	}

	// 2. Get API keys shared with this user by others
	sharedKeys, err := queries().GetApiKeySharesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared api keys: %w", err)
	}
	for _, s := range sharedKeys {
		result = append(result, obj.ApiKeyShare{
			ID: s.ID,
			Meta: obj.Meta{
				CreatedBy:  s.CreatedBy,
				CreatedAt:  &s.CreatedAt,
				ModifiedBy: s.ModifiedBy,
				ModifiedAt: &s.ModifiedAt,
			},
			ApiKey: &obj.ApiKey{
				ID:           s.ApiKeyID,
				Platform:     s.ApiKeyPlatform,
				Key:          s.ApiKeyKey,
				KeyShortened: shortenApiKey(s.ApiKeyKey),
			},
			UserID:                    &s.UserID,
			AllowPublicSponsoredPlays: s.AllowPublicSponsoredPlays,
		})
	}

	return result, nil
}

func uuidPtrToUUID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.UUID{}
	}
	return *id
}

func shortenApiKey(key string) string {
	toLen := 6
	if len(key) >= 6 {
		return ".." + key[len(key)-toLen-1:len(key)-1]
	}
	return key
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
