// package: db / database access and repository layer
// type:    data
// job:     CRUD and listing of institutions with permission checks.
// limits:  does not define SQL queries (-> db/queries) or expose HTTP endpoints (-> api).
package db

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"cgl/legal"
	"cgl/obj"

	"github.com/google/uuid"
)

// InstitutionDetails are an institution's operator (Träger) and address; an
// empty field is none.
type InstitutionDetails struct {
	Operator, Street, PostalCode, City string
}

// CreateInstitution creates a new institution with its details (admin only)
func CreateInstitution(ctx context.Context, createdBy uuid.UUID, name string, details InstitutionDetails) (*obj.Institution, error) {
	// Check permission
	if err := canAccessInstitution(ctx, createdBy, OpCreate, nil); err != nil {
		return nil, err
	}

	now := time.Now()

	arg := CreateInstitutionParams{
		ID:         uuid.New(),
		CreatedBy:  uuid.NullUUID{UUID: createdBy, Valid: true},
		CreatedAt:  now,
		ModifiedBy: uuid.NullUUID{UUID: createdBy, Valid: true},
		ModifiedAt: now,
		Name:       name,
	}

	result, err := queries().CreateInstitution(ctx, arg)
	if err != nil {
		return nil, obj.ErrServerError("failed to create institution")
	}
	optional := func(v string) sql.NullString {
		v = strings.TrimSpace(v)
		return sql.NullString{String: v, Valid: v != ""}
	}
	if err := queries().UpdateInstitutionDetails(ctx, UpdateInstitutionDetailsParams{
		ID:         result.ID,
		Operator:   optional(details.Operator),
		Street:     optional(details.Street),
		PostalCode: optional(details.PostalCode),
		City:       optional(details.City),
	}); err != nil {
		return nil, obj.ErrServerError("institution created, but its details could not be saved")
	}
	result.Operator, result.Street = optional(details.Operator), optional(details.Street)
	result.PostalCode, result.City = optional(details.PostalCode), optional(details.City)

	inst := &obj.Institution{
		ID:   result.ID,
		Name: result.Name,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}
	if result.FreeUseApiKeyShareID.Valid {
		inst.FreeUseApiKeyShareID = &result.FreeUseApiKeyShareID.UUID
	}
	if result.PromptConstraints.Valid {
		inst.PromptConstraints = &result.PromptConstraints.String
	}
	if err := fillInstitutionDetails(ctx, inst, result); err != nil {
		return nil, err
	}
	return inst, nil
}

// GetInstitutionName retrieves just the institution name by ID (no permission check, for display only)
func GetInstitutionName(ctx context.Context, id uuid.UUID) (string, error) {
	result, err := queries().GetInstitutionByID(ctx, id)
	if err != nil {
		return "", obj.ErrNotFound("institution not found")
	}
	return result.Name, nil
}

// GetInstitutionByID retrieves an institution by ID (admin or member of institution)
func GetInstitutionByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*obj.Institution, error) {
	// Check permission
	if err := canAccessInstitution(ctx, userID, OpRead, &id); err != nil {
		return nil, err
	}

	result, err := queries().GetInstitutionByID(ctx, id)
	if err != nil {
		return nil, obj.ErrNotFound("institution not found")
	}

	institution := &obj.Institution{
		ID:   result.ID,
		Name: result.Name,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}
	if result.FreeUseApiKeyShareID.Valid {
		institution.FreeUseApiKeyShareID = &result.FreeUseApiKeyShareID.UUID
	}
	if result.FreeUseAiQualityTier.Valid {
		institution.FreeUseAiQualityTier = &result.FreeUseAiQualityTier.String
	}
	if result.PromptConstraints.Valid {
		institution.PromptConstraints = &result.PromptConstraints.String
	}
	if err := fillInstitutionDetails(ctx, institution, result); err != nil {
		return nil, err
	}

	// Load members if user has permission (admin, head, or staff of this institution)
	if err := canAccessInstitutionMembers(ctx, userID, OpRead, id, nil); err == nil {
		members, err := loadInstitutionMembers(ctx, id)
		if err == nil {
			institution.Members = members
		}
	}

	return institution, nil
}

