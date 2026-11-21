package wire

import "cgl/core"

// InviteCardView is an invite as the caller sees it: the invite, the names
// of its organisation and workshop, and what the caller may do with it.
type InviteCardView struct {
	InviteView
	InstitutionName string  `json:"institutionName"`
	WorkshopName    *string `json:"workshopName,omitempty"`
	// MayRevoke: the caller may revoke it; MayAnswer: accept or decline it.
	MayRevoke bool `json:"mayRevoke"`
	MayAnswer bool `json:"mayAnswer"`
}

// InviteCardViewOf converts one invite, passing a loading error through:
//
//	return wire.InviteCardViewOf(core.ReadInvite(ctx, id))
func InviteCardViewOf(inv *core.InviteCard, err error) (InviteCardView, error) {
	if err != nil {
		return InviteCardView{}, err
	}
	return inviteCardView(inv), nil
}

// InviteCardViewsOf converts a list of invites, passing a loading error
// through.
func InviteCardViewsOf(invs []*core.InviteCard, err error) ([]InviteCardView, error) {
	if err != nil {
		return nil, err
	}
	views := make([]InviteCardView, len(invs))
	for i, inv := range invs {
		views[i] = inviteCardView(inv)
	}
	return views, nil
}

func inviteCardView(inv *core.InviteCard) InviteCardView {
	return InviteCardView{
		InviteView:      inviteView(inv.Invite),
		InstitutionName: inv.OrganisationName,
		WorkshopName:    inv.WorkshopName,
		MayRevoke:       inv.MayRevoke(),
		MayAnswer:       inv.MayAnswer(),
	}
}
