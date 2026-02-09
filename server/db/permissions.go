package db

import (
	sqlc "cgl/db/sqlc"
	"cgl/log"
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

// CRUDOperation represents the type of operation being performed
type CRUDOperation string

const (
	OpCreate CRUDOperation = "create"
	OpRead   CRUDOperation = "read"
	OpUpdate CRUDOperation = "update"
	OpDelete CRUDOperation = "delete"
	OpList   CRUDOperation = "list"
)

// canAccessInstitution checks if user can perform a CRUD operation on an institution
// - operation: the type of CRUD operation (create, read, update, delete, list)
// - institutionID: pointer to the institution ID (nil for create/list all operations)
func canAccessInstitution(ctx context.Context, userID uuid.UUID, operation CRUDOperation, institutionID *uuid.UUID) error {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	// Admin can do everything
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	switch operation {
	case OpCreate:
		// Only admin can create institutions
		return obj.ErrForbidden("only admins can create institutions")

	case OpList:
		// Admin can list all institutions
		// Heads and staff can list institutions they're members of (filtered in query)
		if user.Role != nil && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) {
			return nil
		}
		return obj.ErrForbidden("only admins, heads, or staff can list institutions")

	case OpRead:
		// Anyone can read institution public data (name, ID, etc.)
		// Members list is conditionally included based on canAccessInstitutionMembers
		if institutionID == nil {
			return obj.ErrValidation("institutionID required for read operation")
		}
		return nil

	case OpUpdate:
		// Admin can update any institution
		// Head can update their own institution
		if institutionID == nil {
			return obj.ErrValidation("institutionID required for update operation")
		}
		if user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == *institutionID {
			if user.Role.Role == obj.RoleHead {
				return nil
			}
		}
		return obj.ErrForbidden("only admin or head of institution can update this institution")

	case OpDelete:
		// Only admin can delete institutions
		return obj.ErrForbidden("only admins can delete institutions")

	default:
		return obj.ErrForbidden("unknown operation")
	}
}

// canAccessInstitutionMembers checks if user can perform operations on institution members
// - operation: OpRead (list members) or OpDelete (remove member)
// - institutionID: the institution ID
// - targetUserID: the user being accessed (for delete operations, nil for list)
func canAccessInstitutionMembers(ctx context.Context, userID uuid.UUID, operation CRUDOperation, institutionID uuid.UUID, targetUserID *uuid.UUID) error {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	// Admin can do everything
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	switch operation {
	case OpRead:
		// Members can view other members of their institution
		if user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == institutionID {
			return nil
		}
		return obj.ErrForbidden("only members can view institution members")

	case OpDelete:
		// Heads can remove members from their institution
		if user.Role != nil && user.Role.Role == obj.RoleHead &&
			user.Role.Institution != nil && user.Role.Institution.ID == institutionID {
			// Get target user to check their role
			targetUser, err := GetUserByID(ctx, *targetUserID)
			if err != nil {
				return obj.ErrNotFound("target user not found")
			}

			// If target is a head, apply additional validations
			if targetUser.Role != nil && targetUser.Role.Role == obj.RoleHead &&
				targetUser.Role.Institution != nil && targetUser.Role.Institution.ID == institutionID {
				// Heads cannot remove themselves
				if *targetUserID == userID {
					return obj.ErrForbidden("heads cannot remove themselves from an institution")
				}

				// Count heads in this institution
				members, err := GetInstitutionMembers(ctx, institutionID, userID)
				if err != nil {
					return obj.ErrServerError("failed to get institution members")
				}

				headCount := 0
				for _, member := range members {
					if member.Role != nil && member.Role.Role == obj.RoleHead {
						headCount++
					}
				}

				// Prevent removing the last head
				if headCount <= 1 {
					return obj.ErrForbidden("cannot remove the last head from an institution")
				}
			}

			return nil
		}

		// Non-head users can remove themselves (leave the organization)
		if targetUserID != nil && *targetUserID == userID {
			// Check that the user is actually a member of this institution
			if user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == institutionID {
				return nil
			}
			return obj.ErrForbidden("you are not a member of this institution")
		}

		// Staff can remove participants (no role or participant role)
		if user.Role != nil && user.Role.Role == obj.RoleStaff &&
			user.Role.Institution != nil && user.Role.Institution.ID == institutionID {
			// Get target user to check their role
			targetUser, err := GetUserByID(ctx, *targetUserID)
			if err != nil {
				return obj.ErrNotFound("target user not found")
			}
			// Staff can only remove participants
			if targetUser.Role == nil || targetUser.Role.Role == obj.RoleParticipant {
				return nil
			}
			return obj.ErrForbidden("staff can only remove participants")
		}

		return obj.ErrForbidden("only heads, staff, or admins can remove members")

	default:
		return obj.ErrForbidden("unknown operation")
	}
}

