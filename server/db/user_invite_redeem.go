// package: db / database access and repository layer
// type:    data
// job:     accept, decline, revoke invites and manage workshop membership.
// limits:  does not create or list invites (-> user_invite.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"
	"fmt"
	"time"

	ung "github.com/dillonstreator/go-unique-name-generator"
	dictionaries "github.com/dillonstreator/go-unique-name-generator/dictionaries"
	"github.com/google/uuid"
)

// AddMemberToWorkshop directly adds an org individual to a workshop by setting their active_workshop_id.
// The caller must be a head or staff of the workshop's institution.
// The target user must be an individual in the same institution.
func AddMemberToWorkshop(
	ctx context.Context,
	callerID uuid.UUID,
	workshopID uuid.UUID,
	targetUserID uuid.UUID,
) error {
	// Get workshop to check institution
	workshop, err := queries().GetWorkshopByID(ctx, workshopID)
	if err != nil {
		return obj.ErrNotFound("workshop not found")
	}

	// Check permission: only head or staff of the institution can add members
	if err := canAccessWorkshop(ctx, callerID, OpCreate, workshop.InstitutionID, &workshopID, uuid.Nil); err != nil {
		return err
	}

	// Validate target user is an individual in the same institution
	targetUser, err := GetUserByID(ctx, targetUserID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}
	if targetUser.Role == nil {
		return obj.ErrValidation("user has no role")
	}
	if targetUser.Role.Role != obj.RoleIndividual {
		return obj.ErrValidation("only individual users can be added to workshops this way")
	}
	if targetUser.Role.Institution == nil || targetUser.Role.Institution.ID != workshop.InstitutionID {
		return obj.ErrForbidden("user does not belong to the same institution")
	}

	// Set active workshop for the target user
	return SetActiveWorkshop(ctx, targetUserID, workshopID)
}

