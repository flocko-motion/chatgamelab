// package: db / database access and repository layer
// type:    logic
// job:     evaluate whether a user may perform CRUD on games, game sessions, API keys, and sponsorship shares.
// limits:  does not authorize institutions/workshops/users (-> permissions.go).
package db

import (
	sqlc "cgl/db/sqlc"
	"cgl/log"
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

// canAccessGame checks if user can perform a CRUD operation on a game
// - operation: the type of CRUD operation (create, read, update, delete, list)
// - game: the game object (nil for create/list operations)
// - shareToken: optional share token provided by user (for private share links)
func canAccessGame(ctx context.Context, userID uuid.UUID, operation CRUDOperation, game *obj.Game, shareToken *string) error {
	// Admin can perform any operation on any game
	if operation != OpCreate {
		adminUser, _ := GetUserByID(ctx, userID)
		if adminUser != nil && adminUser.Role != nil && adminUser.Role.Role == obj.RoleAdmin {
			return nil
		}
	}

	switch operation {
	case OpCreate:
		// Any authenticated user can create games
		return nil

	case OpRead:
		if game == nil {
			return obj.ErrValidation("game required for read operation")
		}

		// 1. Owner can always read their game
		if game.Meta.CreatedBy.Valid && game.Meta.CreatedBy.UUID == userID {
			return nil
		}

		// 2. Public games can be read by anyone
		if game.Public {
			return nil
		}

		// 3. Valid share token grants access (checked against game_share table)
		if shareToken != nil {
			gs, err := queries().GetGameShareByToken(ctx, *shareToken)
			if err == nil && gs.GameID == game.ID {
				return nil
			}
		}

		// 4. Workshop members can access workshop games
		if game.WorkshopID != nil {
			user, err := GetUserByID(ctx, userID)
			if err == nil && user.Role != nil {
				// Participant or individual with role for this specific workshop can read
				if (user.Role.Role == obj.RoleParticipant || user.Role.Role == obj.RoleIndividual) && user.Role.Workshop != nil && user.Role.Workshop.ID == *game.WorkshopID {
					return nil
				}
				// Head/staff of the workshop's institution can read
				if (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) && user.Role.Institution != nil {
					ws, wsErr := queries().GetWorkshopByID(ctx, *game.WorkshopID)
					if wsErr == nil && ws.InstitutionID == user.Role.Institution.ID {
						return nil
					}
				}
			}
		}

		return obj.ErrForbidden("not authorized to read this game")

	case OpList:
		// Any authenticated user can list games (filtered by visibility in query)
		return nil

	case OpUpdate, OpDelete:
		if game == nil {
			return obj.ErrValidation("game required for update/delete operation")
		}
		// Owner can update/delete
		if game.Meta.CreatedBy.Valid && game.Meta.CreatedBy.UUID == userID {
			return nil
		}
		// If game belongs to a workshop, head/staff of the workshop's institution can update/delete
		if game.WorkshopID != nil {
			user, err := GetUserByID(ctx, userID)
			if err == nil && user.Role != nil && user.Role.Institution != nil &&
				(user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) {
				// Verify the workshop belongs to the user's institution
				ws, wsErr := queries().GetWorkshopByID(ctx, *game.WorkshopID)
				if wsErr == nil && ws.InstitutionID == user.Role.Institution.ID {
					return nil
				}
			}
		}
		return obj.ErrForbidden("only the owner or institution head/staff can modify this game")

	default:
		return obj.ErrForbidden("unknown operation")
	}
}

// canAccessGameSession checks if user can perform a CRUD operation on a game session
// - operation: the type of CRUD operation (create, read, update, delete, list)
// - session: the session object (nil for list operations)
// - gameID: the game ID (for create operation to check game's workshop)
// - workshopID: optional workshop context for create operation
func canAccessGameSession(ctx context.Context, userID uuid.UUID, operation CRUDOperation, session *obj.GameSession, gameID uuid.UUID, workshopID *uuid.UUID) error {
	switch operation {
	case OpCreate:
		// Load game to check if it belongs to a workshop
		game, err := queries().GetGameByID(ctx, gameID)
		if err != nil {
			return obj.ErrNotFound("game not found")
		}

		// Public games can be played by anyone
		if game.Public {
			return nil
		}

		// If game belongs to a workshop, user must have read access to that workshop.
		// Guest users (created via share tokens) are pre-authorized, so skip this check.
		if game.WorkshopID.Valid {
			isGuest := false
			if userID != uuid.Nil {
				appUser, err := queries().GetUserByID(ctx, userID)
				if err == nil && appUser.PrivateShareID.Valid {
					isGuest = true
				}
			} else {
				isGuest = true
			}
			if !isGuest {
				workshop, err := queries().GetWorkshopByID(ctx, game.WorkshopID.UUID)
				if err != nil {
					return obj.ErrNotFound("workshop not found")
				}
				if err := canAccessWorkshop(ctx, userID, OpRead, workshop.InstitutionID, &game.WorkshopID.UUID, uuid.Nil); err != nil {
					return obj.ErrForbidden("not authorized to play games in this workshop")
				}
			}
		}

		// If explicit workshopID is provided, validate access to it as well
		if workshopID != nil {
			// Get the workshop to find its institution ID
			workshop, err := queries().GetWorkshopByID(ctx, *workshopID)
			if err != nil {
				return obj.ErrNotFound("workshop not found")
			}
			if err := canAccessWorkshop(ctx, userID, OpRead, workshop.InstitutionID, workshopID, uuid.Nil); err != nil {
				return obj.ErrForbidden("not authorized to create sessions in this workshop")
			}
		}

		// Otherwise any authenticated user can create personal sessions
		return nil

	case OpRead:
		if session == nil {
			return obj.ErrValidation("session required for read operation")
		}

		// 1. Owner can always read their session
		if session.UserID == userID {
			return nil
		}

		// 2. Workshop-based sessions can be read by workshop staff/head
		if session.WorkshopID != nil {
			user, err := GetUserByID(ctx, userID)
			if err == nil && user.Role != nil {
				// Staff who has role for this workshop can read sessions
				if user.Role.Role == obj.RoleStaff && user.Role.Workshop != nil && user.Role.Workshop.ID == *session.WorkshopID {
					return nil
				}
				// Head of institution can read all workshop sessions
				if user.Role.Role == obj.RoleHead && user.Role.Institution != nil {
					return nil
				}
			}
		}

		return obj.ErrForbidden("not authorized to read this session")

	case OpList:
		// Users can only list their own sessions (filtered in query)
		return nil

	case OpUpdate:
		if session == nil {
			return obj.ErrValidation("session required for update operation")
		}
		// Only owner can update (play) their session
		if session.UserID == userID {
			return nil
		}
		return obj.ErrForbidden("only the owner can update this session")

	case OpDelete:
		if session == nil {
			return obj.ErrValidation("session required for delete operation")
		}
		// Owner can delete their session
		if session.UserID == userID {
			return nil
		}
		// If session is in workshop context, staff/head can delete
		if session.WorkshopID != nil {
			user, err := GetUserByID(ctx, userID)
			if err == nil && user.Role != nil {
				// Staff who owns the workshop can delete sessions
				if user.Role.Role == obj.RoleStaff && user.Role.Workshop != nil && user.Role.Workshop.ID == *session.WorkshopID {
					return nil
				}
				// Head of institution can delete all workshop sessions
				if user.Role.Role == obj.RoleHead && user.Role.Institution != nil {
					return nil
				}
			}
		}
		return obj.ErrForbidden("only the owner, workshop staff, or institution head can delete this session")

	default:
		return obj.ErrForbidden("unknown operation")
	}
}

// canAccessApiKey checks if user can perform a CRUD operation on an API key
// - operation: the type of CRUD operation (create, read, update, delete, list)
// - apiKeyID: the API key ID (for read operation to check shares)
// - keyOwnerID: the user who owns the API key (only needed for update/delete)
// - gameID: optional game ID for sponsorship context
// - sessionID: optional session ID for sponsorship context
// - workshopID: optional workshop ID for sponsorship context
func canAccessApiKey(ctx context.Context, userID uuid.UUID, operation CRUDOperation, apiKeyID uuid.UUID, keyOwnerID uuid.UUID, gameID *uuid.UUID, sessionID *uuid.UUID, workshopID *uuid.UUID) error {
	switch operation {
	case OpCreate:
		// Any authenticated user can create API keys
		return nil

	case OpRead:
		// Owner can read their API key
		if keyOwnerID == userID {
			return nil
		}
		// Check if user has access via api_key_share
		// Users can access keys shared with them (user_id), their workshop, or their institution
		user, err := GetUserByID(ctx, userID)
		if err == nil {
			// Check for direct user share
			shares, err := queries().GetApiKeySharesByApiKeyID(ctx, apiKeyID)
			if err == nil {
				log.Debug("checking API key shares for access",
					"user_id", userID,
					"api_key_id", apiKeyID,
					"share_count", len(shares))
				for _, share := range shares {
					// Direct user share
					if share.UserID.Valid && share.UserID.UUID == userID {
						log.Debug("access granted via direct user share")
						return nil
					}
					// Workshop share - check if user is an active member of the workshop
					if share.WorkshopID.Valid {
						log.Debug("checking workshop membership",
							"user_id", userID,
							"workshop_id", share.WorkshopID.UUID)
						isMember, err := queries().IsUserInWorkshop(ctx, sqlc.IsUserInWorkshopParams{
							UserID:     userID,
							WorkshopID: share.WorkshopID,
						})
						log.Debug("workshop membership check result",
							"is_member", isMember,
							"error", err)
						if err == nil && isMember {
							log.Debug("access granted via workshop membership")
							return nil
						}
					}
					// Institution share
					if share.InstitutionID.Valid && user.Role != nil && user.Role.Institution != nil && share.InstitutionID.UUID == user.Role.Institution.ID {
						log.Debug("access granted via institution share")
						return nil
					}
					log.Debug("share did not match",
						"share_user_id", share.UserID,
						"share_workshop_id", share.WorkshopID,
						"share_institution_id", share.InstitutionID)
				}
			} else {
				log.Debug("failed to get API key shares", "error", err)
			}
		} else {
			log.Debug("failed to get user", "error", err)
		}

		// Check sponsorship context
		if gameID != nil {
			// Load game to check if this key sponsors it (via share)
			game, err := queries().GetGameByID(ctx, *gameID)
			if err == nil {
				// Public game with sponsored key share
				if game.Public && game.PublicSponsoredApiKeyShareID.Valid {
					share, err := queries().GetApiKeyShareByID(ctx, game.PublicSponsoredApiKeyShareID.UUID)
					if err == nil && share.ApiKeyID == apiKeyID {
						return nil
					}
				}
				// Game share sponsored key (from game_share table)
				gameShares, gsErr := queries().GetGameSharesByGameID(ctx, *gameID)
				if gsErr == nil {
					for _, gs := range gameShares {
						gsShare, gsShareErr := queries().GetApiKeyShareByID(ctx, gs.ApiKeyShareID)
						if gsShareErr == nil && gsShare.ApiKeyID == apiKeyID {
							return nil
						}
					}
				}
			}
		}

		// Check workshop context for sponsored keys
		if workshopID != nil {
			// Check if key is shared with this workshop
			shares, err := queries().GetApiKeySharesByApiKeyID(ctx, apiKeyID)
			if err == nil {
				for _, share := range shares {
					if share.WorkshopID.Valid && share.WorkshopID.UUID == *workshopID {
						return nil
					}
				}
			}
		}

		return obj.ErrForbidden("not authorized to read this API key")

	case OpList:
		// Users can list their own API keys plus keys shared with them (filtered in query)
		return nil

	case OpUpdate, OpDelete:
		// Only owner can update/delete
		if keyOwnerID == userID {
			return nil
		}
		return obj.ErrForbidden("only the owner can modify this API key")

	default:
		return obj.ErrForbidden("unknown operation")
	}
}

// canUseShareForSponsoring checks if a user is authorized to use an API key share for game sponsoring.
// Allowed if:
//   - The user owns the underlying API key, OR
//   - The share belongs to an institution and the user is head/staff of that institution
func canUseShareForSponsoring(ctx context.Context, userID uuid.UUID, share sqlc.GetApiKeyShareByIDRow) error {
	if share.KeyOwnerID == userID {
		return nil
	}
	if share.InstitutionID.Valid {
		caller, err := GetUserByID(ctx, userID)
		if err == nil && caller.Role != nil && caller.Role.Institution != nil &&
			caller.Role.Institution.ID == share.InstitutionID.UUID &&
			(caller.Role.Role == obj.RoleHead || caller.Role.Role == obj.RoleStaff) {
			return nil
		}
	}
	return obj.ErrForbidden("only the key owner or an institution head/staff can sponsor a game with this key")
}
