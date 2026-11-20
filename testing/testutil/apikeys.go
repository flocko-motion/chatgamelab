// package: testutil / the words for API keys
// type:    logic
// job:     adding, sharing, choosing and deleting API keys, and the free-use keys of institutions
// limits:  key words only; the system's free-use key lives with the system settings in orgs.go
package testutil

import (
	"cgl/apiclient"

	"github.com/google/uuid"
)

// AddApiKey adds a key of the user's own.
func (c caller) AddApiKey(apiKey, name, platform string) (ApiKeyShare, error) {
	return decode[ApiKeyShare](c.api.PostApiApikeysNewWithResponse(ctx(), apiclient.CreateApiKeyReq{
		Name:     name,
		Platform: platform,
		Key:      apiKey,
	}))
}

// GetApiKeys is the caller's keys and the shares they can use.
func (c caller) GetApiKeys() (ApiKeys, error) {
	return decode[ApiKeys](c.api.GetApiApikeysWithResponse(ctx()))
}

// GetApiKeyInfo is what the caller may know about a key share.
func (c caller) GetApiKeyInfo(shareID string) (ApiKeyInfo, error) {
	return decode[ApiKeyInfo](c.api.GetApiApikeysIdWithResponse(ctx(), shareID))
}

// RenameApiKey renames a key.
func (c caller) RenameApiKey(shareID, name string) (ApiKeyShare, error) {
	return decode[ApiKeyShare](c.api.PatchApiApikeysIdWithResponse(ctx(), shareID, apiclient.UpdateApiKeyReq{Name: &name}))
}

// DeleteApiKey deletes a key share; with cascade, the owner deletes the key
// and every share of it.
func (c caller) DeleteApiKey(shareID string, cascade bool) error {
	if cascade {
		return done(c.api.DeleteApiApikeysIdWithResponse(ctx(), shareID, query("cascade", "true")))
	}
	return done(c.api.DeleteApiApikeysIdWithResponse(ctx(), shareID))
}

// SetDefaultApiKey makes a key share the caller's default.
func (c caller) SetDefaultApiKey(shareID string) (ApiKeyShare, error) {
	return decode[ApiKeyShare](c.api.PutApiApikeysIdDefaultWithResponse(ctx(), shareID))
}

// ShareApiKeyWithInstitution shares a key with an institution.
func (c caller) ShareApiKeyWithInstitution(shareID, institutionID string) (ApiKeyShare, error) {
	id, err := uuid.Parse(institutionID)
	if err != nil {
		return ApiKeyShare{}, err
	}
	return decode[ApiKeyShare](c.api.PostApiApikeysIdSharesWithResponse(ctx(), shareID, apiclient.ShareApiKeyReq{InstitutionId: &id}))
}

// ShareApiKeyWithWorkshop shares a key with a workshop.
func (c caller) ShareApiKeyWithWorkshop(shareID, workshopID string) (ApiKeyShare, error) {
	id, err := uuid.Parse(workshopID)
	if err != nil {
		return ApiKeyShare{}, err
	}
	return decode[ApiKeyShare](c.api.PostApiApikeysIdSharesWithResponse(ctx(), shareID, apiclient.ShareApiKeyReq{WorkshopId: &id}))
}

// ShareApiKeyWithUser shares a key with another account.
func (c caller) ShareApiKeyWithUser(shareID, userID string) (ApiKeyShare, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return ApiKeyShare{}, err
	}
	return decode[ApiKeyShare](c.api.PostApiApikeysIdSharesWithResponse(ctx(), shareID, apiclient.ShareApiKeyReq{UserId: &id}))
}

// GetApiKeyGameShares is the game shares a key sponsors; context narrows them
// ("" for all).
func (c caller) GetApiKeyGameShares(shareID, context string) ([]ApiKeyGameShare, error) {
	if context != "" {
		return decode[[]ApiKeyGameShare](c.api.GetApiApikeysIdGameSharesWithResponse(ctx(), shareID, query("context", context)))
	}
	return decode[[]ApiKeyGameShare](c.api.GetApiApikeysIdGameSharesWithResponse(ctx(), shareID))
}

// GetInstitutionApiKeys is the keys shared with an institution.
func (c caller) GetInstitutionApiKeys(institutionID string) ([]ApiKeyShare, error) {
	return decode[[]ApiKeyShare](c.api.GetApiInstitutionsIdApikeysWithResponse(ctx(), institutionID))
}

// SetInstitutionFreeUseApiKey sets the key an institution's people play with
// for free, or clears it with nil.
func (c caller) SetInstitutionFreeUseApiKey(institutionID string, shareID *string) (Institution, error) {
	id, err := parseOptional(shareID)
	if err != nil {
		return Institution{}, err
	}
	return decode[Institution](c.api.PatchApiInstitutionsIdFreeUseKeyWithResponse(ctx(), institutionID, apiclient.SetInstitutionFreeUseKeyReq{ShareId: id}))
}
