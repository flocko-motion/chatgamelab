package db

import (
	db "cgl/db/sqlc"
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"
	"sort"
	"strings"
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

// GetWorkshopName retrieves just the workshop name by ID (no permission check, for display only)
func GetWorkshopName(ctx context.Context, id uuid.UUID) (string, error) {
	result, err := queries().GetWorkshopByID(ctx, id)
	if err != nil {
		return "", obj.ErrNotFound("workshop not found")
	}
	return result.Name, nil
}

// GetWorkshopByID retrieves a workshop by ID (admin or any member of institution)
func GetWorkshopByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*obj.Workshop, error) {
	result, err := queries().GetWorkshopByID(ctx, id)
	if err != nil {
		return nil, obj.ErrNotFound("workshop not found")
	}

	// Public workshops can be viewed by anyone
	// Private workshops require institution membership
	if !result.Public {
		if err := canAccessWorkshop(ctx, userID, OpRead, result.InstitutionID, &id, uuid.Nil); err != nil {
			return nil, err
		}
	}

	// Fetch invites for this workshop (only if user has permission)
	var inviteRows []db.UserRoleInvite
	if err := canAccessWorkshopInvites(ctx, userID, result.InstitutionID); err == nil {
		inviteRows, err = queries().GetInvitesByWorkshop(ctx, uuid.NullUUID{UUID: id, Valid: true})
		if err != nil {
			// Don't fail if we can't get invites, just return empty list
			inviteRows = []db.UserRoleInvite{}
		}
	} else {
		// User doesn't have permission to view invites, return empty list
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

	// Fetch workshop participants (only visible to participants, workshop owner, and institution heads)
	var participants []obj.WorkshopParticipant

	// Check if user has permission to see participants
	createdBy := uuid.Nil
	if result.CreatedBy.Valid {
		createdBy = result.CreatedBy.UUID
	}

	if err := canAccessWorkshopParticipants(ctx, userID, id, createdBy, result.InstitutionID); err == nil {
		participantRows, err := queries().GetWorkshopParticipants(ctx, id)
		if err != nil {
			// Don't fail if we can't get participants, just return empty list
			log.Warn("failed to get workshop participants", "workshop_id", id, "error", err)
			participantRows = []db.GetWorkshopParticipantsRow{}
		}

		// Convert participants to obj.WorkshopParticipant
		participants = make([]obj.WorkshopParticipant, 0, len(participantRows))
		for _, p := range participantRows {
			// Parse role from database
			role, err := stringToRole(p.Role.String)
			if err != nil {
				// Default to participant if role parsing fails
				role = obj.RoleParticipant
			}

			participant := obj.WorkshopParticipant{
				ID:          p.ID,
				WorkshopID:  id,
				Name:        p.Name,
				Role:        role,
				AccessToken: p.Auth0ID.String, // Auth token stored in auth0_id field
				Active:      true,
				GamesCount:  int(p.GamesCount),
				Meta: obj.Meta{
					CreatedAt: &p.JoinedAt,
				},
			}
			participants = append(participants, participant)
		}
	} else {
		// User doesn't have permission to see participants, return empty list
		participants = []obj.WorkshopParticipant{}
	}

	var defaultApiKeyShareID *uuid.UUID
	if result.DefaultApiKeyShareID.Valid {
		defaultApiKeyShareID = &result.DefaultApiKeyShareID.UUID
	}

	var useSpecificAiModel *string
	if result.UseSpecificAiModel.Valid {
		useSpecificAiModel = &result.UseSpecificAiModel.String
	}

	return &obj.Workshop{
		ID:                         result.ID,
		Name:                       result.Name,
		Institution:                &obj.Institution{ID: result.InstitutionID},
		Active:                     result.Active,
		Public:                     result.Public,
		DefaultApiKeyShareID:       defaultApiKeyShareID,
		Participants:               participants,
		Invites:                    invites,
		UseSpecificAiModel:         useSpecificAiModel,
		ShowAiModelSelector:        result.ShowAiModelSelector,
		ShowPublicGames:            result.ShowPublicGames,
		ShowOtherParticipantsGames: result.ShowOtherParticipantsGames,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
}

// ListWorkshopsOptions contains filtering and sorting options for listing workshops
type ListWorkshopsOptions struct {
	Search     string
	SortBy     string // "name", "createdAt", "participantCount"
	SortDir    string // "asc", "desc"
	ActiveOnly *bool
}

// fetchWorkshopParticipants retrieves participants for a workshop (helper for list operations)
func fetchWorkshopParticipants(ctx context.Context, workshopID uuid.UUID) []obj.WorkshopParticipant {
	participantRows, err := queries().GetWorkshopParticipants(ctx, workshopID)
	if err != nil {
		return []obj.WorkshopParticipant{}
	}

	participants := make([]obj.WorkshopParticipant, 0, len(participantRows))
	for _, p := range participantRows {
		// Parse role from database
		role, err := stringToRole(p.Role.String)
		if err != nil {
			// Default to participant if role parsing fails
			role = obj.RoleParticipant
		}

		participant := obj.WorkshopParticipant{
			ID:          p.ID,
			WorkshopID:  workshopID,
			Name:        p.Name,
			AccessToken: p.Auth0ID.String,
			Active:      true,
			Role:        role,
			GamesCount:  int(p.GamesCount),
			Meta: obj.Meta{
				CreatedAt: &p.JoinedAt,
			},
		}
		participants = append(participants, participant)
	}
	return participants
}

// fetchWorkshopInvites retrieves invites for a workshop (helper for list operations)
func fetchWorkshopInvites(ctx context.Context, workshopID uuid.UUID) []obj.UserRoleInvite {
	inviteRows, err := queries().GetInvitesByWorkshop(ctx, uuid.NullUUID{UUID: workshopID, Valid: true})
	if err != nil {
		return []obj.UserRoleInvite{}
	}

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
	return invites
}

// ListWorkshopsWithOptions retrieves workshops with optional institution filter and options
func ListWorkshopsWithOptions(ctx context.Context, userID uuid.UUID, institutionID *uuid.UUID, opts ListWorkshopsOptions) ([]obj.Workshop, error) {
	// Get base list first
	workshops, err := ListWorkshops(ctx, userID, institutionID)
	if err != nil {
		return nil, err
	}

	// Apply search filter
	if opts.Search != "" {
		searchLower := strings.ToLower(opts.Search)
		filtered := make([]obj.Workshop, 0)
		for _, w := range workshops {
			if strings.Contains(strings.ToLower(w.Name), searchLower) {
				filtered = append(filtered, w)
			}
		}
		workshops = filtered
	}

	// Apply active filter
	if opts.ActiveOnly != nil && *opts.ActiveOnly {
		filtered := make([]obj.Workshop, 0)
		for _, w := range workshops {
			if w.Active {
				filtered = append(filtered, w)
			}
		}
		workshops = filtered
	}

	// Apply sorting
	sortDir := opts.SortDir
	if sortDir == "" {
		sortDir = "asc"
	}

	switch opts.SortBy {
	case "name":
		sort.Slice(workshops, func(i, j int) bool {
			if sortDir == "desc" {
				return workshops[i].Name > workshops[j].Name
			}
			return workshops[i].Name < workshops[j].Name
		})
	case "createdAt":
		sort.Slice(workshops, func(i, j int) bool {
			ti := time.Time{}
			tj := time.Time{}
			if workshops[i].Meta.CreatedAt != nil {
				ti = *workshops[i].Meta.CreatedAt
			}
			if workshops[j].Meta.CreatedAt != nil {
				tj = *workshops[j].Meta.CreatedAt
			}
			if sortDir == "desc" {
				return ti.After(tj)
			}
			return ti.Before(tj)
		})
	case "participantCount":
		sort.Slice(workshops, func(i, j int) bool {
			ci := len(workshops[i].Participants)
			cj := len(workshops[j].Participants)
			if sortDir == "desc" {
				return ci > cj
			}
			return ci < cj
		})
	default:
		// Default: sort by createdAt desc
		sort.Slice(workshops, func(i, j int) bool {
			ti := time.Time{}
			tj := time.Time{}
			if workshops[i].Meta.CreatedAt != nil {
				ti = *workshops[i].Meta.CreatedAt
			}
			if workshops[j].Meta.CreatedAt != nil {
				tj = *workshops[j].Meta.CreatedAt
			}
			return ti.After(tj)
		})
	}

	return workshops, nil
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
			var defaultApiKeyShareID *uuid.UUID
			if r.DefaultApiKeyShareID.Valid {
				defaultApiKeyShareID = &r.DefaultApiKeyShareID.UUID
			}

			var useSpecificAiModel *string
			if r.UseSpecificAiModel.Valid {
				useSpecificAiModel = &r.UseSpecificAiModel.String
			}

			workshop := obj.Workshop{
				ID:                         r.ID,
				Name:                       r.Name,
				Institution:                &obj.Institution{ID: r.InstitutionID},
				Active:                     r.Active,
				Public:                     r.Public,
				DefaultApiKeyShareID:       defaultApiKeyShareID,
				UseSpecificAiModel:         useSpecificAiModel,
				ShowAiModelSelector:        r.ShowAiModelSelector,
				ShowPublicGames:            r.ShowPublicGames,
				ShowOtherParticipantsGames: r.ShowOtherParticipantsGames,
				Meta: obj.Meta{
					CreatedBy:  r.CreatedBy,
					CreatedAt:  &r.CreatedAt,
					ModifiedBy: r.ModifiedBy,
					ModifiedAt: &r.ModifiedAt,
				},
			}

			// Fetch participants for this workshop
			participants := fetchWorkshopParticipants(ctx, r.ID)
			workshop.Participants = participants

			// Fetch invites for this workshop
			invites := fetchWorkshopInvites(ctx, r.ID)
			workshop.Invites = invites

			workshops = append(workshops, workshop)
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
			var defaultApiKeyShareID *uuid.UUID
			if r.DefaultApiKeyShareID.Valid {
				defaultApiKeyShareID = &r.DefaultApiKeyShareID.UUID
			}

			var useSpecificAiModel *string
			if r.UseSpecificAiModel.Valid {
				useSpecificAiModel = &r.UseSpecificAiModel.String
			}

			workshop := obj.Workshop{
				ID:                         r.ID,
				Name:                       r.Name,
				Institution:                &obj.Institution{ID: r.InstitutionID},
				Active:                     r.Active,
				Public:                     r.Public,
				DefaultApiKeyShareID:       defaultApiKeyShareID,
				UseSpecificAiModel:         useSpecificAiModel,
				ShowAiModelSelector:        r.ShowAiModelSelector,
				ShowPublicGames:            r.ShowPublicGames,
				ShowOtherParticipantsGames: r.ShowOtherParticipantsGames,
				Meta: obj.Meta{
					CreatedBy:  r.CreatedBy,
					CreatedAt:  &r.CreatedAt,
					ModifiedBy: r.ModifiedBy,
					ModifiedAt: &r.ModifiedAt,
				},
			}

			// Fetch participants for this workshop
			participants := fetchWorkshopParticipants(ctx, r.ID)
			workshop.Participants = participants

			// Fetch invites for this workshop
			invites := fetchWorkshopInvites(ctx, r.ID)
			workshop.Invites = invites

			workshops = append(workshops, workshop)
		}
		return workshops, nil
	}
}

