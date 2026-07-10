// package: testutil / org & account API helpers
// type:    logic
// job:     high-level UserClient helpers for invites, institutions, workshops, users and API keys
// limits:  account/org HTTP helpers only; no game/session (-> testclient_games) or assertions (-> testclient_assertions)
package testutil

import (
	"fmt"

	"cgl/api/routes"
	"cgl/obj"

	"github.com/google/uuid"
)

// GetInvitesIncoming returns the user's incoming invites (composable high-level API)
func (u *UserClient) GetInvitesIncoming() ([]obj.UserRoleInvite, error) {
	u.t.Helper()
	var invites []obj.UserRoleInvite
	err := u.Get("invites", &invites)
	return invites, err
}

// GetInvitesIncomingDetailed returns the user's incoming invites with full details (workshop/institution names)
func (u *UserClient) GetInvitesIncomingDetailed() ([]routes.InviteResponse, error) {
	u.t.Helper()
	var invites []routes.InviteResponse
	err := u.Get("invites", &invites)
	return invites, err
}

// GetInvitesOutgoing returns all invites for a specific institution (composable high-level API)
func (u *UserClient) GetInvitesOutgoing(institutionID string) ([]obj.UserRoleInvite, error) {
	u.t.Helper()
	var invites []obj.UserRoleInvite
	err := u.Get("invites/institution/"+institutionID, &invites)
	return invites, err
}

// GetInvite returns a specific invite by ID (composable high-level API)
func (u *UserClient) GetInvite(inviteID string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	var invite obj.UserRoleInvite
	err := u.Get("invites/"+inviteID, &invite)
	return invite, err
}

// GetInstitutions returns the user's institutions (composable high-level API)
func (u *UserClient) GetInstitutions() ([]obj.Institution, error) {
	u.t.Helper()
	var institutions []obj.Institution
	err := u.Get("institutions", &institutions)
	return institutions, err
}

// AcceptInvite accepts an invite by ID (composable high-level API)
func (u *UserClient) AcceptInvite(inviteID string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	if err := u.Post("invites/"+inviteID+"/accept", nil, nil); err != nil {
		return obj.UserRoleInvite{}, err
	}
	return u.GetInvite(inviteID)
}

// DeclineInvite declines an invite by ID (composable high-level API)
func (u *UserClient) DeclineInvite(inviteID string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	if err := u.Post("invites/"+inviteID+"/decline", nil, nil); err != nil {
		return obj.UserRoleInvite{}, err
	}
	return u.GetInvite(inviteID)
}

// RevokeInvite revokes an invite by ID (composable high-level API)
func (u *UserClient) RevokeInvite(inviteID string) error {
	u.t.Helper()
	return u.Delete("invites/" + inviteID)
}

// CreateInstitution creates an institution (composable high-level API)
func (u *UserClient) CreateInstitution(name string) (obj.Institution, error) {
	u.t.Helper()
	var result obj.Institution
	payload := routes.CreateInstitutionRequest{
		Name: name,
	}
	err := u.Post("institutions", payload, &result)
	return result, err
}

// InviteToInstitution creates an institution invite by user ID (composable high-level API)
func (u *UserClient) InviteToInstitution(institutionID, role, invitedUserID string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	var result obj.UserRoleInvite
	payload := routes.CreateInstitutionInviteRequest{
		InstitutionID: institutionID,
		Role:          role,
		InvitedUserID: &invitedUserID,
	}
	err := u.Post("invites/institution", payload, &result)
	return result, err
}

// InviteToInstitutionByEmail creates an institution invite by email (composable high-level API)
func (u *UserClient) InviteToInstitutionByEmail(institutionID, role, invitedEmail string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	var result obj.UserRoleInvite
	payload := routes.CreateInstitutionInviteRequest{
		InstitutionID: institutionID,
		Role:          role,
		InvitedEmail:  &invitedEmail,
	}
	err := u.Post("invites/institution", payload, &result)
	return result, err
}

// GetRole returns the current user's role name, failing the test on error.
func (u *UserClient) GetRole() string {
	u.t.Helper()
	me, err := u.GetMe()
	if err != nil {
		u.t.Fatalf("User %q: failed to get me: %v", u.Name, err)
	}
	if me.Role == nil {
		u.t.Fatalf("User %q: no role", u.Name)
	}
	return string(me.Role.Role)
}

