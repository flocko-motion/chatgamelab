// package: db / database access and repository layer
// type:    logic
// job:     evaluate whether a user may perform CRUD on institutions, workshops, users, and invites by role.
// limits:  does not authorize games/keys (-> permissions_game.go) or define SQL (-> db/queries).
package db

import (
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
		// Head or staff can update their own institution
		if institutionID == nil {
			return obj.ErrValidation("institutionID required for update operation")
		}
		if user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == *institutionID {
			if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
				return nil
			}
		}
		return obj.ErrForbidden("only admin, head, or staff of institution can update this institution")

	case OpDelete:
		// Only admin can delete institutions
		return obj.ErrForbidden("only admins can delete institutions")

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

// canAccessWorkshopParticipants checks if user can view the members of a workshop
// Only its Guests, the workshop owner, and institution heads/staff can view the member list
func canAccessWorkshopParticipants(ctx context.Context, userID uuid.UUID, workshopID uuid.UUID, workshopCreatedBy uuid.UUID, institutionID uuid.UUID) error {
	// Workshop owner can always see the members
	if workshopCreatedBy == userID {
		return nil
	}

	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	// Admin can view all members
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	// A Guest in this workshop can see the other members
	if user.Role != nil && user.Role.Workshop != nil && user.Role.Workshop.ID == workshopID && user.Role.Role == obj.RoleGuest {
		return nil
	}

	// Head or Staff of the institution that owns this workshop can see the members
	if user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == institutionID && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) {
		return nil
	}

	return obj.ErrForbidden("only guests, workshop owner, or institution staff/head can view the member list")
}

// CanDeleteUser checks if user can delete another user
// - Admins can delete any user
// - Staff/heads can only delete Guests (those with a guest_token) in their institution
func CanDeleteUser(ctx context.Context, currentUserID uuid.UUID, targetUserID uuid.UUID) error {
	currentUser, err := GetUserByID(ctx, currentUserID)
	if err != nil {
		return obj.ErrNotFound("current user not found")
	}

	// Admin can delete any user
	if currentUser.Role != nil && currentUser.Role.Role == obj.RoleAdmin {
		return nil
	}

	// Staff/heads can only delete Guests in their institution
	if currentUser.Role != nil && (currentUser.Role.Role == obj.RoleStaff || currentUser.Role.Role == obj.RoleHead) {
		// Get target user
		targetUser, err := GetUserByID(ctx, targetUserID)
		if err != nil {
			return obj.ErrNotFound("target user not found")
		}

		// Must be a Guest
		if targetUser.Role == nil || targetUser.Role.Role != obj.RoleGuest {
			return obj.ErrForbidden("can only delete guests")
		}

		// Must be in the same institution
		if targetUser.Role.Institution == nil || currentUser.Role.Institution == nil ||
			targetUser.Role.Institution.ID != currentUser.Role.Institution.ID {
			return obj.ErrForbidden("can only delete guests in your institution")
		}

		// Must have no Auth0 account (no auth0_id)
		// Users with Auth0 accounts cannot be deleted by staff/heads, even if their role is Guest
		rawUser, err := GetUserByIDRaw(ctx, targetUserID)
		if err != nil {
			return obj.ErrNotFound("target user not found")
		}

		// Check that the user has no Auth0 account
		if rawUser.Auth0ID.Valid && rawUser.Auth0ID.String != "" {
			return obj.ErrForbidden("can only delete guests, not users with an Auth0 account")
		}

		return nil
	}

	return obj.ErrForbidden("insufficient permissions to delete users")
}

// canAccessWorkshop checks if user can perform a CRUD operation on a workshop
// - operation: the type of CRUD operation (create, read, update, delete, list)
// - institutionID: the institution the workshop belongs to
// - workshopID: pointer to workshop ID (needed for guest read checks)
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

	// Individuals in workshop mode can read their active workshop (they have no institution)
	if user.Role != nil && user.Role.Role == obj.RoleIndividual && operation == OpRead && workshopID != nil {
		if user.Role.Workshop != nil && user.Role.Workshop.ID == *workshopID {
			return nil
		}
	}

	// Members (head/staff/guest) can access their institution's workshops
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
		// Guests can only read the workshop they have a role for
		if user.Role.Role == obj.RoleGuest && workshopID != nil {
			// Check if user's role is scoped to this specific workshop
			if user.Role.Workshop != nil && user.Role.Workshop.ID == *workshopID {
				return nil
			}
			log.Debug("guest workshop access denied",
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
		// Head/staff can read users in their institution
		if user.Role != nil && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) && user.Role.Institution != nil {
			targetUser, err := GetUserByID(ctx, targetUserID)
			if err == nil && targetUser.Role != nil {
				// If target has workshop role, check if workshop belongs to the institution
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
		// Head/staff can list users in their institution (filtered in handler)
		if user.Role != nil && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff) && user.Role.Institution != nil {
			return nil
		}
		return obj.ErrForbidden("only admins or institution heads/staff can list users")

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
