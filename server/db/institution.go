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
	if canViewInstitutionMembers(ctx, userID, id) {
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

// ListInstitutions retrieves all non-deleted institutions (admin only)
func ListInstitutions(ctx context.Context, userID uuid.UUID) ([]obj.Institution, error) {
	// Check permission
	if err := canAccessInstitution(ctx, userID, OpList, nil); err != nil {
		return nil, err
	}

	results, err := queries().ListInstitutions(ctx)
	if err != nil {
		return nil, obj.ErrServerError("failed to list institutions")
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