// GetMe returns the current user's profile (composable high-level API)
func (u *UserClient) GetMe() (obj.User, error) {
	u.t.Helper()
	var result obj.User
	err := u.Get("users/me", &result)
	return result, err
}

// UpdateUserName updates a user's name by ID (composable high-level API)
func (u *UserClient) UpdateUserName(userID string, name string) (obj.User, error) {
	u.t.Helper()
	var result obj.User
	payload := map[string]string{"name": name}
	err := u.Post("users/"+userID, payload, &result)
	return result, err
}

// SetUserLanguage sets the user's language preference (composable high-level API)
func (u *UserClient) SetUserLanguage(language string) error {
	u.t.Helper()
	return u.Patch("users/me/language", map[string]string{"language": language}, nil)
}

// GetInstitution returns a specific institution by ID (composable high-level API)
func (u *UserClient) GetInstitution(institutionID string) (obj.Institution, error) {
	u.t.Helper()
	var result obj.Institution
	err := u.Get("institutions/"+institutionID, &result)
	return result, err
}

// GetUsers returns all users (composable high-level API)
func (u *UserClient) GetUsers() ([]obj.User, error) {
	u.t.Helper()
	var result []obj.User
	err := u.Get("users", &result)
	return result, err
}

// RemoveMember removes a member from an institution (composable high-level API)
func (u *UserClient) RemoveMember(institutionID, userID string) error {
	u.t.Helper()
	return u.Delete("institutions/" + institutionID + "/members/" + userID)
}

// CreateWorkshop creates a new workshop (composable high-level API)
func (u *UserClient) CreateWorkshop(institutionID, name string) (obj.Workshop, error) {
	u.t.Helper()
	payload := map[string]interface{}{
		"institutionId": institutionID,
		"name":          name,
		"active":        true,
		"public":        false,
	}
	var result obj.Workshop
	err := u.Post("workshops", payload, &result)
	return result, err
}

// UpdateWorkshop updates a workshop (composable high-level API)
func (u *UserClient) UpdateWorkshop(workshopID string, updates map[string]interface{}) (obj.Workshop, error) {
	u.t.Helper()
	var result obj.Workshop
	err := u.Patch("workshops/"+workshopID, updates, &result)
	return result, err
}

// GetWorkshop retrieves a workshop by ID (composable high-level API)
func (u *UserClient) GetWorkshop(workshopID string) (obj.Workshop, error) {
	u.t.Helper()
	var result obj.Workshop
	err := u.Get("workshops/"+workshopID, &result)
	return result, err
}

// ListWorkshops lists workshops for an institution (composable high-level API)
func (u *UserClient) ListWorkshops(institutionID string) ([]obj.Workshop, error) {
	u.t.Helper()
	var result []obj.Workshop
	err := u.Get("workshops?institutionId="+institutionID, &result)
	return result, err
}

// GetParticipantToken retrieves the access token for a workshop participant (composable high-level API)
func (u *UserClient) GetParticipantToken(participantID string) (*string, error) {
	u.t.Helper()
	var result map[string]string
	err := u.Get("workshops/participants/"+participantID+"/token", &result)
	if err != nil {
		return nil, err
	}
	token := result["token"]
	return &token, nil
}

// CreateWorkshopInvite creates a workshop invite (composable high-level API)
func (u *UserClient) CreateWorkshopInvite(workshopID, role string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	payload := map[string]interface{}{
		"workshopId": workshopID,
		"role":       role,
	}
	var result obj.UserRoleInvite
	err := u.Post("invites/workshop", payload, &result)
	return result, err
}

// AcceptWorkshopInviteByToken accepts a workshop invite by token as an authenticated user (composable high-level API)
// For individuals/head/staff this enters workshop mode; for participants this switches workshops.
func (u *UserClient) AcceptWorkshopInviteByToken(token string) error {
	u.t.Helper()
	return u.Post("invites/"+token+"/accept", nil, nil)
}