// loadInstitutionMembers loads all users with roles for a given institution
func loadInstitutionMembers(ctx context.Context, institutionID uuid.UUID) ([]obj.InstitutionMember, error) {
	rows, err := queries().GetInstitutionMembers(ctx, uuid.NullUUID{UUID: institutionID, Valid: true})
	if err != nil {
		return nil, err
	}

	members := make([]obj.InstitutionMember, 0, len(rows))
	for _, row := range rows {
		member := obj.InstitutionMember{
			UserID: row.ID,
			Name:   row.Name.String,
		}
		if row.Email.Valid {
			member.Email = &row.Email.String
		}
		member.Role = obj.Role(row.Role)
		members = append(members, member)
	}

	return members, nil
}

// ListInstitutions retrieves institutions based on user permissions
// - Admins see all institutions
// - Heads/staff see only their own institution
//
// With registeredSince, only those that registered themselves since then,
// as only admins see.
func ListInstitutions(ctx context.Context, userID uuid.UUID, registeredSince *time.Time) ([]obj.Institution, error) {
	// Check permission
	if err := canAccessInstitution(ctx, userID, OpList, nil); err != nil {
		return nil, err
	}

	// Get user to check role
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.ErrNotFound("user not found")
	}

	var results []Institution

	// Admin can see all institutions
	if user.Role != nil && user.Role.Role == obj.RoleAdmin {
		results, err = queries().ListInstitutions(ctx)
		if err != nil {
			return nil, obj.ErrServerError("failed to list institutions")
		}
	} else if user.Role != nil && user.Role.Institution != nil {
		// Head/staff can only see their own institution
		inst, err := queries().GetInstitutionByID(ctx, user.Role.Institution.ID)
		if err != nil {
			return nil, obj.ErrServerError("failed to get institution")
		}
		results = []Institution{inst}
	} else {
		// User has no institution role
		return []obj.Institution{}, nil
	}

	institutions := make([]obj.Institution, 0, len(results))
	for _, r := range results {
		inst := obj.Institution{
			ID:   r.ID,
			Name: r.Name,
			Meta: obj.Meta{
				CreatedBy:  r.CreatedBy,
				CreatedAt:  &r.CreatedAt,
				ModifiedBy: r.ModifiedBy,
				ModifiedAt: &r.ModifiedAt,
			},
		}
		if r.FreeUseApiKeyShareID.Valid {
			inst.FreeUseApiKeyShareID = &r.FreeUseApiKeyShareID.UUID
		}
		if r.PromptConstraints.Valid {
			inst.PromptConstraints = &r.PromptConstraints.String
		}
		if err := fillInstitutionDetails(ctx, &inst, r); err != nil {
			return nil, err
		}
		if user.Role.Role == obj.RoleAdmin {
			if err := fillAdminInstitutionView(ctx, &inst, r); err != nil {
				return nil, err
			}
		}
		if registeredSince != nil && (inst.Registration == nil || inst.Registration.At.Before(*registeredSince)) {
			continue
		}
		members, err := loadInstitutionMembers(ctx, r.ID)
		if err == nil {
			inst.Members = members
		}
		institutions = append(institutions, inst)
	}

	return institutions, nil
}