// RemoveMemberFromWorkshop clears the active workshop for a non-permanent member (individual/head/staff visiting).
// Only head or staff of the institution that owns the workshop can perform this action.
func RemoveMemberFromWorkshop(
	ctx context.Context,
	callerID uuid.UUID,
	workshopID uuid.UUID,
	targetUserID uuid.UUID,
) error {
	workshop, err := queries().GetWorkshopByID(ctx, workshopID)
	if err != nil {
		return obj.ErrNotFound("workshop not found")
	}

	if err := canAccessWorkshop(ctx, callerID, OpCreate, workshop.InstitutionID, &workshopID, uuid.Nil); err != nil {
		return err
	}

	// Verify the target user actually has this workshop as their active workshop
	targetUser, err := GetUserByID(ctx, targetUserID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}
	if targetUser.Role == nil || targetUser.Role.Workshop == nil || targetUser.Role.Workshop.ID != workshopID {
		return obj.ErrValidation("user is not an active member of this workshop")
	}

	return ClearActiveWorkshop(ctx, targetUserID, workshopID)
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
	if err := txQueries.DeleteUserRoles(ctx, userID); err != nil {
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
// This permanently deletes the invite (hard delete).
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

	return queries().DeleteInvite(ctx, inviteID)
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

	// Check if user is already associated with this workshop or organization
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return uuid.Nil, obj.ErrNotFound("user not found")
	}

	if user.Role != nil {
		// Check if user is already a participant of this specific workshop
		if invite.WorkshopID.Valid && user.Role.Workshop != nil && user.Role.Workshop.ID == invite.WorkshopID.UUID {
			return uuid.Nil, obj.ErrConflict("you are already a member of this workshop")
		}

		// Check if user is head/staff of the organization that owns this workshop
		if user.Role.Institution != nil && user.Role.Institution.ID == invite.InstitutionID {
			if user.Role.Role == obj.RoleHead || user.Role.Role == obj.RoleStaff {
				return uuid.Nil, obj.ErrConflict("you are already a member of the organization that owns this workshop")
			}
		}
	}

	// Use a transaction to ensure atomicity (delete old role + create new role + increment uses)
	tx, err := sqlDb.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, obj.ErrServerError("failed to begin transaction")
	}
	defer tx.Rollback() // Rollback if not committed

	txQueries := queries().WithTx(tx)

	// Delete existing role (enforce single-role constraint)
	if err := txQueries.DeleteUserRoles(ctx, userID); err != nil {
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

// AcceptWorkshopInviteAnonymously accepts a workshop invite token without authentication.
// Creates an ad-hoc user account with a generated name and assigns them as a participant.
// Returns the created user and their authentication token.
//
// IMPORTANT CONSTRAINTS for anonymous participants:
// - They are bound to their workshop via the participant role and CANNOT change roles
// - They can only acquire a JWT (short-lived) if the workshop is still active
// - The auth token is their permanent credential stored in the auth0_id field
//
// language is an ISO 639-1 code resolved from the request
// (-> httpx.PreferredLanguage). Participants join through a link and never see
// a registration form, so their browser language is the only signal we have.
func AcceptWorkshopInviteAnonymously(ctx context.Context, inviteToken string, language string) (*obj.User, string, error) {
	// Get the invite by token
	invite, err := queries().GetInviteByToken(ctx, sql.NullString{String: inviteToken, Valid: true})
	if err != nil {
		return nil, "", obj.ErrNotFound("invite not found")
	}

	// Validate this is a workshop invite
	if !invite.WorkshopID.Valid {
		return nil, "", obj.ErrValidation("this is not a workshop invite")
	}

	// Validate invite status
	if invite.Status != string(obj.InviteStatusPending) {
		return nil, "", obj.ErrValidationf("invite is not pending (status: %s)", invite.Status)
	}

	// Check expiration
	if invite.ExpiresAt.Valid && invite.ExpiresAt.Time.Before(time.Now()) {
		return nil, "", obj.ErrValidation("invite has expired")
	}

	// Check max uses
	if invite.MaxUses.Valid && invite.UsesCount >= invite.MaxUses.Int32 {
		return nil, "", obj.ErrValidation("invite has reached maximum uses")
	}

	// Generate funny readable name for anonymous participant (e.g., "red-dragon-a1b2").
	// Append a short random suffix to guarantee uniqueness — the readable
	// color+animal pair has limited combinations and can collide under load.
	nameGenerator := ung.NewUniqueNameGenerator(
		ung.WithDictionaries(
			[][]string{
				dictionaries.Colors,
				dictionaries.Animals,
			},
		),
		ung.WithSeparator("-"),
	)
	anonymousName := nameGenerator.Generate() + "-" + functional.First(functional.GenerateSecureToken(2))

	// Generate participant token for this participant (prefixed to distinguish from JWT)
	participantToken := "participant-" + functional.First(functional.GenerateSecureToken(32))

	// Use a transaction to ensure atomicity (create user + create role + increment uses)
	tx, err := sqlDb.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", obj.ErrServerError("failed to begin transaction")
	}
	defer tx.Rollback()

	txQueries := queries().WithTx(tx)

	// Generate user ID
	userID := uuid.New()

	// Create anonymous user with participant token (no email, no auth0_id)
	userArg := db.CreateUserWithParticipantTokenParams{
		ID:               userID,
		Name:             anonymousName,
		Email:            sql.NullString{Valid: false},
		Auth0ID:          sql.NullString{Valid: false},
		ParticipantToken: sql.NullString{String: participantToken, Valid: true},
		Language:         language,
	}
	_, err = txQueries.CreateUserWithParticipantToken(ctx, userArg)
	if err != nil {
		log.Error("failed to create participant user", "name", anonymousName, "error", err)
		return nil, "", obj.ErrServerError("failed to create user: " + err.Error())
	}

	// Create participant role for the workshop
	roleArg := db.CreateUserRoleParams{
		UserID:        userID,
		Role:          sql.NullString{String: string(obj.RoleParticipant), Valid: true},
		InstitutionID: uuid.NullUUID{UUID: invite.InstitutionID, Valid: true},
		WorkshopID:    invite.WorkshopID,
	}
	_, err = txQueries.CreateUserRole(ctx, roleArg)
	if err != nil {
		return nil, "", obj.ErrServerError("failed to create participant role")
	}

	// Increment uses count
	if err := txQueries.IncrementInviteUses(ctx, invite.ID); err != nil {
		return nil, "", fmt.Errorf("failed to increment invite uses: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, "", obj.ErrServerError("failed to commit transaction")
	}

	// Get the created user
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, "", obj.ErrServerErrorf("failed to retrieve created user: %v", err)
	}

	// Return the user and their participant token (prefixed with "participant-")
	// This token will be detected by auth middleware and handled via DB lookup
	return user, participantToken, nil
}
