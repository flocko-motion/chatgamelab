// package: db / database access and repository layer
// type:    data
// job:     convert, list, and create user role invites, generating unique invite codes.
// limits:  does not accept/redeem invites (-> user_invite_redeem.go) or define SQL (-> db/sqlc).
package db

import (
	db "cgl/db/sqlc"
	"cgl/functional"
	"cgl/obj"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// dbInviteToObj converts db.UserRoleInvite to obj.UserRoleInvite
func dbInviteToObj(dbInv db.UserRoleInvite) obj.UserRoleInvite {
	inv := obj.UserRoleInvite{
		ID:            dbInv.ID,
		InstitutionID: dbInv.InstitutionID,
		Role:          obj.Role(dbInv.Role),
		Status:        obj.InviteStatus(dbInv.Status),
		UsesCount:     dbInv.UsesCount,
	}

	// Meta
	inv.Meta.CreatedAt = &dbInv.CreatedAt
	inv.Meta.ModifiedAt = &dbInv.ModifiedAt
	if dbInv.CreatedBy.Valid {
		inv.Meta.CreatedBy = uuid.NullUUID{UUID: dbInv.CreatedBy.UUID, Valid: true}
	}
	if dbInv.ModifiedBy.Valid {
		inv.Meta.ModifiedBy = uuid.NullUUID{UUID: dbInv.ModifiedBy.UUID, Valid: true}
	}

	// Optional fields
	if dbInv.WorkshopID.Valid {
		inv.WorkshopID = &dbInv.WorkshopID.UUID
	}
	if dbInv.InvitedUserID.Valid {
		inv.InvitedUserID = &dbInv.InvitedUserID.UUID
	}
	inv.InvitedEmail = sqlNullStringToMaybeString(dbInv.InvitedEmail)
	inv.InviteToken = sqlNullStringToMaybeString(dbInv.InviteToken)
	if dbInv.MaxUses.Valid {
		inv.MaxUses = &dbInv.MaxUses.Int32
	}
	if dbInv.ExpiresAt.Valid {
		inv.ExpiresAt = &dbInv.ExpiresAt.Time
	}
	if dbInv.AcceptedAt.Valid {
		inv.AcceptedAt = &dbInv.AcceptedAt.Time
	}
	if dbInv.AcceptedBy.Valid {
		inv.AcceptedBy = &dbInv.AcceptedBy.UUID
	}

	return inv
}

// GetInviteByToken retrieves a specific invite by token
// - Anyone can look up an invite by token (for open invites)
// - For targeted invites with tokens, user must be the invited user
func GetInviteByToken(ctx context.Context, userID uuid.UUID, token string) (obj.UserRoleInvite, error) {
	// Get the invite by token
	dbInvite, err := queries().GetInviteByToken(ctx, sql.NullString{String: token, Valid: true})
	if err != nil {
		return obj.UserRoleInvite{}, obj.ErrNotFound("invite not found")
	}

	// Check permissions using centralized permission system
	if err := canAccessInvite(ctx, userID, OpRead, &dbInvite); err != nil {
		return obj.UserRoleInvite{}, err
	}

	return dbInviteToObj(dbInvite), nil
}

// GetInviteByID retrieves a specific invite by ID
// - Admins can see any invite
// - Regular users can only see invites targeted to them
func GetInviteByID(ctx context.Context, userID uuid.UUID, inviteID uuid.UUID) (obj.UserRoleInvite, error) {
	// Get the invite
	dbInvite, err := queries().GetInviteByID(ctx, inviteID)
	if err != nil {
		return obj.UserRoleInvite{}, obj.ErrNotFound("invite not found")
	}

	// Get user to check permissions
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.UserRoleInvite{}, obj.ErrNotFound("user not found")
	}

	// Check if user can access this invite
	isAdmin := user.Role != nil && user.Role.Role == obj.RoleAdmin
	isInvitedUser := (dbInvite.InvitedUserID.Valid && dbInvite.InvitedUserID.UUID == userID) ||
		(dbInvite.InvitedEmail.Valid && user.Email != nil && *user.Email == dbInvite.InvitedEmail.String)
	isCreator := dbInvite.CreatedBy.Valid && dbInvite.CreatedBy.UUID == userID

	if !isAdmin && !isInvitedUser && !isCreator {
		return obj.UserRoleInvite{}, obj.ErrForbidden("not authorized to view this invite")
	}

	return dbInviteToObj(dbInvite), nil
}

