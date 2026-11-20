// package: testutil / the words for institutions, workshops and invites
// type:    logic
// job:     creating and running institutions and workshops, their members, invites and participants' tokens, and the system settings
// limits:  organisation words only; API keys live in apikeys.go, games in games.go
package testutil

import (
	"cgl/apiclient"

	"github.com/google/uuid"
)

// --- Institutions ---

// CreateInstitution creates an institution.
func (c caller) CreateInstitution(name string) (Institution, error) {
	return decode[Institution](c.api.PostApiInstitutionsWithResponse(ctx(), apiclient.CreateInstitutionReq{Name: name}))
}

// GetInstitution is one institution.
func (c caller) GetInstitution(institutionID string) (Institution, error) {
	return decode[Institution](c.api.GetApiInstitutionsIdWithResponse(ctx(), institutionID))
}

// ListInstitutions is the institutions the caller sees.
func (c caller) ListInstitutions() ([]Institution, error) {
	return decode[[]Institution](c.api.GetApiInstitutionsWithResponse(ctx()))
}

// GetInstitutions is the institutions the caller sees.
func (c caller) GetInstitutions() ([]Institution, error) {
	return c.ListInstitutions()
}

// RenameInstitution renames an institution.
func (c caller) RenameInstitution(institutionID, name string) (Institution, error) {
	return decode[Institution](c.api.PatchApiInstitutionsIdWithResponse(ctx(), institutionID, apiclient.UpdateInstitutionReq{Name: name}))
}

// DeleteInstitution deletes an institution.
func (c caller) DeleteInstitution(institutionID string) error {
	return done(c.api.DeleteApiInstitutionsIdWithResponse(ctx(), institutionID))
}

// GetInstitutionMembers is an institution's members.
func (c caller) GetInstitutionMembers(institutionID string) ([]User, error) {
	return decode[[]User](c.api.GetApiInstitutionsIdMembersWithResponse(ctx(), institutionID))
}

// RemoveMember removes a member from an institution.
func (c caller) RemoveMember(institutionID, userID string) error {
	return done(c.api.DeleteApiInstitutionsIdMembersUserIDWithResponse(ctx(), institutionID, userID))
}

// --- Invites ---

// InviteToInstitution invites an account into an institution with a role.
func (c caller) InviteToInstitution(institutionID, role, invitedUserID string) (Invite, error) {
	return decode[Invite](c.api.PostApiInvitesInstitutionWithResponse(ctx(), apiclient.CreateInstitutionInviteReq{
		InstitutionId: institutionID,
		Role:          role,
		InvitedUserId: &invitedUserID,
	}))
}

// InviteToInstitutionByEmail invites an email address into an institution
// with a role.
func (c caller) InviteToInstitutionByEmail(institutionID, role, invitedEmail string) (Invite, error) {
	return decode[Invite](c.api.PostApiInvitesInstitutionWithResponse(ctx(), apiclient.CreateInstitutionInviteReq{
		InstitutionId: institutionID,
		Role:          role,
		InvitedEmail:  &invitedEmail,
	}))
}

// CreateWorkshopInvite creates a workshop's open invite link for
// participants.
func (c caller) CreateWorkshopInvite(workshopID string) (Invite, error) {
	return c.CreateWorkshopInviteWith(WorkshopInvite{WorkshopId: workshopID})
}

// CreateWorkshopInviteWith creates a workshop invite link with a use limit
// or an expiry.
func (c caller) CreateWorkshopInviteWith(invite WorkshopInvite) (Invite, error) {
	return decode[Invite](c.api.PostApiInvitesWorkshopWithResponse(ctx(), invite))
}

// InviteToWorkshopByEmail invites an email address into a workshop.
func (c caller) InviteToWorkshopByEmail(workshopID, email string) (InviteDetails, error) {
	return decode[InviteDetails](c.api.PostApiInvitesWorkshopEmailWithResponse(ctx(), apiclient.CreateWorkshopEmailInviteReq{
		WorkshopId: workshopID,
		Email:      email,
	}))
}

// GetInvitesIncoming is the invites addressed to the caller.
func (c caller) GetInvitesIncoming() ([]InviteDetails, error) {
	return decode[[]InviteDetails](c.api.GetApiInvitesWithResponse(ctx()))
}

