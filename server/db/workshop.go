package db

import (
	db "cgl/db/sqlc"
	"cgl/obj"
	"context"
	"time"

	"github.com/google/uuid"
)

// CreateWorkshop creates a new workshop (admin or head/staff of institution)
// If institutionID is nil, it will be auto-resolved from the user's role (non-admin users only)
func CreateWorkshop(ctx context.Context, createdBy uuid.UUID, institutionID *uuid.UUID, name string, active, public bool) (*obj.Workshop, error) {
	// Get user to check role and potentially auto-resolve institution
	user, err := GetUserByID(ctx, createdBy)
	if err != nil {
		return nil, obj.ErrNotFound("user not found")
	}

	// Auto-resolve institutionID if not provided
	var finalInstitutionID uuid.UUID
	if institutionID == nil {
		// Only non-admin users can auto-resolve from their role
		if user.Role == nil || user.Role.Role == obj.RoleAdmin {
			return nil, obj.ErrValidation("admin users must specify institutionId")
		}
		if user.Role.Institution == nil {
			return nil, obj.ErrValidation("user has no institution assigned")
		}
		finalInstitutionID = user.Role.Institution.ID
	} else {
		finalInstitutionID = *institutionID
	}

	// Check permission
	if err := canAccessWorkshop(ctx, createdBy, OpCreate, finalInstitutionID, nil, uuid.Nil); err != nil {
		return nil, err
	}

	now := time.Now()
	id := uuid.New()

	arg := db.CreateWorkshopParams{
		ID:            id,
		CreatedBy:     uuid.NullUUID{UUID: createdBy, Valid: true},
		CreatedAt:     now,
		ModifiedBy:    uuid.NullUUID{UUID: createdBy, Valid: true},
		ModifiedAt:    now,
		Name:          name,
		InstitutionID: finalInstitutionID,
		Active:        active,
		Public:        public,
	}

	result, err := queries().CreateWorkshop(ctx, arg)
	if err != nil {
		return nil, obj.ErrServerError("failed to create workshop")
	}

	return &obj.Workshop{
		ID:          result.ID,
		Name:        result.Name,
		Institution: &obj.Institution{ID: result.InstitutionID},
		Active:      result.Active,
		Public:      result.Public,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
}

// GetWorkshopByID retrieves a workshop by ID (admin or any member of institution)
func GetWorkshopByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*obj.Workshop, error) {
	result, err := queries().GetWorkshopByID(ctx, id)
	if err != nil {
		return nil, obj.ErrNotFound("workshop not found")
	}

	// Check permission for this workshop's institution
	if err := canAccessWorkshop(ctx, userID, OpRead, result.InstitutionID, &id, uuid.Nil); err != nil {
		return nil, err
	}

	// Fetch invites for this workshop
	inviteRows, err := queries().GetInvitesByWorkshop(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		// Don't fail if we can't get invites, just return empty list
		inviteRows = []db.UserRoleInvite{}
	}

	// Convert invites to obj.UserRoleInvite
	invites := make([]obj.UserRoleInvite, 0, len(inviteRows))
	for _, inv := range inviteRows {
		invite := obj.UserRoleInvite{
			ID:            inv.ID,
			InstitutionID: inv.InstitutionID,
			Role:          obj.Role(inv.Role),
			UsesCount:     inv.UsesCount,
			Status:        obj.InviteStatus(inv.Status),
			Meta: obj.Meta{
				CreatedBy:  inv.CreatedBy,
				CreatedAt:  &inv.CreatedAt,
				ModifiedBy: inv.ModifiedBy,
				ModifiedAt: &inv.ModifiedAt,
			},
		}
		if inv.WorkshopID.Valid {
			invite.WorkshopID = &inv.WorkshopID.UUID
		}
		if inv.InvitedUserID.Valid {
			invite.InvitedUserID = &inv.InvitedUserID.UUID
		}
		if inv.InvitedEmail.Valid {
			invite.InvitedEmail = &inv.InvitedEmail.String
		}
		if inv.InviteToken.Valid {
			invite.InviteToken = &inv.InviteToken.String
		}
		if inv.MaxUses.Valid {
			invite.MaxUses = &inv.MaxUses.Int32
		}
		if inv.ExpiresAt.Valid {
			invite.ExpiresAt = &inv.ExpiresAt.Time
		}
		if inv.AcceptedAt.Valid {
			invite.AcceptedAt = &inv.AcceptedAt.Time
		}
		if inv.AcceptedBy.Valid {
			invite.AcceptedBy = &inv.AcceptedBy.UUID
		}
		invites = append(invites, invite)
	}

	return &obj.Workshop{
		ID:          result.ID,
		Name:        result.Name,
		Institution: &obj.Institution{ID: result.InstitutionID},
		Active:      result.Active,
		Public:      result.Public,
		Invites:     invites,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
}

// ListWorkshops retrieves workshops with optional institution filter
// - If institutionID is nil: only admin can list all workshops
// - If institutionID is set: admin or head/staff of that institution can list
func ListWorkshops(ctx context.Context, userID uuid.UUID, institutionID *uuid.UUID) ([]obj.Workshop, error) {
	if institutionID == nil {
		// Listing all workshops - only admin allowed
		if err := canAccessInstitution(ctx, userID, OpList, nil); err != nil {
			return nil, err
		}

		results, err := queries().ListWorkshops(ctx)
		if err != nil {
			return nil, obj.ErrServerError("failed to list workshops")
		}

		workshops := make([]obj.Workshop, 0, len(results))
		for _, r := range results {
			workshops = append(workshops, obj.Workshop{
				ID:          r.ID,
				Name:        r.Name,
				Institution: &obj.Institution{ID: r.InstitutionID},
				Active:      r.Active,
				Public:      r.Public,
				Meta: obj.Meta{
					CreatedBy:  r.CreatedBy,
					CreatedAt:  &r.CreatedAt,
					ModifiedBy: r.ModifiedBy,
					ModifiedAt: &r.ModifiedAt,
				},
			})
		}
		return workshops, nil
	} else {
		// Listing workshops for specific institution - admin or head/staff of institution
		if err := canAccessWorkshop(ctx, userID, OpList, *institutionID, nil, uuid.Nil); err != nil {
			return nil, err
		}

		results, err := queries().ListWorkshopsByInstitution(ctx, *institutionID)
		if err != nil {
			return nil, obj.ErrServerError("failed to list workshops")
		}

		workshops := make([]obj.Workshop, 0, len(results))
		for _, r := range results {
			workshops = append(workshops, obj.Workshop{
				ID:          r.ID,
				Name:        r.Name,
				Institution: &obj.Institution{ID: r.InstitutionID},
				Active:      r.Active,
				Public:      r.Public,
				Meta: obj.Meta{
					CreatedBy:  r.CreatedBy,
					CreatedAt:  &r.CreatedAt,
					ModifiedBy: r.ModifiedBy,
					ModifiedAt: &r.ModifiedAt,
				},
			})
		}
		return workshops, nil
	}
}

// UpdateWorkshop updates a workshop (admin, head of institution, or staff who created it)
func UpdateWorkshop(ctx context.Context, id uuid.UUID, modifiedBy uuid.UUID, name string, active, public bool) (*obj.Workshop, error) {
	// Get existing to preserve created fields and institution_id
	existing, err := queries().GetWorkshopByID(ctx, id)
	if err != nil {
		return nil, obj.ErrNotFound("workshop not found")
	}

	// Check permission - need to know who created it
	var createdByID uuid.UUID
	if existing.CreatedBy.Valid {
		createdByID = existing.CreatedBy.UUID
	}
	if err := canAccessWorkshop(ctx, modifiedBy, OpUpdate, existing.InstitutionID, &id, createdByID); err != nil {
		return nil, err
	}

	existing, err = queries().GetWorkshopByID(ctx, id)
	if err != nil {
		return nil, obj.ErrNotFound("workshop not found")
	}

	now := time.Now()
	arg := db.UpdateWorkshopParams{
		ID:            id,
		CreatedBy:     existing.CreatedBy,
		CreatedAt:     existing.CreatedAt,
		ModifiedBy:    uuid.NullUUID{UUID: modifiedBy, Valid: true},
		ModifiedAt:    now,
		Name:          name,
		InstitutionID: existing.InstitutionID,
		Active:        active,
		Public:        public,
	}

	result, err := queries().UpdateWorkshop(ctx, arg)
	if err != nil {
		return nil, obj.ErrServerError("failed to update workshop")
	}

	return &obj.Workshop{
		ID:          result.ID,
		Name:        result.Name,
		Institution: &obj.Institution{ID: result.InstitutionID},
		Active:      result.Active,
		Public:      result.Public,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
}

// DeleteWorkshop soft-deletes a workshop (admin, head of institution, or staff who created it)
func DeleteWorkshop(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	// Get workshop to check institution and creator
	workshop, err := queries().GetWorkshopByID(ctx, id)
	if err != nil {
		return obj.ErrNotFound("workshop not found")
	}

	// Check permission - need to know who created it
	var createdByID uuid.UUID
	if workshop.CreatedBy.Valid {
		createdByID = workshop.CreatedBy.UUID
	}
	if err := canAccessWorkshop(ctx, deletedBy, OpDelete, workshop.InstitutionID, &id, createdByID); err != nil {
		return err
	}

	err = queries().DeleteWorkshop(ctx, id)
	if err != nil {
		return obj.ErrServerError("failed to delete workshop")
	}
	return nil
}
