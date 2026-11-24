package wire

import (
	"cgl/core"

	"github.com/google/uuid"
)

// The pages of the core's lists, converted, passing a loading error through:
//
//	return wire.UserViewPageOf(core.List(ctx, core.Users, q))

func UserViewPageOf(p core.Page[*core.UserCard], err error) (Page[UserView], error) {
	return pageOf(p, err, userView)
}

func OrganisationViewPageOf(p core.Page[*core.OrganisationCard], err error) (Page[OrganisationView], error) {
	return pageOf(p, err, organisationView)
}

func InviteCardViewPageOf(p core.Page[*core.InviteCard], err error) (Page[InviteCardView], error) {
	return pageOf(p, err, inviteCardView)
}

// VerifiedPageViewPageOf converts a page of verified pages.
func VerifiedPageViewPageOf(p core.Page[*core.VerifiedPage], err error) (Page[VerifiedPageView], error) {
	return pageOf(p, err, verifiedPageView)
}

func WorkshopViewPageOf(p core.Page[*core.WorkshopCard], err error) (Page[WorkshopView], error) {
	return pageOf(p, err, workshopView)
}

// WorkshopParticipantViewPageOf converts a page of workshop's participants:
//
//	return wire.WorkshopParticipantViewPageOf(id)(core.List(ctx, core.Participants(id), q))
func WorkshopParticipantViewPageOf(workshop uuid.UUID) func(core.Page[core.WorkshopParticipant], error) (Page[WorkshopParticipantView], error) {
	return func(p core.Page[core.WorkshopParticipant], err error) (Page[WorkshopParticipantView], error) {
		return pageOf(p, err, func(part core.WorkshopParticipant) WorkshopParticipantView {
			return participantView(workshop, part)
		})
	}
}

func KeyShareViewPageOf(p core.Page[*core.KeyShareCard], err error) (Page[KeyShareView], error) {
	return pageOf(p, err, keyShareView)
}

// OwnApiKeyView is one of the caller's keys: the key, their own use of it
// (selfShare: what makes it the default or deletes it) and every share of
// it, the own use included.
type OwnApiKeyView struct {
	Key       ApiKeyView     `json:"key"`
	SelfShare KeyShareView   `json:"selfShare"`
	Shares    []KeyShareView `json:"shares"`
}

func OwnApiKeyViewPageOf(p core.Page[core.OwnKey], err error) (Page[OwnApiKeyView], error) {
	return pageOf(p, err, func(k core.OwnKey) OwnApiKeyView {
		v := OwnApiKeyView{Key: ApiKeyViewOf(k.Self), SelfShare: keyShareView(k.Self), Shares: make([]KeyShareView, len(k.Shares))}
		for i, s := range k.Shares {
			v.Shares[i] = keyShareView(s)
		}
		return v
	})
}