// GetAllInvites is every invite, for an admin.
func (c caller) GetAllInvites() ([]InviteDetails, error) {
	return decode[[]InviteDetails](c.api.GetApiInvitesAllWithResponse(ctx()))
}

// GetInvitesOutgoing is an institution's invites.
func (c caller) GetInvitesOutgoing(institutionID string) ([]InviteDetails, error) {
	return decode[[]InviteDetails](c.api.GetApiInvitesInstitutionInstitutionIdWithResponse(ctx(), institutionID))
}

// GetInvite is one invite, by its id or its token.
func (c caller) GetInvite(idOrToken string) (InviteDetails, error) {
	return decode[InviteDetails](c.api.GetApiInvitesIdOrTokenWithResponse(ctx(), idOrToken))
}

// AcceptInvite accepts an invite by its id and answers it as it is now.
func (c caller) AcceptInvite(inviteID string) (InviteDetails, error) {
	if _, err := c.AcceptInviteByToken(inviteID); err != nil {
		return InviteDetails{}, err
	}
	return c.GetInvite(inviteID)
}

// AcceptInviteByToken accepts an invite by its id or its token.
func (c caller) AcceptInviteByToken(idOrToken string) (AcceptedInvite, error) {
	return decode[AcceptedInvite](c.api.PostApiInvitesIdOrTokenAcceptWithResponse(ctx(), idOrToken))
}

// AcceptWorkshopInviteByToken accepts a workshop invite by its token.
func (c caller) AcceptWorkshopInviteByToken(token string) error {
	_, err := c.AcceptInviteByToken(token)
	return err
}

// DeclineInvite declines an invite and answers it as it is now.
func (c caller) DeclineInvite(inviteID string) (Invite, error) {
	return decode[Invite](c.api.PostApiInvitesIdDeclineWithResponse(ctx(), inviteID))
}

// RevokeInvite withdraws an invite.
func (c caller) RevokeInvite(inviteID string) error {
	return done(c.api.DeleteApiInvitesIdWithResponse(ctx(), inviteID))
}

// --- Workshops ---

// CreateWorkshop creates an active, non-public workshop in an institution.
func (c caller) CreateWorkshop(institutionID, name string) (Workshop, error) {
	return decode[Workshop](c.api.PostApiWorkshopsWithResponse(ctx(), apiclient.CreateWorkshopReq{
		InstitutionId: &institutionID,
		Name:          name,
		Active:        true,
	}))
}

// GetWorkshop is one workshop.
func (c caller) GetWorkshop(workshopID string) (Workshop, error) {
	return decode[Workshop](c.api.GetApiWorkshopsIdWithResponse(ctx(), workshopID))
}

// ListWorkshops is an institution's workshops the caller sees.
func (c caller) ListWorkshops(institutionID string) ([]Workshop, error) {
	return decode[[]Workshop](c.api.GetApiWorkshopsWithResponse(ctx(), query("institutionId", institutionID)))
}

// ListAllWorkshops is every workshop the caller sees.
func (c caller) ListAllWorkshops() ([]Workshop, error) {
	return decode[[]Workshop](c.api.GetApiWorkshopsWithResponse(ctx()))
}

// UpdateWorkshop replaces a workshop's settings. A setting left out is
// false or empty; the public page's slug, text and links left out (nil) keep
// what they are.
func (c caller) UpdateWorkshop(workshopID string, update WorkshopUpdate) (Workshop, error) {
	return decode[Workshop](c.api.PatchApiWorkshopsIdWithResponse(ctx(), workshopID, update))
}

// EditWorkshop reads a workshop, lets edit change its settings, and saves
// them: the settings edit leaves alone stay as they were.
func (c caller) EditWorkshop(workshopID string, edit func(*WorkshopUpdate)) (Workshop, error) {
	w, err := c.GetWorkshop(workshopID)
	if err != nil {
		return Workshop{}, err
	}
	update := WorkshopUpdate{
		Name:                       w.Name,
		Active:                     w.Active,
		Public:                     w.Public,
		AiQualityTier:              w.AiQualityTier,
		PromptConstraints:          w.PromptConstraints,
		ShowPublicGames:            w.ShowPublicGames,
		ShowOtherParticipantsGames: w.ShowOtherParticipantsGames,
		IsPaused:                   w.IsPaused,
		AllowGameSharing:           w.AllowGameSharing,
	}
	edit(&update)
	return c.UpdateWorkshop(workshopID, update)
}

