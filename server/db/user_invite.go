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

// GetInvites returns invites scoped by user permissions.
// - Admins see all invites
// - Regular users see only their own pending invites (targeted to them by user_id or email)
func GetInvites(ctx context.Context, userID uuid.UUID) ([]obj.UserRoleInvite, error) {
	// Check permissions using centralized permission system
	if err := canAccessInvite(ctx, userID, OpList, nil); err != nil {
		return nil, err
	}

	// Get user to determine scope
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.ErrNotFound("user not found")
	}

	var dbInvites []db.UserRoleInvite

	// Admin can see all invites
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		dbInvites, err = queries().GetInvites(ctx)
	} else {
		// Regular users only see their own pending invites
		dbInvites, err = queries().GetInvitesByUser(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	}

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
) (obj.UserRoleInvite, error) {
	// Validate role - only head and staff allowed
	if role != obj.RoleHead && role != obj.RoleStaff {
		return obj.UserRoleInvite{}, obj.ErrValidationf("institution invites only allow head or staff roles, got: %s", role)
	}

	// Validate that at least one target is provided
	if invitedUserID == nil && invitedEmail == nil {
		return obj.UserRoleInvite{}, obj.ErrValidation("either invitedUserID or invitedEmail must be provided")
	}

	// Check permission using centralized system
	// Creating invites requires update permission on the institution
	if err := canAccessInstitution(ctx, createdBy, OpUpdate, &institutionID); err != nil {
		return obj.UserRoleInvite{}, err
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

	// Check permission using centralized system
	// Creating invites requires update permission on the workshop
	var createdByID uuid.UUID
	if workshop.CreatedBy.Valid {
		createdByID = workshop.CreatedBy.UUID
	}
	if err := canAccessWorkshop(ctx, createdBy, OpUpdate, workshop.InstitutionID, &workshopID, createdByID); err != nil {
		return obj.UserRoleInvite{}, err
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

// updateInviteStatusUnchecked updates the status of an invite without permission checks.
// This is an internal function used by RevokeInvite and ReactivateInvite after they've done their own permission checks.
func updateInviteStatusUnchecked(ctx context.Context, inviteID uuid.UUID, status obj.InviteStatus) error {
	arg := db.UpdateInviteStatusParams{
		ID:     inviteID,
		Status: string(status),
	}
	return queries().UpdateInviteStatus(ctx, arg)
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

	return updateInviteStatusUnchecked(ctx, inviteID, status)
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

	// Use a transaction to ensure atomicity (delete old role + create new role + mark invite accepted)
	tx, err := sqlDb.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, obj.ErrServerError("failed to begin transaction")
	}
	defer tx.Rollback() // Rollback if not committed

	txQueries := queries().WithTx(tx)

	// Delete existing role (enforce single-role constraint)
	if err := txQueries.DeleteUserRole(ctx, userID); err != nil {
		// Log but don't fail if no existing role
		// This is expected for users accepting their first role
	}

	// Create the user role
	arg := db.CreateUserRoleParams{
		UserID:        userID,
		Role:          sql.NullString{String: invite.Role, Valid: true},
		InstitutionID: uuid.NullUUID{UUID: invite.InstitutionID, Valid: true},
		WorkshopID:    invite.WorkshopID,
	}

	roleID, err := txQueries.CreateUserRole(ctx, arg)
	if err != nil {
		return uuid.Nil, obj.ErrServerError("failed to create user role")
	}

	// Mark invite as accepted
	if err := txQueries.AcceptTargetedInvite(ctx, db.AcceptTargetedInviteParams{
		ID:         inviteID,
		AcceptedBy: uuid.NullUUID{UUID: userID, Valid: true},
	}); err != nil {
		return uuid.Nil, obj.ErrServerError("failed to mark invite as accepted")
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return uuid.Nil, obj.ErrServerError("failed to commit transaction")
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

// canManageInvite checks if a user can manage (revoke/reactivate) an invite.
// Returns true if the user is: admin, creator, or staff/head of the institution (for workshop invites).
func canManageInvite(ctx context.Context, invite db.UserRoleInvite, userID uuid.UUID) error {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Admin can manage any invite (god-mode)
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		return nil
	}

	// Creator can manage their own invites
	if invite.CreatedBy.Valid && invite.CreatedBy.UUID == userID {
		return nil
	}

	// For workshop invites, staff/heads of the institution can manage
	if invite.WorkshopID.Valid {
		if err := canAccessWorkshopInvites(ctx, userID, invite.InstitutionID); err == nil {
			return nil
		}
	}

	return fmt.Errorf("only the invite creator, admin, or institution staff can manage invites")
}

// RevokeInvite revokes an invite (can only be done by the creator, admin, or institution staff for workshop invites).
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

	// Check permission
	if err := canManageInvite(ctx, invite, revokedBy); err != nil {
		return err
	}

	return updateInviteStatusUnchecked(ctx, inviteID, obj.InviteStatusRevoked)
}

// ReactivateInvite re-activates a revoked invite (can only be done by the creator, admin, or institution staff for workshop invites).
// This marks the invite as 'pending' so it can be accepted again.
func ReactivateInvite(ctx context.Context, inviteID uuid.UUID, reactivatedBy uuid.UUID) error {
	// Get the invite
	invite, err := queries().GetInviteByID(ctx, inviteID)
	if err != nil {
		return fmt.Errorf("invite not found: %w", err)
	}

	// Validate the invite can be reactivated (only revoked invites)
	if invite.Status != string(obj.InviteStatusRevoked) {
		return fmt.Errorf("only revoked invites can be reactivated (current status: %s)", invite.Status)
	}

	// Check permission
	if err := canManageInvite(ctx, invite, reactivatedBy); err != nil {
		return err
	}

	return updateInviteStatusUnchecked(ctx, inviteID, obj.InviteStatusPending)
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

	// Use a transaction to ensure atomicity (delete old role + create new role + increment uses)
	tx, err := sqlDb.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, obj.ErrServerError("failed to begin transaction")
	}
	defer tx.Rollback() // Rollback if not committed

	txQueries := queries().WithTx(tx)

	// Delete existing role (enforce single-role constraint)
	if err := txQueries.DeleteUserRole(ctx, userID); err != nil {
		// Log but don't fail if no existing role
		// This is expected for users accepting their first role
	}

	// Create the user role
	arg := db.CreateUserRoleParams{
		UserID:        userID,
		Role:          sql.NullString{String: invite.Role, Valid: true},
		InstitutionID: uuid.NullUUID{UUID: invite.InstitutionID, Valid: true},
		WorkshopID:    invite.WorkshopID,
	}

	roleID, err := txQueries.CreateUserRole(ctx, arg)
	if err != nil {
		return uuid.Nil, obj.ErrServerError("failed to create user role")
	}

	// Increment uses count
	if err := txQueries.IncrementInviteUses(ctx, invite.ID); err != nil {
		return uuid.Nil, fmt.Errorf("failed to increment invite uses: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return uuid.Nil, obj.ErrServerError("failed to commit transaction")
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