// canAccessWorkshopInvites checks if user can view workshop invites
// Only staff, heads, and admins of the institution can view workshop invites
func canAccessWorkshopInvites(ctx context.Context, userID uuid.UUID, institutionID uuid.UUID) error {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	// Admin can view all invites
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	// Head or Staff of the institution can view invites
	if user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == institutionID {
		if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
			return nil
		}
	}

	return obj.ErrForbidden("only admin, head, or staff can view workshop invites")
}

// canAccessWorkshopParticipantTokens checks if user can view workshop participant access tokens
// Only staff and heads of the institution owning the workshop can access tokens
// targetUserID is optional - if provided, validates that the target is actually a workshop participant
func canAccessWorkshopParticipantTokens(ctx context.Context, userID uuid.UUID, institutionID uuid.UUID, targetUserID *uuid.UUID) error {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	// Admin can access any participant token
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	// If target user is specified, verify they are a workshop participant
	if targetUserID != nil {
		targetUser, err := GetUserByID(ctx, *targetUserID)
		if err != nil {
			return obj.ErrNotFound("participant not found")
		}

		// Verify the target is a participant with a workshop role and institution
		if targetUser.Role == nil || targetUser.Role.Role != obj.RoleParticipant ||
			targetUser.Role.Workshop == nil || targetUser.Role.Institution == nil {
			return obj.ErrNotFound("user is not a workshop participant")
		}

		// Use the target's institution for permission check
		// (For workshop participants, the institution in their role IS the workshop's institution)
		institutionID = targetUser.Role.Institution.ID
	}

	// Staff or head of the institution can access participant tokens
	if user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == institutionID {
		if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
			return nil
		}
	}

	return obj.ErrForbidden("only staff and heads of the institution can access participant tokens")
}

// canAccessWorkshopParticipants checks if user can view workshop participants
// Only participants, workshop owner, and institution heads can view the participant list
func canAccessWorkshopParticipants(ctx context.Context, userID uuid.UUID, workshopID uuid.UUID, workshopCreatedBy uuid.UUID, institutionID uuid.UUID) error {
	// Workshop owner can always see participants
	if workshopCreatedBy == userID {
		return nil
	}

	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	// Admin can view all participants
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	// Participant in this workshop can see other participants
	if user.Role != nil && user.Role.Workshop != nil && user.Role.Workshop.ID == workshopID && user.Role.Role == obj.RoleParticipant {
		return nil
	}

	// Head or Staff of the institution that owns this workshop can see participants
	if user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == institutionID && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) {
		return nil
	}

	return obj.ErrForbidden("only participants, workshop owner, or institution staff/head can view participant list")
}

