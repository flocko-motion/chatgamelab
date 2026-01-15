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

// CreateInstitutionInvite creates an invitation for a specific user (by user_id or email) to join an institution.
// Only head and staff roles are allowed (use CreateWorkshopInvite for participant invites).
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
) (db.UserRoleInvite, error) {
	// Validate role - only head and staff allowed
	if role != obj.RoleHead && role != obj.RoleStaff {
		return db.UserRoleInvite{}, obj.ErrValidationf("institution invites only allow head or staff roles, got: %s", role)
	}

	// Validate that at least one target is provided
	if invitedUserID == nil && invitedEmail == nil {
		return db.UserRoleInvite{}, obj.ErrValidation("either invitedUserID or invitedEmail must be provided")
	}

	// Check permission using centralized system
	// Creating invites requires update permission on the institution
	if err := canAccessInstitution(ctx, createdBy, OpUpdate, &institutionID); err != nil {
		return db.UserRoleInvite{}, err
	}

	// Generate secure token (32 bytes = ~43 chars, 256 bits entropy)
	inviteToken := functional.First(functional.GenerateSecureToken(32))

	arg := db.CreateTargetedInviteParams{
		CreatedBy:     uuid.NullUUID{UUID: createdBy, Valid: true},
		InstitutionID: institutionID,
		Role:          string(role),
		WorkshopID:    uuid.NullUUID{}, // Institution invites don't have workshop scope
		InvitedUserID: uuid.NullUUID{UUID: uuidPtrToUUID(invitedUserID), Valid: invitedUserID != nil},
		InvitedEmail:  sql.NullString{String: functional.Deref(invitedEmail, ""), Valid: invitedEmail != nil},
		InviteToken:   sql.NullString{String: inviteToken, Valid: true},
	}

	result, err := queries().CreateTargetedInvite(ctx, arg)
	if err != nil {
		return db.UserRoleInvite{}, err
	}

	return result, nil
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
) (db.UserRoleInvite, error) {
	// Check permission using centralized system
	// Creating invites requires update permission on the workshop
	if err := canAccessWorkshop(ctx, createdBy, OpUpdate, uuid.Nil, &workshopID, uuid.Nil); err != nil {
		return db.UserRoleInvite{}, err
	}

	// Get workshop to look up institution_id
	workshop, err := queries().GetWorkshopByID(ctx, workshopID)
	if err != nil {
		return db.UserRoleInvite{}, obj.ErrNotFound("workshop not found")
	}

	// Generate secure token (32 bytes = ~43 chars, 256 bits entropy)
	inviteToken := functional.First(functional.GenerateSecureToken(32))

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
		return db.UserRoleInvite{}, err
	}

	return result, nil
}

// UpdateInviteStatus updates the status of an invite.
// Only the creator, admin, or the invited user can update the status.
func UpdateInviteStatus(ctx context.Context, userID uuid.UUID, inviteID uuid.UUID, status obj.InviteStatus) error {
	// Get the invite to check permissions
	invite, err := queries().GetInviteByID(ctx, inviteID)
	if err != nil {
		return obj.ErrNotFound("invite not found")
	}

	// Check permission: creator, admin, or invited user
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}

	isAdmin := user.Role != nil && user.Role.Role == obj.RoleAdmin
	isCreator := invite.CreatedBy.Valid && invite.CreatedBy.UUID == userID
	isInvitedUser := (invite.InvitedUserID.Valid && invite.InvitedUserID.UUID == userID) ||
		(invite.InvitedEmail.Valid && user.Email != nil && *user.Email == invite.InvitedEmail.String)

	if !isAdmin && !isCreator && !isInvitedUser {
		return obj.ErrForbidden("not authorized to update this invite")
	}

	arg := db.UpdateInviteStatusParams{
		ID:     inviteID,
		Status: string(status),
	}

	return queries().UpdateInviteStatus(ctx, arg)
}

