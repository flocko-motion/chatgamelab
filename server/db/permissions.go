package db

import (
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
		// Only admin can list all institutions
		return obj.ErrForbidden("only admins can list institutions")

	case OpRead:
		// Admin can read any institution
		// Members (head/staff/participant) can read their institution
		if institutionID == nil {
			return obj.ErrValidation("institutionID required for read operation")
		}
		if user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == *institutionID {
			return nil
		}
		// Participants with workshop role can read the institution that owns the workshop
		if user.Role != nil && user.Role.Role == obj.RoleParticipant && user.Role.Workshop != nil {
			// Need to load workshop to check its institution
			workshop, err := queries().GetWorkshopByID(ctx, user.Role.Workshop.ID)
			if err == nil && workshop.InstitutionID == *institutionID {
				return nil
			}
		}
		return obj.ErrForbidden("not authorized to read this institution")

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
		}
		return obj.ErrForbidden("not authorized to read this workshop")

	case OpList:
		// Head or Staff can list workshops of their institution
		if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
			return nil
		}
		return obj.ErrForbidden("only admin, head, or staff can list workshops")

	case OpUpdate, OpDelete:
		// Head can update/delete any workshop in their institution
		if user.Role.Role == obj.RoleHead {
			return nil
		}
		// Staff can only update/delete workshops they created
		if user.Role.Role == obj.RoleStaff && createdBy == userID {
			return nil
		}
		return obj.ErrForbidden("only admin, head of institution, or staff who created the workshop can modify it")

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

		// If game belongs to a workshop, user must have read access to that workshop
		if game.WorkshopID.Valid {
			if err := canAccessWorkshop(ctx, userID, OpRead, uuid.Nil, &game.WorkshopID.UUID, uuid.Nil); err != nil {
				return obj.ErrForbidden("not authorized to play games in this workshop")
			}
		}

		// If explicit workshopID is provided, validate access to it as well
		if workshopID != nil {
			if err := canAccessWorkshop(ctx, userID, OpRead, uuid.Nil, workshopID, uuid.Nil); err != nil {
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
func canAccessApiKey(ctx context.Context, userID uuid.UUID, operation CRUDOperation, apiKeyID uuid.UUID, keyOwnerID uuid.UUID) error {
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
		if err == nil && user.Role != nil {
			// Check for direct user share
			shares, err := queries().GetApiKeySharesByApiKeyID(ctx, apiKeyID)
			if err == nil {
				for _, share := range shares {
					// Direct user share
					if share.UserID.Valid && share.UserID.UUID == userID {
						return nil
					}
					// Workshop share
					if share.WorkshopID.Valid && user.Role.Workshop != nil && share.WorkshopID.UUID == user.Role.Workshop.ID {
						return nil
					}
					// Institution share
					if share.InstitutionID.Valid && user.Role.Institution != nil && share.InstitutionID.UUID == user.Role.Institution.ID {
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