// AddApiKey reads an API key from a file and creates it (composable high-level API)
func (u *UserClient) AddApiKey(apiKey, name, platform string) (obj.ApiKeyShare, error) {
	u.t.Helper()

	var result obj.ApiKeyShare
	err := u.Post("apikeys/new", routes.CreateApiKeyRequest{
		Name:     name,
		Platform: platform,
		Key:      apiKey,
	}, &result)
	return result, err
}

// RemoveUserRole removes a user's role (composable high-level API)
func (u *UserClient) RemoveUserRole(userID string) error {
	u.t.Helper()
	return u.Delete("users/" + userID + "/role")
}

// GetSystemSettings returns the global system settings (composable high-level API)
func (u *UserClient) GetSystemSettings() (obj.SystemSettings, error) {
	u.t.Helper()
	var result obj.SystemSettings
	err := u.Get("system/settings", &result)
	return result, err
}

// SetSystemFreeUseApiKey sets or clears the global free-use API key (composable high-level API)
// Pass nil to clear the free-use key.
func (u *UserClient) SetSystemFreeUseApiKey(apiKeyID *string) (obj.SystemSettings, error) {
	u.t.Helper()
	var payload interface{}
	if apiKeyID != nil {
		id, err := uuid.Parse(*apiKeyID)
		if err != nil {
			return obj.SystemSettings{}, fmt.Errorf("invalid apiKeyID: %w", err)
		}
		payload = routes.SetFreeUseApiKeyRequest{ApiKeyID: &id}
	} else {
		payload = routes.SetFreeUseApiKeyRequest{ApiKeyID: nil}
	}
	var result obj.SystemSettings
	err := u.Patch("system/settings/free-use-key", payload, &result)
	return result, err
}

// DeleteApiKey deletes an API key share, optionally cascading to delete the key and all shares (composable high-level API)
func (u *UserClient) DeleteApiKey(shareID string, cascade bool) error {
	u.t.Helper()
	endpoint := "apikeys/" + shareID
	if cascade {
		endpoint += "?cascade=true"
	}
	return u.Delete(endpoint)
}

// GetApiKeys returns the user's API keys and all their linked shares (composable high-level API)
func (u *UserClient) GetApiKeys() (routes.ApiKeysResponse, error) {
	u.t.Helper()
	var result routes.ApiKeysResponse
	err := u.Get("apikeys", &result)
	return result, err
}

// SetDefaultApiKey sets the given API key share as the user's default (composable high-level API)
func (u *UserClient) SetDefaultApiKey(shareID string) (obj.ApiKeyShare, error) {
	u.t.Helper()
	var result obj.ApiKeyShare
	err := u.Put("apikeys/"+shareID+"/default", nil, &result)
	return result, err
}

// SetInstitutionFreeUseApiKey sets or clears the free-use API key share for an institution (composable high-level API)
// Pass nil to clear.
func (u *UserClient) SetInstitutionFreeUseApiKey(institutionID string, shareID *string) (obj.Institution, error) {
	u.t.Helper()
	var sid *uuid.UUID
	if shareID != nil {
		parsed, err := uuid.Parse(*shareID)
		if err != nil {
			return obj.Institution{}, fmt.Errorf("invalid shareID: %w", err)
		}
		sid = &parsed
	}
	var result obj.Institution
	err := u.Patch("institutions/"+institutionID+"/free-use-key", map[string]interface{}{
		"shareId": sid,
	}, &result)
	return result, err
}

// ShareApiKeyWithInstitution shares an API key with an institution (composable high-level API)
func (u *UserClient) ShareApiKeyWithInstitution(shareID string, institutionID string) (obj.ApiKeyShare, error) {
	u.t.Helper()
	instID, err := uuid.Parse(institutionID)
	if err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("invalid institutionID: %w", err)
	}
	var result obj.ApiKeyShare
	err = u.Post("apikeys/"+shareID+"/shares", routes.ShareRequest{
		InstitutionID: &instID,
	}, &result)
	return result, err
}

// SetWorkshopApiKey sets (or clears) the default API key for a workshop (composable high-level API)
func (u *UserClient) SetWorkshopApiKey(workshopID string, apiKeyShareID *string) (obj.Workshop, error) {
	u.t.Helper()
	var result obj.Workshop
	err := u.Put("workshops/"+workshopID+"/api-key", routes.SetWorkshopApiKeyRequest{
		ApiKeyShareID: apiKeyShareID,
	}, &result)
	return result, err
}