// AcceptTargetedInvite accepts a targeted invite (by invite ID or token) and creates a user role.
// Either inviteID or inviteToken must be provided (pass uuid.Nil for inviteID if using token).
// The userID must match either the invited_user_id or the user associated with invited_email.
// Returns the created role ID or an error if the invite is invalid or already processed.
func AcceptTargetedInvite(ctx context.Context, inviteID uuid.UUID, inviteToken string, userID uuid.UUID) (uuid.UUID, error) {
	// Get the invite (by ID or token)
	var invite db.UserRoleInvite
	var err error

	if inviteToken != "" {
		invite, err = queries().GetInviteByToken(ctx, sql.NullString{String: inviteToken, Valid: true})
		if err != nil {
			return uuid.Nil, obj.ErrNotFound("invite not found")
		}
	} else if inviteID != uuid.Nil {
		invite, err = queries().GetInviteByID(ctx, inviteID)
		if err != nil {
			return uuid.Nil, obj.ErrNotFound("invite not found")
		}
	} else {
		return uuid.Nil, obj.ErrValidation("either inviteID or inviteToken must be provided")
	}

	// Validate this is a targeted invite (has invited_user_id or invited_email, not for open invites)
	if !invite.InvitedUserID.Valid && !invite.InvitedEmail.Valid {
		return uuid.Nil, obj.ErrValidation("this is an open invite, use AcceptOpenInvite instead")
	}

	// Validate invite status
	if invite.Status != string(obj.InviteStatusPending) {
		return uuid.Nil, obj.ErrValidationf("invite is not pending (status: %s)", invite.Status)
	}

	// Check expiration
	if invite.ExpiresAt.Valid && invite.ExpiresAt.Time.Before(time.Now()) {
		_ = UpdateInviteStatus(ctx, userID, invite.ID, obj.InviteStatusExpired)
		return uuid.Nil, obj.ErrValidation("invite has expired")
	}

	// Validate the user is the intended recipient
	if invite.InvitedUserID.Valid {
		// Invite is by user ID - must match exactly
		if invite.InvitedUserID.UUID != userID {
			return uuid.Nil, obj.ErrForbidden("this invite is for a different user")
		}
	} else if invite.InvitedEmail.Valid {
		// Invite is by email - look up user and verify email matches
		user, err := GetUserByID(ctx, userID)
		if err != nil {
			return uuid.Nil, obj.ErrNotFound("user not found")
		}
		if user.Email == nil || *user.Email != invite.InvitedEmail.String {
			return uuid.Nil, obj.ErrForbidden("this invite is for a different email address")
		}
	}

	// Create the user role
	arg := db.CreateUserRoleParams{
		UserID:        userID,
		Role:          sql.NullString{String: invite.Role, Valid: true},
		InstitutionID: uuid.NullUUID{UUID: invite.InstitutionID, Valid: true},
		WorkshopID:    invite.WorkshopID,
	}

	roleID, err := queries().CreateUserRole(ctx, arg)
	if err != nil {
		return uuid.Nil, obj.ErrServerError("failed to create user role")
	}

	// Mark invite as accepted
	if err := queries().AcceptTargetedInvite(ctx, db.AcceptTargetedInviteParams{
		ID:         inviteID,
		AcceptedBy: uuid.NullUUID{UUID: userID, Valid: true},
	}); err != nil {
		return uuid.Nil, obj.ErrServerError("failed to mark invite as accepted")
	}

	// Return the role ID (handle NullUUID)
	if !roleID.Valid {
		return uuid.Nil, fmt.Errorf("failed to create user role: no ID returned")
	}
	return roleID.UUID, nil
}

// DeclineTargetedInvite declines a targeted invite.
// Either inviteID or inviteToken must be provided (pass uuid.Nil for inviteID if using token).
// The userID must match either the invited_user_id or the user associated with invited_email.
func DeclineTargetedInvite(ctx context.Context, inviteID uuid.UUID, inviteToken string, userID uuid.UUID) error {
	// Get the invite (by ID or token)
	var invite db.UserRoleInvite
	var err error

	if inviteToken != "" {
		invite, err = queries().GetInviteByToken(ctx, sql.NullString{String: inviteToken, Valid: true})
		if err != nil {
			return fmt.Errorf("invite not found: %w", err)
		}
	} else if inviteID != uuid.Nil {
		invite, err = queries().GetInviteByID(ctx, inviteID)
		if err != nil {
			return fmt.Errorf("invite not found: %w", err)
		}
	} else {
		return fmt.Errorf("either inviteID or inviteToken must be provided")
	}

	// Validate this is a targeted invite (has invited_user_id or invited_email, not for open invites)
	if !invite.InvitedUserID.Valid && !invite.InvitedEmail.Valid {
		return fmt.Errorf("this is an open invite, cannot decline")
	}

	// Validate invite status
	if invite.Status != string(obj.InviteStatusPending) {
		return fmt.Errorf("invite is not pending (status: %s)", invite.Status)
	}

	// Validate the user is the intended recipient
	if invite.InvitedUserID.Valid {
		// Invite is by user ID - must match exactly
		if invite.InvitedUserID.UUID != userID {
			return fmt.Errorf("this invite is for a different user")
		}
	} else if invite.InvitedEmail.Valid {
		// Invite is by email - look up user and verify email matches
		user, err := GetUserByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("user not found: %w", err)
		}
		if user.Email == nil || *user.Email != invite.InvitedEmail.String {
			return fmt.Errorf("this invite is for a different email address")
		}
	}

	// Mark invite as declined
	return UpdateInviteStatus(ctx, userID, inviteID, obj.InviteStatusDeclined)
}