// UpdateInstitution updates an institution's name (admin or head of institution)
func UpdateInstitution(ctx context.Context, id uuid.UUID, modifiedBy uuid.UUID, name string) (*obj.Institution, error) {
	// Check permission
	if err := canAccessInstitution(ctx, modifiedBy, OpUpdate, &id); err != nil {
		return nil, err
	}

	// Get existing to preserve created fields
	existing, err := queries().GetInstitutionByID(ctx, id)
	if err != nil {
		return nil, obj.ErrNotFound("institution not found")
	}

	now := time.Now()
	arg := UpdateInstitutionParams{
		ID:         id,
		CreatedBy:  existing.CreatedBy,
		CreatedAt:  existing.CreatedAt,
		ModifiedBy: uuid.NullUUID{UUID: modifiedBy, Valid: true},
		ModifiedAt: now,
		Name:       name,
	}

	result, err := queries().UpdateInstitution(ctx, arg)
	if err != nil {
		return nil, obj.ErrServerError("failed to update institution")
	}

	inst2 := &obj.Institution{
		ID:   result.ID,
		Name: result.Name,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}
	if result.FreeUseApiKeyShareID.Valid {
		inst2.FreeUseApiKeyShareID = &result.FreeUseApiKeyShareID.UUID
	}
	if result.PromptConstraints.Valid {
		inst2.PromptConstraints = &result.PromptConstraints.String
	}
	return inst2, nil
}

// UpdateInstitutionPromptConstraints sets or clears the prompt constraints for an institution.
// Permission check: head of the institution or admin.
func UpdateInstitutionPromptConstraints(ctx context.Context, userID uuid.UUID, institutionID uuid.UUID, constraints *string) error {
	if err := canAccessInstitution(ctx, userID, OpUpdate, &institutionID); err != nil {
		return err
	}

	return queries().UpdateInstitutionPromptConstraints(ctx, UpdateInstitutionPromptConstraintsParams{
		ID:                institutionID,
		PromptConstraints: stringPtrToNullString(constraints),
	})
}

// SetInstitutionFreeUseApiKeyShare sets or clears the free-use API key share for an institution.
// Permission check: head of the institution or admin.
func SetInstitutionFreeUseApiKeyShare(ctx context.Context, userID uuid.UUID, institutionID uuid.UUID, shareID *uuid.UUID) error {
	// Check permission - must be head or admin
	if err := canAccessInstitution(ctx, userID, OpUpdate, &institutionID); err != nil {
		return err
	}

	return queries().SetInstitutionFreeUseApiKeyShare(ctx, SetInstitutionFreeUseApiKeyShareParams{
		ID:                   institutionID,
		FreeUseApiKeyShareID: uuidPtrToNullUUID(shareID),
	})
}

// DeleteInstitution deletes an institution and all its data (admin only).
// Cascade: workshops (with their Guests and games), member users (with their data),
// invites, API key shares, free-use key ref.
func DeleteInstitution(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	// Check permission - only admin can delete institutions
	if err := canAccessInstitution(ctx, deletedBy, OpDelete, nil); err != nil {
		return err
	}

	// 1. Delete all workshops (cascades to their Guests, games, etc.)
	workshopIDs, _ := queries().GetWorkshopIDsByInstitution(ctx, id)
	for _, wsID := range workshopIDs {
		// Use deletedBy (admin) to bypass per-workshop permission checks
		_ = DeleteWorkshop(ctx, wsID, deletedBy)
	}

	// 2. Delete the remaining Guests
	guestIDs, _ := queries().GetGuestUserIDsByInstitution(ctx, uuid.NullUUID{UUID: id, Valid: true})
	for _, guestID := range guestIDs {
		_ = DeleteUser(ctx, guestID)
	}

	// 3. Collect the other members (head/staff/individual) before removing roles
	memberIDs, _ := queries().GetNonGuestUserIDsByInstitution(ctx, uuid.NullUUID{UUID: id, Valid: true})

	// 4. Clean up remaining FK references before hard-delete
	_ = queries().DeleteInvitesByInstitution(ctx, id)
	_ = queries().DeleteApiKeySharesByInstitution(ctx, uuid.NullUUID{UUID: id, Valid: true})
	_ = queries().DeleteUserRolesByInstitution(ctx, uuid.NullUUID{UUID: id, Valid: true})
	_ = queries().HardDeleteWorkshopsByInstitution(ctx, id)
	_ = queries().ClearInstitutionFreeUseApiKeyShare(ctx, id)

	// 5. Assign individual role to former members so they aren't left without a role
	for _, memberID := range memberIDs {
		_ = assignDefaultIndividualRole(ctx, memberID)
	}

	// 6. Hard-delete the institution
	if err := queries().HardDeleteInstitution(ctx, id); err != nil {
		return obj.ErrServerError("failed to delete institution")
	}
	return nil
}