// CanDeleteUser checks if user can delete another user
// - Admins can delete any user
// - Staff/heads can only delete anonymous participants (those with participant_token) in their institution
func CanDeleteUser(ctx context.Context, currentUserID uuid.UUID, targetUserID uuid.UUID) error {
	currentUser, err := GetUserByID(ctx, currentUserID)
	if err != nil {
		return obj.ErrNotFound("current user not found")
	}

	// Admin can delete any user
	if currentUser.Role != nil && currentUser.Role.Role == obj.RoleAdmin {
		return nil
	}

	// Staff/heads can only delete anonymous participants in their institution
	if currentUser.Role != nil && (currentUser.Role.Role == obj.RoleStaff || currentUser.Role.Role == obj.RoleHead) {
		// Get target user
		targetUser, err := GetUserByID(ctx, targetUserID)
		if err != nil {
			return obj.ErrNotFound("target user not found")
		}

		// Must be a participant
		if targetUser.Role == nil || targetUser.Role.Role != obj.RoleParticipant {
			return obj.ErrForbidden("can only delete anonymous participants")
		}

		// Must be in the same institution
		if targetUser.Role.Institution == nil || currentUser.Role.Institution == nil ||
			targetUser.Role.Institution.ID != currentUser.Role.Institution.ID {
			return obj.ErrForbidden("can only delete participants in your institution")
		}

		// Must be an anonymous participant (no auth0_id)
		// Regular users with Auth0 accounts cannot be deleted by staff/heads, even if they're participants
		rawUser, err := GetUserByIDRaw(ctx, targetUserID)
		if err != nil {
			return obj.ErrNotFound("target user not found")
		}

		// Check if this is truly an anonymous user (no Auth0 account)
		if rawUser.Auth0ID.Valid && rawUser.Auth0ID.String != "" {
			return obj.ErrForbidden("can only delete anonymous participants, not regular users")
		}

		return nil
	}

	return obj.ErrForbidden("insufficient permissions to delete users")
}

// CanUpdateParticipantName checks if user can update a participant's name
// - Admins can update any user
// - Staff/heads can only update participants in their institution's workshops
func CanUpdateParticipantName(ctx context.Context, currentUserID uuid.UUID, targetUserID uuid.UUID) error {
	currentUser, err := GetUserByID(ctx, currentUserID)
	if err != nil {
		return obj.ErrNotFound("current user not found")
	}

	// Admin can update anyone
	if currentUser.Role != nil && currentUser.Role.Role == obj.RoleAdmin {
		return nil
	}

	// Staff/heads can only update participants in their institution
	if currentUser.Role != nil && (currentUser.Role.Role == obj.RoleStaff || currentUser.Role.Role == obj.RoleHead) {
		// Get target user
		targetUser, err := GetUserByID(ctx, targetUserID)
		if err != nil {
			return obj.ErrNotFound("target user not found")
		}

		// Must be a participant
		if targetUser.Role == nil || targetUser.Role.Role != obj.RoleParticipant {
			return obj.ErrForbidden("can only update participant names")
		}

		// Must be in the same institution
		if targetUser.Role.Institution == nil || currentUser.Role.Institution == nil ||
			targetUser.Role.Institution.ID != currentUser.Role.Institution.ID {
			return obj.ErrForbidden("can only update participants in your institution")
		}

		return nil
	}

	return obj.ErrForbidden("insufficient permissions to update participant name")
}

// canAccessWorkshop checks if user can perform a CRUD operation on a workshop
// - operation: the type of CRUD operation (create, read, update, delete, list)
// - institutionID: the institution the workshop belongs to
// - workshopID: pointer to workshop ID (needed for participant read checks)
// - createdBy: the user who created the workshop (only needed for update/delete)
func canAccessWorkshop(ctx context.Context, userID uuid.UUID, operation CRUDOperation, institutionID uuid.UUID, workshopID *uuid.UUID, createdBy uuid.UUID) error {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	// Admin can do everything
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	// Members (head/staff/participant) can access their institution's workshops
	if user.Role == nil || user.Role.Institution == nil || user.Role.Institution.ID != institutionID {
		return obj.ErrForbidden("not authorized to access workshops for this institution")
	}

	switch operation {
	case OpCreate:
		// Head or Staff can create workshops
		if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
			return nil
		}
		return obj.ErrForbidden("only admin, head, or staff can create workshops")

	case OpRead:
		// Head or Staff can read any workshop of their institution
		if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
			return nil
		}
		// Participants can only read specific workshops they have a role for
		if user.Role.Role == obj.RoleParticipant && workshopID != nil {
			// Check if user's role is scoped to this specific workshop
			if user.Role.Workshop != nil && user.Role.Workshop.ID == *workshopID {
				return nil
			}
			log.Debug("participant workshop access denied",
				"user_workshop_id", user.Role.Workshop,
				"requested_workshop_id", *workshopID,
				"user_role", user.Role.Role)
		}
		return obj.ErrForbidden("not authorized to read this workshop")

	case OpList:
		// Head or Staff can list workshops of their institution
		if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
			return nil
		}
		return obj.ErrForbidden("only admin, head, or staff can list workshops")

	case OpUpdate, OpDelete:
		// Head or Staff can update/delete any workshop in their institution
		if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
			return nil
		}
		return obj.ErrForbidden("only admin, head, or staff of the institution can modify workshops")

	default:
		return obj.ErrForbidden("unknown operation")
	}
}

