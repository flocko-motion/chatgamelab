// package: db / database access and repository layer
// type:    data
// job:     what v1 code still reads and writes of workshops: reading one for v1 callers, its key (until #451), the delete cascade; the rest is the core's (core/workshop*.go).
// limits:  does not list workshops or fetch member/invite collections (-> workshop_list.go).
package db

import (
	"cgl/log"
	"cgl/obj"
	"context"

	"github.com/google/uuid"
)

// GetWorkshopName retrieves just the workshop name by ID (no permission check, for display only)
func GetWorkshopName(ctx context.Context, id uuid.UUID) (string, error) {
	result, err := queries().GetWorkshopByID(ctx, id)
	if err != nil {
		return "", obj.ErrNotFound("workshop not found")
	}
	return result.Name, nil
}

// WorkshopSummary is a lightweight struct with just ID and Name.
type WorkshopSummary struct {
	ID   uuid.UUID
	Name string
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
	var inviteRows []UserRoleInvite
	if err := canAccessWorkshopInvites(ctx, userID, result.InstitutionID); err == nil {
		inviteRows, err = queries().GetInvitesByWorkshop(ctx, uuid.NullUUID{UUID: id, Valid: true})
		if err != nil {
			// Don't fail if we can't get invites, just return empty list
			inviteRows = []UserRoleInvite{}
		}
	} else {
		// User doesn't have permission to view invites, return empty list
		inviteRows = []UserRoleInvite{}
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
		participantRows, err := queries().GetWorkshopParticipants(ctx, uuid.NullUUID{UUID: id, Valid: true})
		if err != nil {
			// Don't fail if we can't get participants, just return empty list
			log.Warn("failed to get workshop participants", "workshop_id", id, "error", err)
			participantRows = []GetWorkshopParticipantsRow{}
		}

		// Convert participants to obj.WorkshopParticipant
		participants = make([]obj.WorkshopParticipant, 0, len(participantRows))
		for _, p := range participantRows {
			// Parse role from database
			role, err := stringToRole(p.Role)
			if err != nil {
				// Default to Guest if role parsing fails
				role = obj.RoleGuest
			}

			participant := obj.WorkshopParticipant{
				ID:          p.ID,
				WorkshopID:  id,
				Name:        p.Name.String,
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

	var aiQualityTier *string
	if result.AiQualityTier.Valid {
		aiQualityTier = &result.AiQualityTier.String
	}

	var promptConstraints *string
	if result.PromptConstraints.Valid {
		promptConstraints = &result.PromptConstraints.String
	}

	institution := &obj.Institution{ID: result.InstitutionID}
	if instName, err := GetInstitutionName(ctx, result.InstitutionID); err == nil {
		institution.Name = instName
	}

	return &obj.Workshop{
		ID:                         result.ID,
		Name:                       result.Name,
		Institution:                institution,
		Active:                     result.Active,
		Public:                     result.Public,
		DefaultApiKeyShareID:       defaultApiKeyShareID,
		Participants:               participants,
		Invites:                    invites,
		AiQualityTier:              aiQualityTier,
		PromptConstraints:          promptConstraints,
		ShowPublicGames:            result.ShowPublicGames,
		ShowOtherParticipantsGames: result.ShowOtherParticipantsGames,
		IsPaused:                   result.IsPaused,
		AllowGameSharing:           result.AllowGameSharing,
		PublicSlug:                 nullStringToPtr(result.PublicSlug),
		PublicDescription:          nullStringToPtr(result.PublicDescription),
		PublicLinks:                UnmarshalPublicLinks(result.PublicLinks),
		Meta: obj.Meta{
			CreatedBy:  result.CreatedBy,
			CreatedAt:  &result.CreatedAt,
			ModifiedBy: result.ModifiedBy,
			ModifiedAt: &result.ModifiedAt,
		},
	}, nil
}

// DeleteWorkshopCascade deletes a workshop with its guests, for the core,
// which has checked that the caller may (core.DeleteWorkshop): v1's cascade
// until guests and share links are ported. Its other games stay with their
// creators.
func DeleteWorkshopCascade(ctx context.Context, id uuid.UUID) error {
	wsNullUUID := uuid.NullUUID{UUID: id, Valid: true}

	// Clean up Guests: collect user IDs, delete roles and user accounts
	guestUserIDs, _ := queries().GetGuestUserIDsByWorkshopID(ctx, wsNullUUID)
	_ = queries().DeleteUserRolesByWorkshopID(ctx, wsNullUUID)
	for _, uid := range guestUserIDs {
		// Delete each Guest's account
		_ = DeleteUser(ctx, uid)
	}

	DeleteWorkshopPublicPageShares(ctx, id)

	// Unlink remaining games from this workshop (member games stay with their creator)
	_ = queries().UnlinkGamesFromWorkshop(ctx, wsNullUUID)

	if err := queries().DeleteWorkshop(ctx, id); err != nil {
		return obj.ErrServerError("failed to delete workshop")
	}
	return nil
}
