package db

import (
	db "cgl/db/sqlc"
	"cgl/game/ai"
	"cgl/obj"
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// CreateUserWithID creates a new user with a specific UUID
func CreateUserWithID(ctx context.Context, id uuid.UUID, name string, email *string, auth0ID string) (*obj.User, error) {
	arg := db.CreateUserWithIDParams{
		ID:      id,
		Name:    name,
		Email:   sql.NullString{String: stringPtrToString(email), Valid: email != nil},
		Auth0ID: sql.NullString{String: auth0ID, Valid: auth0ID != ""},
	}

	_, err := queries().CreateUserWithID(ctx, arg)
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

func CreateUserApiKey(ctx context.Context, userID uuid.UUID, name, platform, key string) (*uuid.UUID, error) {
	if !ai.IsValidApiKeyPlatform(platform) {
		return nil, errors.New("unknown platform: " + platform)
	}

	now := time.Now()
	arg := db.CreateApiKeyParams{
		CreatedBy:  uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:  now,
		ModifiedBy: uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt: now,
		UserID:     userID,
		Name:       name,
		Platform:   platform,
		Key:        key,
	}
	result, err := queries().CreateApiKey(ctx, arg)
	if err != nil {
		return nil, err
	}

	// Create a self-share so the user can access their own key via the shares API
	if _, err := createApiKeyShareInternal(ctx, userID, result.ID, &userID, nil, nil, true); err != nil {
		return nil, fmt.Errorf("failed to create self-share: %w", err)
	}

	return &result.ID, nil
}

// GetApiKeyByID gets an API key by ID. If userID is provided, verifies ownership or access.
func GetApiKeyByID(ctx context.Context, userID *uuid.UUID, apiKeyID uuid.UUID) (*obj.ApiKey, error) {
	k, err := queries().GetApiKeyByID(ctx, apiKeyID)
	if err != nil {
		return nil, fmt.Errorf("api key not found: %w", err)
	}

	// If userID provided, verify ownership or access
	if userID != nil && k.UserID != *userID {
		// Check if user has access via sharing
		if err := userIsAllowedToUseApiKey(ctx, *userID, apiKeyID); err != nil {
			return nil, errors.New("access denied: not the owner of this API key")
		}
	}

	return &obj.ApiKey{
		ID:       k.ID,
		UserID:   k.UserID,
		Name:     k.Name,
		Platform: k.Platform,
		Key:      k.Key,
		Meta: obj.Meta{
			CreatedBy:  k.CreatedBy,
			CreatedAt:  &k.CreatedAt,
			ModifiedBy: k.ModifiedBy,
			ModifiedAt: &k.ModifiedAt,
		},
		KeyShortened: shortenApiKey(k.Key),
	}, nil
}

func UpdateUserApiKeyName(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID, name string) error {
	// Verify the key belongs to the user
	key, err := queries().GetApiKeyByID(ctx, apiKeyID)
	if err != nil {
		return err
	}
	if key.UserID != userID {
		return errors.New("api key does not belong to user")
	}

	now := time.Now()
	_, err = queries().UpdateApiKey(ctx, db.UpdateApiKeyParams{
		ID:         apiKeyID,
		ModifiedBy: uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt: now,
		Name:       name,
	})
	return err
}

func DeleteUserApiKey(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID) error {
	// Verify the key belongs to the user
	key, err := queries().GetApiKeyByID(ctx, apiKeyID)
	if err != nil {
		return err
	}
	if key.UserID != userID {
		return errors.New("api key does not belong to user")
	}

	// Delete all shares first
	if err := queries().DeleteApiKeySharesByApiKeyID(ctx, apiKeyID); err != nil {
		return fmt.Errorf("failed to delete shares: %w", err)
	}

	return queries().DeleteApiKey(ctx, db.DeleteApiKeyParams{
		ID:     apiKeyID,
		UserID: userID,
	})
}

func GetUserApiKeys(ctx context.Context, userID uuid.UUID) ([]obj.ApiKeyShare, error) {
	// All API keys (own and shared) are now accessed via shares
	// Own keys have a self-share created when the key is created
	sharedKeys, err := queries().GetApiKeySharesByUserID(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get api key shares: %w", err)
	}

	result := make([]obj.ApiKeyShare, 0, len(sharedKeys))
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
				UserID:       s.OwnerID,
				UserName:     s.OwnerName,
				Name:         s.ApiKeyName,
				Platform:     s.ApiKeyPlatform,
				Key:          s.ApiKeyKey,
				KeyShortened: shortenApiKey(s.ApiKeyKey),
			},
			AllowPublicSponsoredPlays: s.AllowPublicSponsoredPlays,
		})
	}

	return result, nil
}

