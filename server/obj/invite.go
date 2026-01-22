package obj

import (
	"time"

	"github.com/google/uuid"
)

// UserRoleInvite represents an invitation for a user to assume a role
type UserRoleInvite struct {
	ID            uuid.UUID    `json:"id"`
	Meta          Meta         `json:"meta"`
	InstitutionID uuid.UUID    `json:"institutionId"`
	Role          Role         `json:"role"`
	WorkshopID    *uuid.UUID   `json:"workshopId,omitempty"`
	InvitedUserID *uuid.UUID   `json:"invitedUserId,omitempty"`
	InvitedEmail  *string      `json:"invitedEmail,omitempty"`
	InviteToken   *string      `json:"inviteToken,omitempty"`
	MaxUses       *int32       `json:"maxUses,omitempty"`
	UsesCount     int32        `json:"usesCount"`
	ExpiresAt     *time.Time   `json:"expiresAt,omitempty"`
	Status        InviteStatus `json:"status"`
	AcceptedAt    *time.Time   `json:"acceptedAt,omitempty"`
	AcceptedBy    *uuid.UUID   `json:"acceptedBy,omitempty"`
}