// UpdateWorkshopParams contains the parameters for updating a workshop
type UpdateWorkshopParams struct {
	Name                       string
	Active                     bool
	Public                     bool
	UseSpecificAiModel         *string
	ShowAiModelSelector        bool
	ShowPublicGames            bool
	ShowOtherParticipantsGames bool
}

// UpdateWorkshop updates a workshop (admin, head of institution, or staff who created it)
func UpdateWorkshop(ctx context.Context, id uuid.UUID, modifiedBy uuid.UUID, params UpdateWorkshopParams) (*obj.Workshop, error) {
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
		ID:                         id,
		CreatedBy:                  existing.CreatedBy,
		CreatedAt:                  existing.CreatedAt,
		ModifiedBy:                 uuid.NullUUID{UUID: modifiedBy, Valid: true},
		ModifiedAt:                 now,
		Name:                       params.Name,
		InstitutionID:              existing.InstitutionID,
		Active:                     params.Active,
		Public:                     params.Public,
		DefaultApiKeyShareID:       existing.DefaultApiKeyShareID,
		ShowAiModelSelector:        params.ShowAiModelSelector,
		ShowPublicGames:            params.ShowPublicGames,
		ShowOtherParticipantsGames: params.ShowOtherParticipantsGames,
	}
	if params.UseSpecificAiModel != nil {
		arg.UseSpecificAiModel = sql.NullString{String: *params.UseSpecificAiModel, Valid: true}
	}

	result, err := queries().UpdateWorkshop(ctx, arg)
	if err != nil {
		return nil, obj.ErrServerError("failed to update workshop")
	}

	var updateDefaultApiKeyShareID *uuid.UUID
	if result.DefaultApiKeyShareID.Valid {
		updateDefaultApiKeyShareID = &result.DefaultApiKeyShareID.UUID
	}

	var useSpecificAiModel *string
	if result.UseSpecificAiModel.Valid {
		useSpecificAiModel = &result.UseSpecificAiModel.String
	}

	return &obj.Workshop{
		ID:                         result.ID,
		Name:                       result.Name,
		Institution:                &obj.Institution{ID: result.InstitutionID},
		Active:                     result.Active,
		Public:                     result.Public,
		DefaultApiKeyShareID:       updateDefaultApiKeyShareID,
		UseSpecificAiModel:         useSpecificAiModel,
		ShowAiModelSelector:        result.ShowAiModelSelector,
		ShowPublicGames:            result.ShowPublicGames,
		ShowOtherParticipantsGames: result.ShowOtherParticipantsGames,
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
}