// GetApiKeyShares returns all shares for an API key. Only the owner can view shares.
func GetApiKeyShares(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID) ([]obj.ApiKeyShare, error) {
	// Verify the key belongs to the user
	key, err := queries().GetApiKeyByID(ctx, apiKeyID)
	if err != nil {
		return nil, fmt.Errorf("api key not found: %w", err)
	}
	if key.UserID != userID {
		return nil, errors.New("api key does not belong to user")
	}

	shares, err := queries().GetApiKeySharesByApiKeyID(ctx, apiKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shares: %w", err)
	}

	result := make([]obj.ApiKeyShare, 0, len(shares))
	for _, s := range shares {
		share := obj.ApiKeyShare{
			ID: s.ID,
			Meta: obj.Meta{
				CreatedBy:  s.CreatedBy,
				CreatedAt:  &s.CreatedAt,
				ModifiedBy: s.ModifiedBy,
				ModifiedAt: &s.ModifiedAt,
			},
			AllowPublicSponsoredPlays: s.AllowPublicSponsoredPlays,
		}

		if s.UserID.Valid {
			share.User = &obj.User{ID: s.UserID.UUID, Name: s.UserName.String}
		}
		if s.WorkshopID.Valid {
			share.Workshop = &obj.Workshop{ID: s.WorkshopID.UUID, Name: s.WorkshopName.String}
		}
		if s.InstitutionID.Valid {
			share.Institution = &obj.Institution{ID: s.InstitutionID.UUID, Name: s.InstitutionName.String}
		}

		result = append(result, share)
	}

	return result, nil
}

// CreateApiKeyShare creates a new share for an API key. Verifies ownership first.
func CreateApiKeyShare(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID, targetUserID, workshopID, institutionID *uuid.UUID, allowPublic bool) (*uuid.UUID, error) {
	// Verify the key belongs to the user
	key, err := queries().GetApiKeyByID(ctx, apiKeyID)
	if err != nil {
		return nil, fmt.Errorf("api key not found: %w", err)
	}
	if key.UserID != userID {
		return nil, errors.New("api key does not belong to user")
	}

	return createApiKeyShareInternal(ctx, userID, apiKeyID, targetUserID, workshopID, institutionID, allowPublic)
}

// createApiKeyShareInternal creates a share without ownership verification (for internal use)
func createApiKeyShareInternal(ctx context.Context, userID uuid.UUID, apiKeyID uuid.UUID, targetUserID, workshopID, institutionID *uuid.UUID, allowPublic bool) (*uuid.UUID, error) {
	now := time.Now()
	arg := db.CreateApiKeyShareParams{
		CreatedBy:                 uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt:                 now,
		ModifiedBy:                uuid.NullUUID{UUID: userID, Valid: true},
		ModifiedAt:                now,
		ApiKeyID:                  apiKeyID,
		UserID:                    uuidPtrToNullUUID(targetUserID),
		WorkshopID:                uuidPtrToNullUUID(workshopID),
		InstitutionID:             uuidPtrToNullUUID(institutionID),
		AllowPublicSponsoredPlays: allowPublic,
	}

	result, err := queries().CreateApiKeyShare(ctx, arg)
	if err != nil {
		return nil, err
	}
	return &result.ID, nil
}

// DeleteApiKeyShare deletes a share. Only the API key owner can delete shares.
func DeleteApiKeyShare(ctx context.Context, userID uuid.UUID, shareID uuid.UUID) error {
	// Get the share to find the API key
	share, err := queries().GetApiKeyShareByID(ctx, shareID)
	if err != nil {
		return fmt.Errorf("share not found: %w", err)
	}

	// Verify the API key belongs to the user
	key, err := queries().GetApiKeyByID(ctx, share.ApiKeyID)
	if err != nil {
		return fmt.Errorf("api key not found: %w", err)
	}
	if key.UserID != userID {
		return errors.New("api key does not belong to user")
	}

	return queries().DeleteApiKeyShare(ctx, shareID)
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