// GetInvites returns pending invites targeted to the current user.
// This is used for the notification bell - shows only invites the user needs to act on.
// For admin/organization management views, use GetInvitesByInstitutionID instead.
func GetInvites(ctx context.Context, userID uuid.UUID) ([]obj.UserRoleInvite, error) {
	// Check permissions using centralized permission system
	if err := canAccessInvite(ctx, userID, OpList, nil); err != nil {
		return nil, err
	}

	// All users (including heads/staff) only see their own pending invites here
	// This endpoint is for the notification bell, not for admin management
	dbInvites, err := queries().GetInvitesByUser(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get invites: %w", err)
	}

	// Convert to obj.UserRoleInvite
	invites := make([]obj.UserRoleInvite, len(dbInvites))
	for i, dbInv := range dbInvites {
		invites[i] = dbInviteToObj(dbInv)
	}

	return invites, nil
}

// GetAllInvites returns all invites (admin only)
func GetAllInvites(ctx context.Context, userID uuid.UUID) ([]obj.UserRoleInvite, error) {
	// Get user to check permissions
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.ErrNotFound("user not found")
	}

	// Only admins can see all invites
	if user.Role == nil || user.Role.Role != obj.RoleAdmin {
		return nil, obj.ErrForbidden("only admins can view all invites")
	}

	dbInvites, err := queries().GetInvites(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get invites: %w", err)
	}

	// Convert to obj.UserRoleInvite
	invites := make([]obj.UserRoleInvite, len(dbInvites))
	for i, dbInv := range dbInvites {
		invites[i] = dbInviteToObj(dbInv)
	}

	return invites, nil
}

// GetInvitesByInstitutionID returns all invites for an institution.
// Used for admin/organization management views.
// Permission check: admin can see all, heads/staff can see their institution's invites.
func GetInvitesByInstitutionID(ctx context.Context, userID uuid.UUID, institutionID uuid.UUID) ([]obj.UserRoleInvite, error) {
	// Get user to check permissions
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.ErrNotFound("user not found")
	}

	// Check permission
	isAdmin := user.Role != nil && user.Role.Role == obj.RoleAdmin
	isMemberOfInstitution := user.Role != nil && user.Role.Institution != nil && user.Role.Institution.ID == institutionID
	isHeadOrStaff := user.Role != nil && (user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff)

	if !isAdmin && !(isMemberOfInstitution && isHeadOrStaff) {
		return nil, obj.ErrForbidden("not authorized to view institution invites")
	}

	dbInvites, err := queries().GetInvitesByInstitution(ctx, institutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invites: %w", err)
	}

	// Convert to obj.UserRoleInvite, filtering based on role
	// Staff should not see head invites
	isStaff := user.Role != nil && user.Role.Role == obj.RoleStaff
	var invites []obj.UserRoleInvite
	for _, dbInv := range dbInvites {
		invite := dbInviteToObj(dbInv)
		// Staff can only see non-head invites
		if isStaff && invite.Role == obj.RoleHead {
			continue
		}
		invites = append(invites, invite)
	}

	return invites, nil
}

