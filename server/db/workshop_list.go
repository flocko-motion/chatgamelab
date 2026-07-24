// package: db / database access and repository layer
// type:    data
// job:     list workshops with filters and load their participant and invite collections.
// limits:  does not create or mutate individual workshops (-> workshop.go).
package db

import (
	"cgl/obj"
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ListWorkshopsForInstitution returns a lightweight list of workshops for an institution (no permission checks).
func ListWorkshopsForInstitution(ctx context.Context, institutionID uuid.UUID) ([]WorkshopSummary, error) {
	rows, err := queries().ListWorkshopsByInstitution(ctx, institutionID)
	if err != nil {
		return nil, err
	}
	result := make([]WorkshopSummary, len(rows))
	for i, r := range rows {
		result[i] = WorkshopSummary{ID: r.ID, Name: r.Name}
	}
	return result, nil
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
	participantRows, err := queries().GetWorkshopParticipants(ctx, uuid.NullUUID{UUID: workshopID, Valid: true})
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
			Permanent:   p.Permanent,
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

			var aiQualityTier *string
			if r.AiQualityTier.Valid {
				aiQualityTier = &r.AiQualityTier.String
			}

			var promptConstraints *string
			if r.PromptConstraints.Valid {
				promptConstraints = &r.PromptConstraints.String
			}

			workshop := obj.Workshop{
				ID:                         r.ID,
				Name:                       r.Name,
				Institution:                &obj.Institution{ID: r.InstitutionID},
				Active:                     r.Active,
				Public:                     r.Public,
				DefaultApiKeyShareID:       defaultApiKeyShareID,
				AiQualityTier:              aiQualityTier,
				PromptConstraints:          promptConstraints,
				ShowPublicGames:            r.ShowPublicGames,
				ShowOtherParticipantsGames: r.ShowOtherParticipantsGames,
				DesignEditingEnabled:       r.DesignEditingEnabled,
				IsPaused:                   r.IsPaused,
				AllowGameSharing:           r.AllowGameSharing,
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

			var aiQualityTier *string
			if r.AiQualityTier.Valid {
				aiQualityTier = &r.AiQualityTier.String
			}

			var promptConstraints *string
			if r.PromptConstraints.Valid {
				promptConstraints = &r.PromptConstraints.String
			}

			workshop := obj.Workshop{
				ID:                         r.ID,
				Name:                       r.Name,
				Institution:                &obj.Institution{ID: r.InstitutionID},
				Active:                     r.Active,
				Public:                     r.Public,
				DefaultApiKeyShareID:       defaultApiKeyShareID,
				AiQualityTier:              aiQualityTier,
				PromptConstraints:          promptConstraints,
				ShowPublicGames:            r.ShowPublicGames,
				ShowOtherParticipantsGames: r.ShowOtherParticipantsGames,
				DesignEditingEnabled:       r.DesignEditingEnabled,
				IsPaused:                   r.IsPaused,
				AllowGameSharing:           r.AllowGameSharing,
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