// GetInstitutionMembers returns all members of an institution
// Email addresses are only visible to head and staff members
func GetInstitutionMembers(ctx context.Context, institutionID uuid.UUID, userID uuid.UUID) ([]obj.User, error) {
	// Check permission - must be a member or admin to view members
	if err := canAccessInstitutionMembers(ctx, userID, OpRead, institutionID, nil); err != nil {
		return nil, err
	}

	// Determine if the requesting user can see email addresses
	// Only head and staff (and admin) can see emails
	canSeeEmails := false
	currentUser, err := GetUserByID(ctx, userID)
	if err == nil && currentUser.Role != nil {
		role := currentUser.Role.Role
		canSeeEmails = role == obj.RoleAdmin || role == obj.RoleHead || role == obj.RoleStaff
	}

	// Get all users with roles in this institution
	dbUsers, err := queries().GetUsersByInstitution(ctx, uuid.NullUUID{UUID: institutionID, Valid: true})
	if err != nil {
		return nil, obj.ErrServerError("failed to get institution members")
	}

	users := make([]obj.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		var email *string
		// Only include email if the requester has permission to see it
		if canSeeEmails && dbUser.Email.Valid {
			email = &dbUser.Email.String
		}

		users[i] = obj.User{
			ID:    dbUser.ID,
			Name:  dbUser.Name.String,
			Email: email,
			Meta: obj.Meta{
				CreatedBy:  dbUser.CreatedBy,
				CreatedAt:  &dbUser.CreatedAt,
				ModifiedBy: dbUser.ModifiedBy,
				ModifiedAt: &dbUser.ModifiedAt,
			},
		}

		users[i].Role = &obj.UserRole{
			ID:   dbUser.RoleID,
			Role: obj.Role(dbUser.RoleRole),
		}
	}

	return users, nil
}

// RemoveInstitutionMember removes a member from an institution
func RemoveInstitutionMember(ctx context.Context, institutionID uuid.UUID, memberUserID uuid.UUID, requestingUserID uuid.UUID) error {
	// Check permission - must be head or admin to remove members
	// This also validates business rules (e.g., preventing removal of last head, self-removal for heads)
	if err := canAccessInstitutionMembers(ctx, requestingUserID, OpDelete, institutionID, &memberUserID); err != nil {
		return err
	}

	// Clean up API key shares owned by this user that target the institution
	if err := queries().DeleteApiKeySharesByOwnerForInstitution(ctx, DeleteApiKeySharesByOwnerForInstitutionParams{
		UserID:        memberUserID,
		InstitutionID: uuid.NullUUID{UUID: institutionID, Valid: true},
	}); err != nil {
		return obj.ErrServerError("failed to clean up institution API key shares")
	}

	// Clean up API key shares owned by this user that target any workshop in this institution
	if err := queries().DeleteApiKeySharesByOwnerForInstitutionWorkshops(ctx, DeleteApiKeySharesByOwnerForInstitutionWorkshopsParams{
		UserID:        memberUserID,
		InstitutionID: institutionID,
	}); err != nil {
		return obj.ErrServerError("failed to clean up workshop API key shares")
	}

	// Atomically swap the institutional role for an "individual" role so the
	// user is never left with zero roles if the second step fails.
	tx, err := sqlDb().BeginTx(ctx, nil)
	if err != nil {
		return obj.ErrServerError("failed to begin transaction")
	}
	defer tx.Rollback()

	txQueries := queries().WithTx(tx)

	if err := txQueries.DeleteUserRoles(ctx, memberUserID); err != nil {
		return obj.ErrServerError("failed to remove member")
	}

	if _, err := txQueries.CreateUserRole(ctx, CreateUserRoleParams{
		UserID: memberUserID,
		Role:   string(obj.RoleIndividual),
	}); err != nil {
		return obj.ErrServerError("failed to assign individual role after member removal")
	}

	if err := tx.Commit(); err != nil {
		return obj.ErrServerError("failed to commit member removal")
	}

	return nil
}