// SetWorkshopDefaultApiKey sets the default API key share for a workshop (staff/heads only)
func SetWorkshopDefaultApiKey(ctx context.Context, workshopID uuid.UUID, modifiedBy uuid.UUID, apiKeyShareID *uuid.UUID) (*obj.Workshop, error) {
	// Get existing workshop to check permission
	existing, err := queries().GetWorkshopByID(ctx, workshopID)
	if err != nil {
		return nil, obj.ErrNotFound("workshop not found")
	}

	// Check permission - need to know who created it
	var createdByID uuid.UUID
	if existing.CreatedBy.Valid {
		createdByID = existing.CreatedBy.UUID
	}
	if err := canAccessWorkshop(ctx, modifiedBy, OpUpdate, existing.InstitutionID, &workshopID, createdByID); err != nil {
		return nil, err
	}

	// Build params
	arg := db.SetWorkshopDefaultApiKeyParams{
		ID:         workshopID,
		ModifiedBy: uuid.NullUUID{UUID: modifiedBy, Valid: true},
	}
	if apiKeyShareID != nil {
		arg.DefaultApiKeyShareID = uuid.NullUUID{UUID: *apiKeyShareID, Valid: true}
	} else {
		// Explicitly set to NULL when clearing
		arg.DefaultApiKeyShareID = uuid.NullUUID{Valid: false}
	}

	result, err := queries().SetWorkshopDefaultApiKey(ctx, arg)
	if err != nil {
		return nil, obj.ErrServerError("failed to set workshop API key")
	}

	var setDefaultApiKeyShareID *uuid.UUID
	if result.DefaultApiKeyShareID.Valid {
		setDefaultApiKeyShareID = &result.DefaultApiKeyShareID.UUID
	}

	return &obj.Workshop{
		ID:                   result.ID,
		Name:                 result.Name,
		Institution:          &obj.Institution{ID: result.InstitutionID},
		Active:               result.Active,
		Public:               result.Public,
		DefaultApiKeyShareID: setDefaultApiKeyShareID,
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

// GetWorkshopParticipantToken retrieves the access token for a workshop participant
// Only staff and heads of the institution owning the workshop can access this
// participantUserID is the user ID of the participant (not workshop_participant table ID)
func GetWorkshopParticipantToken(ctx context.Context, participantUserID uuid.UUID, requestingUserID uuid.UUID) (string, error) {
	// Check permission using centralized function (validates participant and checks access)
	if err := canAccessWorkshopParticipantTokens(ctx, requestingUserID, uuid.Nil, &participantUserID); err != nil {
		return "", err
	}

	// Get the raw user record to access participant_token field
	userRecord, err := queries().GetUserByID(ctx, participantUserID)
	if err != nil {
		return "", obj.ErrNotFound("user not found")
	}

	if !userRecord.ParticipantToken.Valid || userRecord.ParticipantToken.String == "" {
		return "", obj.ErrNotFound("participant has no access token")
	}

	return userRecord.ParticipantToken.String, nil
}