// DeleteWorkshop deletes a workshop.
func (c caller) DeleteWorkshop(workshopID string) error {
	return done(c.api.DeleteApiWorkshopsIdWithResponse(ctx(), workshopID))
}

// SetWorkshopApiKey sets the key a workshop plays with, or clears it with nil.
func (c caller) SetWorkshopApiKey(workshopID string, apiKeyShareID *string) (Workshop, error) {
	return decode[Workshop](c.api.PutApiWorkshopsIdApiKeyWithResponse(ctx(), workshopID, apiclient.SetWorkshopApiKeyReq{ApiKeyShareId: apiKeyShareID}))
}

// AddMemberToWorkshop adds an institution member to a workshop.
func (c caller) AddMemberToWorkshop(workshopID, userID string) error {
	return done(c.api.PostApiWorkshopsIdMembersWithResponse(ctx(), workshopID, apiclient.AddWorkshopMemberReq{UserId: userID}))
}

// RemoveMemberFromWorkshop removes a member from a workshop.
func (c caller) RemoveMemberFromWorkshop(workshopID, userID string) error {
	return done(c.api.DeleteApiWorkshopsIdMembersUserIdWithResponse(ctx(), workshopID, userID))
}

// SetActiveWorkshop sets the workshop the caller works in, or clears it with nil.
func (c caller) SetActiveWorkshop(workshopID *string) (User, error) {
	var id *uuid.UUID
	if workshopID != nil {
		parsed, err := uuid.Parse(*workshopID)
		if err != nil {
			return User{}, err
		}
		id = &parsed
	}
	return decode[User](c.api.PutApiUsersMeActiveWorkshopWithResponse(ctx(), apiclient.SetActiveWorkshopReq{WorkshopId: id}))
}

// GetParticipantToken is a participant's token, for a workshop's staff.
func (c caller) GetParticipantToken(participantID string) (string, error) {
	resp, err := decode[apiclient.WorkshopParticipantTokenResp](c.api.GetApiWorkshopsParticipantsParticipantIdTokenWithResponse(ctx(), participantID))
	return resp.Token, err
}

// ResetParticipantToken gives a participant a new token, ending the old one,
// and answers the new one.
func (c caller) ResetParticipantToken(participantID string) (string, error) {
	resp, err := decode[apiclient.WorkshopParticipantTokenResp](c.api.PostApiWorkshopsParticipantsParticipantIdTokenResetWithResponse(ctx(), participantID))
	return resp.Token, err
}

// --- System settings ---

// GetSystemSettings is the platform's settings.
func (c caller) GetSystemSettings() (SystemSettings, error) {
	return decode[SystemSettings](c.api.GetApiSystemSettingsWithResponse(ctx()))
}

// UpdateSystemSettings changes the platform's settings; a setting left out
// (nil) stays.
func (c caller) UpdateSystemSettings(update SettingsUpdate) (SystemSettings, error) {
	return decode[SystemSettings](c.api.PatchApiSystemSettingsWithResponse(ctx(), update))
}

// SetSystemFreeUseApiKey sets the key everyone plays with for free, or
// clears it with nil.
func (c caller) SetSystemFreeUseApiKey(apiKeyID *string) (SystemSettings, error) {
	id, err := parseOptional(apiKeyID)
	if err != nil {
		return SystemSettings{}, err
	}
	return decode[SystemSettings](c.api.PatchApiSystemSettingsFreeUseKeyWithResponse(ctx(), apiclient.SetSystemFreeUseApiKeyReq{ApiKeyId: id}))
}

// parseOptional parses an optional id; nil stays nil.
func parseOptional(id *string) (*uuid.UUID, error) {
	if id == nil {
		return nil, nil
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// MembersOf is the members an institution came with; none when it came
// without its members.
func MembersOf(inst Institution) []InstitutionMember {
	if inst.Members == nil {
		return nil
	}
	return *inst.Members
}

// WorkshopInvitesOf is the invites a workshop came with; none when the
// caller may not see them.
func WorkshopInvitesOf(w Workshop) []Invite {
	if w.Invites == nil {
		return nil
	}
	return *w.Invites
}

// WorkshopParticipantsOf is the members a workshop came with; none when the
// caller may not see them.
func WorkshopParticipantsOf(w Workshop) []WorkshopParticipant {
	if w.Participants == nil {
		return nil
	}
	return *w.Participants
}