// CreateInstitutionInvite creates an invitation for a specific user (by user_id or email) to join an institution.
// Role can be head, staff, or empty (for users without a role).
// Either invitedUserID or invitedEmail must be provided.
// The creator must be a head of the institution (only heads can invite users to become staff/heads, admins can invite users to become admin/staff/heads).
// Returns the complete invite record including the ID.
func CreateInstitutionInvite(
	ctx context.Context,
	createdBy uuid.UUID,
	institutionID uuid.UUID,
	role obj.Role,
	invitedUserID *uuid.UUID,
	invitedEmail *string,
) (obj.UserRoleInvite, error) {
	// Validate role - only head, staff, or individual allowed
	if role != obj.RoleHead && role != obj.RoleStaff && role != obj.RoleIndividual {
		return obj.UserRoleInvite{}, obj.ErrValidationf("institution invites only allow head, staff, or individual roles, got: %s", role)
	}

	// Validate that at least one target is provided
	if invitedUserID == nil && invitedEmail == nil {
		return obj.UserRoleInvite{}, obj.ErrValidation("either invitedUserID or invitedEmail must be provided")
	}

	// If inviting by email, validate that a user with that email exists
	if invitedEmail != nil {
		user, err := queries().GetUserByEmail(ctx, sql.NullString{String: *invitedEmail, Valid: true})
		if err != nil {
			return obj.UserRoleInvite{}, obj.ErrNotFound("no user found with email: " + *invitedEmail)
		}
		// Set invitedUserID to the found user's ID for consistency
		invitedUserID = &user.ID
	}

	// Check permission using centralized system
	// Creating invites requires update permission on the institution
	if err := canAccessInstitution(ctx, createdBy, OpUpdate, &institutionID); err != nil {
		return obj.UserRoleInvite{}, err
	}

	// Only heads (and admins) can invite someone as head
	if role == obj.RoleHead {
		creator, err := GetUserByID(ctx, createdBy)
		if err != nil {
			return obj.UserRoleInvite{}, obj.ErrServerError("failed to get creator")
		}
		if creator.Role == nil || (creator.Role.Role != obj.RoleAdmin && creator.Role.Role != obj.RoleHead) {
			return obj.UserRoleInvite{}, obj.ErrForbidden("only heads or admins can invite users as head")
		}
	}

	// Check for existing pending invite for the same target
	existingInvite, err := queries().GetPendingInviteByTarget(ctx, db.GetPendingInviteByTargetParams{
		InstitutionID: institutionID,
		InvitedUserID: uuid.NullUUID{UUID: uuidPtrToUUID(invitedUserID), Valid: invitedUserID != nil},
		InvitedEmail:  sql.NullString{String: functional.Deref(invitedEmail, ""), Valid: invitedEmail != nil},
	})
	if err == nil && existingInvite.ID != uuid.Nil {
		// Return error - duplicate invite exists
		return obj.UserRoleInvite{}, obj.ErrConflict("A pending invite already exists for this user")
	}

	// Targeted invites don't use invite_token (constraint: either targeted OR open, not both)
	arg := db.CreateTargetedInviteParams{
		CreatedBy:     uuid.NullUUID{UUID: createdBy, Valid: true},
		InstitutionID: institutionID,
		Role:          string(role),
		WorkshopID:    uuid.NullUUID{}, // Institution invites don't have workshop scope
		InvitedUserID: uuid.NullUUID{UUID: uuidPtrToUUID(invitedUserID), Valid: invitedUserID != nil},
		InvitedEmail:  sql.NullString{String: functional.Deref(invitedEmail, ""), Valid: invitedEmail != nil},
		InviteToken:   sql.NullString{}, // NULL for targeted invites
	}

	result, err := queries().CreateTargetedInvite(ctx, arg)
	if err != nil {
		return obj.UserRoleInvite{}, err
	}

	return dbInviteToObj(result), nil
}

// CreateWorkshopInvite creates an invitation for unspecified users to join a workshop as participants.
// The institution is automatically looked up from the workshop.
// The creator must be a head or staff member of the institution that owns the workshop.
// A cryptographically secure token is automatically generated (32 bytes, ~43 chars, 256 bits entropy).
// Returns the complete invite record including the generated token.
func CreateWorkshopInvite(
	ctx context.Context,
	createdBy uuid.UUID,
	workshopID uuid.UUID,
	maxUses *int32,
	expiresAt *time.Time,
) (obj.UserRoleInvite, error) {
	// Get workshop first to look up institution_id for permission check
	workshop, err := queries().GetWorkshopByID(ctx, workshopID)
	if err != nil {
		return obj.UserRoleInvite{}, obj.ErrNotFound("workshop not found")
	}

	// Check permission: only head or staff of the institution can create workshop invites
	// (participants have OpRead access to their workshop but must not create invites)
	if err := canAccessWorkshop(ctx, createdBy, OpCreate, workshop.InstitutionID, &workshopID, uuid.Nil); err != nil {
		return obj.UserRoleInvite{}, err
	}

	// Check if there's already a pending invite for this workshop - return it instead of creating a new one
	existingInvites, err := queries().GetInvitesByWorkshop(ctx, uuid.NullUUID{UUID: workshopID, Valid: true})
	if err == nil {
		for _, inv := range existingInvites {
			if inv.Status == string(obj.InviteStatusPending) && inv.InviteToken.Valid {
				return dbInviteToObj(inv), nil
			}
		}
	}

	// Generate secure token (32 bytes = ~43 chars, 256 bits entropy)
	inviteToken := "ws-" + functional.First(functional.GenerateSecureToken(32))

	arg := db.CreateOpenInviteParams{
		CreatedBy:     uuid.NullUUID{UUID: createdBy, Valid: true},
		InstitutionID: workshop.InstitutionID,
		Role:          string(obj.RoleParticipant),
		WorkshopID:    uuid.NullUUID{UUID: workshopID, Valid: true},
		InviteToken:   sql.NullString{String: inviteToken, Valid: true},
		MaxUses:       sql.NullInt32{Int32: functional.Deref(maxUses, 0), Valid: maxUses != nil},
		ExpiresAt:     sql.NullTime{Time: functional.Deref(expiresAt, time.Time{}), Valid: expiresAt != nil},
	}

	result, err := queries().CreateOpenInvite(ctx, arg)
	if err != nil {
		return obj.UserRoleInvite{}, err
	}

	return dbInviteToObj(result), nil
}