// SetActiveWorkshop sets the user's active workshop context (composable high-level API)
// Pass nil to leave workshop mode.
func (u *UserClient) SetActiveWorkshop(workshopID *string) (obj.User, error) {
	u.t.Helper()
	var wsID *uuid.UUID
	if workshopID != nil {
		parsed, err := uuid.Parse(*workshopID)
		if err != nil {
			return obj.User{}, fmt.Errorf("invalid workshopID: %w", err)
		}
		wsID = &parsed
	}
	var result obj.User
	err := u.Put("users/me/active-workshop", map[string]interface{}{
		"workshopId": wsID,
	}, &result)
	return result, err
}

// SetUserAiQualityTier sets the user's AI quality tier (composable high-level API)
// Valid tiers: "low", "medium", "high", "max"
func (u *UserClient) SetUserAiQualityTier(tier string) error {
	u.t.Helper()
	return u.Post("users/"+u.ID, map[string]interface{}{
		"aiQualityTier": tier,
	}, nil)
}

// ShareApiKeyWithWorkshop shares an API key with a workshop (composable high-level API)
func (u *UserClient) ShareApiKeyWithWorkshop(shareID string, workshopID string) (obj.ApiKeyShare, error) {
	u.t.Helper()
	wsID, err := uuid.Parse(workshopID)
	if err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("invalid workshopID: %w", err)
	}
	var result obj.ApiKeyShare
	err = u.Post("apikeys/"+shareID+"/shares", routes.ShareRequest{
		WorkshopID: &wsID,
	}, &result)
	return result, err
}

// ListInstitutions returns all institutions visible to the user (composable high-level API)
func (u *UserClient) ListInstitutions() ([]obj.Institution, error) {
	u.t.Helper()
	var result []obj.Institution
	err := u.Get("institutions", &result)
	return result, err
}

// DeleteInstitution deletes an institution by ID (composable high-level API)
func (u *UserClient) DeleteInstitution(institutionID string) error {
	u.t.Helper()
	return u.Delete("institutions/" + institutionID)
}

// DeleteWorkshop deletes a workshop by ID (composable high-level API)
func (u *UserClient) DeleteWorkshop(workshopID string) error {
	u.t.Helper()
	return u.Delete("workshops/" + workshopID)
}

// GetInstitutionApiKeys returns API keys shared with an institution (composable high-level API)
func (u *UserClient) GetInstitutionApiKeys(institutionID string) ([]obj.ApiKeyShare, error) {
	u.t.Helper()
	var result []obj.ApiKeyShare
	err := u.Get("institutions/"+institutionID+"/apikeys", &result)
	return result, err
}

// ReactivateInvite reactivates a revoked invite (composable high-level API)
func (u *UserClient) ReactivateInvite(inviteID string) error {
	u.t.Helper()
	return u.Post("invites/"+inviteID+"/reactivate", nil, nil)
}

// InviteToWorkshopByEmail creates a targeted workshop invite by email
func (u *UserClient) InviteToWorkshopByEmail(workshopID, email string) (obj.UserRoleInvite, error) {
	u.t.Helper()
	var result obj.UserRoleInvite
	err := u.Post("invites/workshop/email", routes.CreateWorkshopEmailInviteRequest{
		WorkshopID: workshopID,
		Email:      email,
	}, &result)
	return result, err
}

// AddMemberToWorkshop directly adds an org individual to a workshop
func (u *UserClient) AddMemberToWorkshop(workshopID, userID string) error {
	u.t.Helper()
	return u.Post("workshops/"+workshopID+"/members", routes.AddMemberToWorkshopRequest{
		UserID: userID,
	}, nil)
}

// GetApiKeyGameShares returns game shares for a specific API key share.
// context: "personal" (all, owner only), "organization" (org/workshop only), or "" (no filter).
func (u *UserClient) GetApiKeyGameShares(shareID string, context string) ([]routes.EnrichedGameShare, error) {
	u.t.Helper()
	var result []routes.EnrichedGameShare
	path := "apikeys/" + shareID + "/game-shares"
	if context != "" {
		path += "?context=" + context
	}
	err := u.Get(path, &result)
	return result, err
}