// canAccessGame checks if user can perform a CRUD operation on a game
// - operation: the type of CRUD operation (create, read, update, delete, list)
// - game: the game object (nil for create/list operations)
// - shareToken: optional share token provided by user (for private share links)
func canAccessGame(ctx context.Context, userID uuid.UUID, operation CRUDOperation, game *obj.Game, shareToken *string) error {
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

		// 3. Valid share token grants access
		if shareToken != nil && game.PrivateShareHash != nil && *shareToken == *game.PrivateShareHash {
			return nil
		}

		// 4. Workshop members can access workshop games
		if game.WorkshopID != nil {
			user, err := GetUserByID(ctx, userID)
			if err == nil && user.Role != nil {
				// Participant with role for this specific workshop can read
				if user.Role.Role == obj.RoleParticipant && user.Role.Workshop != nil && user.Role.Workshop.ID == *game.WorkshopID {
					return nil
				}
				// Staff with role for this specific workshop can read
				if user.Role.Role == obj.RoleStaff && user.Role.Workshop != nil && user.Role.Workshop.ID == *game.WorkshopID {
					return nil
				}
				// Head of institution can read all workshop games in their institution
				if user.Role.Role == obj.RoleHead && user.Role.Institution != nil {
					return nil
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
		// If game belongs to a workshop, head of institution can update/delete
		if game.WorkshopID != nil {
			user, err := GetUserByID(ctx, userID)
			if err == nil && user.Role != nil && user.Role.Role == obj.RoleHead && user.Role.Institution != nil {
				return nil
			}
		}
		return obj.ErrForbidden("only the owner or institution head can modify this game")

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

		// If game belongs to a workshop, user must have read access to that workshop
		if game.WorkshopID.Valid {
			// Get the workshop to find its institution ID
			workshop, err := queries().GetWorkshopByID(ctx, game.WorkshopID.UUID)
			if err != nil {
				return obj.ErrNotFound("workshop not found")
			}
			if err := canAccessWorkshop(ctx, userID, OpRead, workshop.InstitutionID, &game.WorkshopID.UUID, uuid.Nil); err != nil {
				return obj.ErrForbidden("not authorized to play games in this workshop")
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
			// Load game to check if this key sponsors it
			game, err := queries().GetGameByID(ctx, *gameID)
			if err == nil {
				// Public game with sponsored key
				if game.Public && game.PublicSponsoredApiKeyID.Valid && game.PublicSponsoredApiKeyID.UUID == apiKeyID {
					return nil
				}
				// Private game with sponsored key (share link context)
				if game.PrivateSponsoredApiKeyID.Valid && game.PrivateSponsoredApiKeyID.UUID == apiKeyID {
					return nil
				}
			}
		}

		// Check workshop context for sponsored keys
		if workshopID != nil {
			// Check if key is shared with this workshop
			shares, err := queries().GetApiKeySharesByApiKeyID(ctx, apiKeyID)
			if err == nil {
				for _, share := range shares {
					if share.WorkshopID.Valid && share.WorkshopID.UUID == *workshopID && share.AllowPublicGameSponsoring {
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

// canAccessUser checks if user can perform a CRUD operation on a user account
// - operation: the type of CRUD operation (create, read, update, delete, list)
// - targetUserID: the user being accessed
func canAccessUser(ctx context.Context, userID uuid.UUID, operation CRUDOperation, targetUserID uuid.UUID) error {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	// Admin can do everything
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	switch operation {
	case OpCreate:
		// Any authenticated request can create users (auth validation at API level)
		return nil

	case OpRead:
		// Users can read their own profile
		if targetUserID == userID {
			return nil
		}
		// Heads can read users in their institution's workshops
		if user.Role != nil && user.Role.Role == obj.RoleHead && user.Role.Institution != nil {
			// Check if target user has a role in any workshop of this institution
			targetUser, err := GetUserByID(ctx, targetUserID)
			if err == nil && targetUser.Role != nil {
				// If target has workshop role, check if workshop belongs to head's institution
				if targetUser.Role.Workshop != nil {
					workshop, err := queries().GetWorkshopByID(ctx, targetUser.Role.Workshop.ID)
					if err == nil && workshop.InstitutionID == user.Role.Institution.ID {
						return nil
					}
				}
				// If target has institution role for same institution
				if targetUser.Role.Institution != nil && targetUser.Role.Institution.ID == user.Role.Institution.ID {
					return nil
				}
			}
		}
		return obj.ErrForbidden("not authorized to read this user")

	case OpList:
		// Admin can list all users
		// Heads can list users in their institution
		// Complex filtering logic implemented in the list function itself
		if user.Role != nil && user.Role.Role == obj.RoleHead && user.Role.Institution != nil {
			return nil
		}
		return obj.ErrForbidden("only admins or institution heads can list users")

	case OpUpdate:
		// Users can update their own profile
		if targetUserID == userID {
			return nil
		}
		return obj.ErrForbidden("not authorized to update this user")

	case OpDelete:
		// Only admin can delete users
		return obj.ErrForbidden("only admins can delete users")

	default:
		return obj.ErrForbidden("unknown operation")
	}
}

// canManageUserRole checks if user can manage (set/remove) roles
// Currently only admins can manage roles
func canManageUserRole(ctx context.Context, userID uuid.UUID) error {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}
	// Only admin can manage roles
	if user.Role == nil || user.Role.Role != obj.RoleAdmin {
		return obj.ErrForbidden("only admins can manage user roles")
	}

	return nil
}

// canAccessInvite checks if user can perform a CRUD operation on invites
// - operation: the type of CRUD operation
// - inviteID: pointer to the invite (nil for list operations)
// - dbInvite: pointer to the database invite record (for read operations)
// - userID can be zero UUID for anonymous users (only allowed for reading open invites)
func canAccessInvite(ctx context.Context, userID uuid.UUID, operation CRUDOperation, dbInvite *sqlc.UserRoleInvite) error {
	// Handle anonymous users (zero UUID)
	isAnonymous := userID == uuid.Nil

	// For anonymous users, only allow reading open invites
	if isAnonymous {
		if operation == OpRead && dbInvite != nil {
			// For open invites (no specific user), anyone can read
			isOpenInvite := !dbInvite.InvitedUserID.Valid && !dbInvite.InvitedEmail.Valid
			if isOpenInvite {
				return nil
			}
		}
		return obj.ErrUnauthorized("authentication required")
	}

	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	// Admin can do everything
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	switch operation {
	case OpList:
		// Regular users can list their own pending invites (filtered in query)
		return nil

	case OpRead:
		// Check if user can access this specific invite
		if dbInvite == nil {
			return obj.ErrValidation("invite required for read operation")
		}

		// User is the invited user (by ID or email)
		isInvitedUser := (dbInvite.InvitedUserID.Valid && dbInvite.InvitedUserID.UUID == userID) ||
			(dbInvite.InvitedEmail.Valid && user.Email != nil && *user.Email == dbInvite.InvitedEmail.String)

		// User is the creator
		isCreator := dbInvite.CreatedBy.Valid && dbInvite.CreatedBy.UUID == userID

		// For open invites (no specific user), anyone can read
		isOpenInvite := !dbInvite.InvitedUserID.Valid && !dbInvite.InvitedEmail.Valid

		if isInvitedUser || isCreator || isOpenInvite {
			return nil
		}

		return obj.ErrForbidden("not authorized to view this invite")

	default:
		return obj.ErrForbidden("unknown operation")
	}
}