// CreateWorkshopEmailInvite creates a targeted invite for a specific user (by email) to join a workshop.
// When accepted, head/staff/individual users enter workshop mode (SetActiveWorkshop);
// users without a role become participants.
func CreateWorkshopEmailInvite(
	ctx context.Context,
	createdBy uuid.UUID,
	workshopID uuid.UUID,
	invitedEmail string,
) (obj.UserRoleInvite, error) {
	if invitedEmail == "" {
		return obj.UserRoleInvite{}, obj.ErrValidation("email is required")
	}

	// Get workshop to look up institution_id
	workshop, err := queries().GetWorkshopByID(ctx, workshopID)
	if err != nil {
		return obj.UserRoleInvite{}, obj.ErrNotFound("workshop not found")
	}

	// Check permission: only head or staff of the institution can create workshop invites
	if err := canAccessWorkshop(ctx, createdBy, OpCreate, workshop.InstitutionID, &workshopID, uuid.Nil); err != nil {
		return obj.UserRoleInvite{}, err
	}

	// Resolve email to user — user must exist
	user, err := queries().GetUserByEmail(ctx, sql.NullString{String: invitedEmail, Valid: true})
	if err != nil {
		return obj.UserRoleInvite{}, obj.ErrNotFound("no user found with email: " + invitedEmail)
	}

	// Head/staff of the same org cannot be invited — they already have access to all workshops
	fullUser, err := GetUserByID(ctx, user.ID)
	if err == nil && fullUser != nil && fullUser.Role != nil &&
		(fullUser.Role.Role == obj.RoleHead || fullUser.Role.Role == obj.RoleStaff) &&
		fullUser.Role.Institution != nil && fullUser.Role.Institution.ID == workshop.InstitutionID {
		return obj.UserRoleInvite{}, obj.ErrValidation("This user is already head or staff of the organization and can access all workshops")
	}

	// Check for existing pending invite for same user + institution
	existingInvite, err := queries().GetPendingInviteByTarget(ctx, db.GetPendingInviteByTargetParams{
		InstitutionID: workshop.InstitutionID,
		InvitedUserID: uuid.NullUUID{UUID: user.ID, Valid: true},
		InvitedEmail:  sql.NullString{String: invitedEmail, Valid: true},
	})
	if err == nil && existingInvite.ID != uuid.Nil {
		// Check if the existing invite is for the same workshop
		if existingInvite.WorkshopID.Valid && existingInvite.WorkshopID.UUID == workshopID {
			return obj.UserRoleInvite{}, obj.ErrConflict("A pending invite already exists for this user and workshop")
		}
	}

	// Create targeted invite with workshop scope
	arg := db.CreateTargetedInviteParams{
		CreatedBy:     uuid.NullUUID{UUID: createdBy, Valid: true},
		InstitutionID: workshop.InstitutionID,
		Role:          string(obj.RoleParticipant),
		WorkshopID:    uuid.NullUUID{UUID: workshopID, Valid: true},
		InvitedUserID: uuid.NullUUID{UUID: user.ID, Valid: true},
		InvitedEmail:  sql.NullString{String: invitedEmail, Valid: true},
		InviteToken:   sql.NullString{}, // NULL for targeted invites
	}

	result, err := queries().CreateTargetedInvite(ctx, arg)
	if err != nil {
		return obj.UserRoleInvite{}, err
	}

	return dbInviteToObj(result), nil
}
