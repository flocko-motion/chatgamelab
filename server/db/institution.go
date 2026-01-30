package db

import (
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"
	"time"

	"github.com/google/uuid"
)

// CreateInstitution creates a new institution (admin only)
func CreateInstitution(ctx context.Context, createdBy uuid.UUID, name string) (*obj.Institution, error) {
	// Check permission
	if err := canAccessInstitution(ctx, createdBy, OpCreate, nil); err != nil {
		return nil, err
	}

	now := time.Now()

	arg := db.CreateInstitutionParams{
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

	return &obj.Institution{
		ID:   result.ID,
		Name: result.Name,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
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
			Name:   row.Name,
		}
		if row.Email.Valid {
			member.Email = &row.Email.String
		}
		if row.Role.Valid {
			member.Role = obj.Role(row.Role.String)
		}
		members = append(members, member)
	}

	return members, nil
}

// ListInstitutions retrieves institutions based on user permissions
// - Admins see all institutions
// - Heads/staff see only their own institution
func ListInstitutions(ctx context.Context, userID uuid.UUID) ([]obj.Institution, error) {
	// Check permission
	if err := canAccessInstitution(ctx, userID, OpList, nil); err != nil {
		return nil, err
	}

	// Get user to check role
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.ErrNotFound("user not found")
	}

	var results []db.Institution

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
		results = []db.Institution{inst}
	} else {
		// User has no institution role
		return []obj.Institution{}, nil
	}

	institutions := make([]obj.Institution, 0, len(results))
	for _, r := range results {
		institutions = append(institutions, obj.Institution{
			ID:   r.ID,
			Name: r.Name,
			Meta: obj.Meta{
				CreatedBy:  r.CreatedBy,
				CreatedAt:  &r.CreatedAt,
				ModifiedBy: r.ModifiedBy,
				ModifiedAt: &r.ModifiedAt,
			},
		})
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
	arg := db.UpdateInstitutionParams{
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

	return &obj.Institution{
		ID:   result.ID,
		Name: result.Name,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
}

// DeleteInstitution soft-deletes an institution (admin only)
func DeleteInstitution(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	// Check permission - only admin can delete institutions
	if err := canAccessInstitution(ctx, deletedBy, OpDelete, nil); err != nil {
		return err
	}

	err := queries().DeleteInstitution(ctx, id)
	if err != nil {
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
			Name:  dbUser.Name,
			Email: email,
			Meta: obj.Meta{
				CreatedBy:  dbUser.CreatedBy,
				CreatedAt:  &dbUser.CreatedAt,
				ModifiedBy: dbUser.ModifiedBy,
				ModifiedAt: &dbUser.ModifiedAt,
			},
		}

		// Add role information if available
		if dbUser.RoleID.Valid {
			users[i].Role = &obj.UserRole{
				ID:   dbUser.RoleID.UUID,
				Role: obj.Role(dbUser.RoleRole.String),
			}
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

	// Delete the user's role (which removes them from the institution)
	err := queries().DeleteUserRole(ctx, memberUserID)
	if err != nil {
		return obj.ErrServerError("failed to remove member")
	}

	return nil
}