// RevokeInvite revokes an invite (can only be done by the creator or an admin).
// This marks the invite as 'revoked' so it can no longer be accepted.
func RevokeInvite(ctx context.Context, inviteID uuid.UUID, revokedBy uuid.UUID) error {
	// Get the invite
	invite, err := queries().GetInviteByID(ctx, inviteID)
	if err != nil {
		return fmt.Errorf("invite not found: %w", err)
	}

	// Validate the invite can be revoked (only pending invites)
	if invite.Status != string(obj.InviteStatusPending) {
		return fmt.Errorf("only pending invites can be revoked (current status: %s)", invite.Status)
	}

	// Validate revokedBy has permission (must be creator or admin)
	revoker, err := GetUserByID(ctx, revokedBy)
	if err != nil {
		return fmt.Errorf("revoker not found: %w", err)
	}
	if revoker.Role == nil {
		return fmt.Errorf("revoker has no role assigned")
	}

	// Admin can revoke any invite (god-mode)
	isAdmin := revoker.Role.Role == obj.RoleAdmin
	// Creator can revoke their own invites
	isCreator := invite.CreatedBy.Valid && invite.CreatedBy.UUID == revokedBy

	if !isAdmin && !isCreator {
		return fmt.Errorf("only the invite creator or an admin can revoke invites")
	}

	// Mark invite as revoked
	return UpdateInviteStatus(ctx, revokedBy, inviteID, obj.InviteStatusRevoked)
}

// AcceptOpenInvite accepts an open invite using its token and creates a user role.
// Returns the created role ID or an error if the invite is invalid, expired, or exhausted.
func AcceptOpenInvite(ctx context.Context, inviteToken string, userID uuid.UUID) (uuid.UUID, error) {
	// Get the invite by token
	invite, err := queries().GetInviteByToken(ctx, sql.NullString{String: inviteToken, Valid: true})
	if err != nil {
		return uuid.Nil, fmt.Errorf("invite not found: %w", err)
	}

	// Validate invite status
	if invite.Status != string(obj.InviteStatusPending) {
		return uuid.Nil, obj.ErrValidationf("invite is not pending (status: %s)", invite.Status)
	}

	// Check expiration
	if invite.ExpiresAt.Valid && invite.ExpiresAt.Time.Before(time.Now()) {
		// Mark as expired
		_ = UpdateInviteStatus(ctx, userID, invite.ID, obj.InviteStatusExpired)
		return uuid.Nil, fmt.Errorf("invite has expired")
	}

	// Check max uses
	if invite.MaxUses.Valid && invite.UsesCount >= invite.MaxUses.Int32 {
		// Mark as expired
		_ = UpdateInviteStatus(ctx, userID, invite.ID, obj.InviteStatusExpired)
		return uuid.Nil, fmt.Errorf("invite has reached maximum uses")
	}

	// Create the user role
	arg := db.CreateUserRoleParams{
		UserID:        userID,
		Role:          sql.NullString{String: invite.Role, Valid: true},
		InstitutionID: uuid.NullUUID{UUID: invite.InstitutionID, Valid: true},
		WorkshopID:    invite.WorkshopID,
	}

	roleID, err := queries().CreateUserRole(ctx, arg)
	if err != nil {
		return uuid.Nil, obj.ErrServerError("failed to create user role")
	}

	// Increment uses count
	if err := queries().IncrementInviteUses(ctx, invite.ID); err != nil {
		return uuid.Nil, fmt.Errorf("failed to increment invite uses: %w", err)
	}

	// If max uses reached, mark as expired
	if invite.MaxUses.Valid && invite.UsesCount+1 >= invite.MaxUses.Int32 {
		_ = UpdateInviteStatus(ctx, userID, invite.ID, obj.InviteStatusExpired)
	}

	// Return the role ID (handle NullUUID)
	if !roleID.Valid {
		return uuid.Nil, fmt.Errorf("failed to create user role: no ID returned")
	}
	return roleID.UUID, nil
}
