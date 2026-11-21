// package: db / database access and repository layer
// type:    data
// job:     create, read, update, delete users; participant tokens, default shares, and active workshop.
// limits:  does not manage roles/admin promotion or user listings/stats (-> user_roles.go).
package db

import (
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// GuestAuthError represents a specific error during participant authentication
type GuestAuthError struct {
	Code    string // "invalid_token", "workshop_inactive"
	Message string
}

// Error returns the participant authentication error message.
func (e *GuestAuthError) Error() string {
	return e.Message
}

// GuestTokenPrefix starts a guest's login code: "guest-" and its words.
const GuestTokenPrefix = "guest-"

// LegacyGuestTokenPrefix started guests' login codes before GuestTokenPrefix,
// while guests were called participants. Backwards compatibility only: codes
// issued with it stay valid, because guests' printed links and QR codes carry
// them; nothing new is issued with it.
const LegacyGuestTokenPrefix = "participant-"

func isGuestTokenTaken(ctx context.Context, token string) (bool, error) {
	return queries().GuestTokenExists(ctx, sql.NullString{String: token, Valid: true})
}

// GuestTokenKnown reports whether a token was ever issued, including
// tokens of deleted users or workshops. Unknown tokens are guesses.
func GuestTokenKnown(ctx context.Context, token string) bool {
	ok, err := isGuestTokenTaken(ctx, token)
	return ok || err != nil
}

// GetUserByGuestToken gets a user by their participant token
// Returns specific error codes for different failure scenarios
func GetUserByGuestToken(ctx context.Context, token string) (*obj.User, error) {
	res, err := queries().GetUserByGuestToken(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		// Check if the token exists but workshop is inactive
		status, statusErr := queries().CheckGuestTokenStatus(ctx, sql.NullString{String: token, Valid: true})
		if statusErr == nil && status.TokenExists {
			// Token exists but query failed - likely inactive workshop
			workshopActive, ok := status.WorkshopActive.(bool)
			if ok && !workshopActive {
				return nil, &GuestAuthError{
					Code:    "workshop_inactive",
					Message: "Workshop is inactive",
				}
			}
		}
		// Token doesn't exist or other error
		return nil, &GuestAuthError{
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
		Name:          res.Name.String,
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