// fillInstitutionDetails copies operator, address and the suspension into
// inst. It is suspended by an admin (suspendedAt), or for want of a consent to
// the current declaration.
func fillInstitutionDetails(ctx context.Context, inst *obj.Institution, r Institution) error {
	inst.Operator = sqlNullStringToMaybeString(r.Operator)
	inst.Street = sqlNullStringToMaybeString(r.Street)
	inst.PostalCode = sqlNullStringToMaybeString(r.PostalCode)
	inst.City = sqlNullStringToMaybeString(r.City)
	if r.SuspendedAt.Valid {
		inst.SuspendedAt = &r.SuspendedAt.Time
	}
	declaration := legal.CurrentDeclaration()
	suspended, err := queries().IsInstitutionSuspended(ctx, IsInstitutionSuspendedParams{
		ID: r.ID, Version: declaration.Version, Hash: declaration.Hash,
	})
	if err != nil {
		return obj.ErrServerError("failed to read the institution's suspension")
	}
	inst.Suspended = suspended
	return nil
}

// fillAdminInstitutionView adds what only the admins' list shows: the
// suspension's reason, the consent status, and how the institution
// registered itself.
func fillAdminInstitutionView(ctx context.Context, inst *obj.Institution, r Institution) error {
	inst.SuspendedReason = sqlNullStringToMaybeString(r.SuspendedReason)
	latest, err := queries().GetLatestInstitutionConsent(ctx, r.ID)
	if err != nil {
		return obj.ErrServerError("failed to load consent")
	}
	var entry *legal.LogEntry
	if len(latest) > 0 {
		entry = &legal.LogEntry{
			Withdrawal: latest[0].Kind == "withdrawal",
			Version:    latest[0].DeclarationVersion,
			Hash:       latest[0].DeclarationHash,
			At:         latest[0].CreatedAt,
		}
	}
	status := legal.ConsentStatusOf(entry)
	inst.Declaration = &status

	first, err := queries().GetFirstInstitutionConsent(ctx, r.ID)
	if err != nil {
		return obj.ErrServerError("failed to load consent")
	}
	// It registered itself when its creator gave its first consent; an
	// institution an admin created was consented to by a head later.
	if len(first) > 0 && r.CreatedBy.Valid && first[0].UserID == r.CreatedBy.UUID {
		inst.Registration = &obj.InstitutionRegistration{
			At:             first[0].CreatedAt,
			PersonName:     first[0].PersonName.String,
			PersonFunction: first[0].PersonFunction.String,
			Email:          first[0].Email.String,
		}
	}
	return nil
}

// IsWorkshopInstitutionSuspended reports whether the workshop's institution
// is suspended, by an admin or for want of a consent; an error counts as not
// suspended, as v1 has it.
func IsWorkshopInstitutionSuspended(ctx context.Context, workshopID uuid.UUID) bool {
	declaration := legal.CurrentDeclaration()
	suspended, err := queries().IsWorkshopInstitutionSuspended(ctx, IsWorkshopInstitutionSuspendedParams{
		WorkshopID: workshopID, Version: declaration.Version, Hash: declaration.Hash,
	})
	return err == nil && suspended
}

// IsInstitutionSuspended reports whether the institution is suspended, by an
// admin or for want of a consent; an error counts as not suspended, as v1 has
// it.
func IsInstitutionSuspended(ctx context.Context, id uuid.UUID) bool {
	declaration := legal.CurrentDeclaration()
	suspended, err := queries().IsInstitutionSuspended(ctx, IsInstitutionSuspendedParams{
		ID: id, Version: declaration.Version, Hash: declaration.Hash,
	})
	return err == nil && suspended
}
