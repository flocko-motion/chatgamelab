// package: api / schemaf endpoints
// type:    logic
// job:     lists, creates, renames, shares and deletes API keys, takes shares back, and sets an organisation's free-use key and a workshop's key
// limits:  decides nothing: the core does (core/key.go, core/share_link.go)
package api

import (
	"context"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// ApiKeyShareResp is a share of a key as the caller sees it.
type ApiKeyShareResp = wire.KeyShareView

// ApiKeyIDReq names a key by one of its shares.
type ApiKeyIDReq struct {
	ID string `path:"id"`
}

// ListApiKeysEndpoint lists the caller's keys and every share of them: their
// own use of each, the organisation they gave it to, and the sponsored games
// and share links it pays for.
type ListApiKeysEndpoint struct{}

func (ListApiKeysEndpoint) Method() string             { return "GET" }
func (ListApiKeysEndpoint) Path() string               { return "/api/apikeys" }
func (ListApiKeysEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListApiKeysEndpoint) Handle(ctx context.Context, req ListApiKeysReq) (OwnApiKeyPageResp, error) {
	return wire.OwnApiKeyViewPageOf(core.List(ctx, core.OwnKeys, req.query()))
}

// OwnApiKeySort orders the caller's keys; the default is createdAt, oldest
// first. status orders working keys first, then those not used yet, then
// failed ones.
type OwnApiKeySort string

const (
	OwnApiKeySortCreatedAt OwnApiKeySort = "createdAt"
	OwnApiKeySortName      OwnApiKeySort = "name"
	OwnApiKeySortPlatform  OwnApiKeySort = "platform"
	OwnApiKeySortStatus    OwnApiKeySort = "status"
)

// ListApiKeysReq filters, sorts and pages the caller's keys.
type ListApiKeysReq struct {
	ListParams[KeyFilter, OwnApiKeySort]
}

// OwnApiKeyPageResp is a page of the caller's keys, each with its shares, and the total.
type OwnApiKeyPageResp = wire.Page[wire.OwnApiKeyView]

// CreateApiKeyEndpoint stores a key of the caller's and answers with their
// own use of it. Platform and key are required; surrounding whitespace is
// trimmed. Without a name the key is named after its platform and today's
// date, e.g. "Mistral 12.02.26". The caller's first key becomes their
// default.
type CreateApiKeyEndpoint struct{}

func (CreateApiKeyEndpoint) Method() string             { return "POST" }
func (CreateApiKeyEndpoint) Path() string               { return "/api/apikeys/new" }
func (CreateApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateApiKeyEndpoint) Handle(ctx context.Context, req CreateApiKeyReq) (ApiKeyShareResp, error) {
	return wire.KeyShareViewOf(core.CreateKey(ctx, core.KeyDraft{Name: req.Name, Platform: req.Platform, Secret: req.Key}))
}

// CreateApiKeyReq is a new key.
type CreateApiKeyReq struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Key      string `json:"key"`
}

// GetApiKeyEndpoint returns a share the caller may see, and for the key's
// owner every share of the key (linkedShares).
type GetApiKeyEndpoint struct{}

func (GetApiKeyEndpoint) Method() string             { return "GET" }
func (GetApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}" }
func (GetApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetApiKeyEndpoint) Handle(ctx context.Context, req ApiKeyIDReq) (ApiKeyInfoResp, error) {
	id, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return ApiKeyInfoResp{}, err
	}
	share, err := wire.KeyShareViewOf(core.ReadKeyShare(ctx, id))
	if err != nil {
		return ApiKeyInfoResp{}, err
	}
	resp := ApiKeyInfoResp{Share: &share, LinkedShares: []wire.KeyShareView{}}
	if share.Self {
		linked, err := wire.KeyShareViewsOf(core.ListKeyShares(ctx, core.KeyShareFilter{Key: &share.ApiKeyID}))
		if err != nil {
			return ApiKeyInfoResp{}, err
		}
		resp.LinkedShares = linked
	}
	return resp, nil
}

// ApiKeyInfoResp is a share, and for the key's owner every share of the key.
type ApiKeyInfoResp struct {
	Share        *wire.KeyShareView  `json:"share"`
	LinkedShares []wire.KeyShareView `json:"linkedShares"`
}

// ListApiKeyGameSharesEndpoint lists the share links a key pays for, by one
// of its shares: for the key's owner all of them, for head and staff those
// their organisation pays for. context=personal needs the owner's own share;
// context=organization leaves out the owner's personal links.
type ListApiKeyGameSharesEndpoint struct{}

func (ListApiKeyGameSharesEndpoint) Method() string { return "GET" }
func (ListApiKeyGameSharesEndpoint) Path() string   { return "/api/apikeys/{id}/game-shares" }
func (ListApiKeyGameSharesEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (ListApiKeyGameSharesEndpoint) Handle(ctx context.Context, req ApiKeyGameSharesReq) (ApiKeyGameSharesResp, error) {
	id, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return nil, err
	}
	share, err := core.ReadKeyShare(ctx, id)
	if err != nil {
		return nil, err
	}
	if opt(req.Context) == "personal" && !share.Self {
		return nil, errForbidden("Only the key owner can view personal shares")
	}
	links, err := wire.ShareLinkViewsOf(core.ListShareLinks(ctx, core.ShareLinkFilter{Key: &share.Key.ID}))
	if err != nil {
		return nil, err
	}
	result := make(ApiKeyGameSharesResp, 0, len(links))
	for _, l := range links {
		if opt(req.Context) == "organization" && l.Source == "personal" {
			continue
		}
		result = append(result, l)
	}
	return result, nil
}

// ApiKeyGameSharesReq is the key, by one of its shares, and which links.
type ApiKeyGameSharesReq struct {
	ID      string  `path:"id"`
	Context *string `query:"context"`
}

// ApiKeyGameSharesResp is the share links a key pays for.
type ApiKeyGameSharesResp = []wire.ShareLinkView

// ShareApiKeyEndpoint gives a key to its owner's own organisation: its
// workshops may play with it, its head and staff sponsor games and make
// share links with it. A key is shared with an organisation only, never
// with users or workshops, and never across organisations.
type ShareApiKeyEndpoint struct{}

func (ShareApiKeyEndpoint) Method() string             { return "POST" }
func (ShareApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}/shares" }
func (ShareApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ShareApiKeyEndpoint) Handle(ctx context.Context, req ShareApiKeyReq) (ApiKeyShareResp, error) {
	return shareApiKey(ctx, req)
}

// ShareApiKeyLegacyEndpoint is ShareApiKeyEndpoint at v1's path.
type ShareApiKeyLegacyEndpoint struct{}

func (ShareApiKeyLegacyEndpoint) Method() string             { return "POST" }
func (ShareApiKeyLegacyEndpoint) Path() string               { return "/api/apikeys/{id}" }
func (ShareApiKeyLegacyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ShareApiKeyLegacyEndpoint) Handle(ctx context.Context, req ShareApiKeyReq) (ApiKeyShareResp, error) {
	return shareApiKey(ctx, req)
}

func shareApiKey(ctx context.Context, req ShareApiKeyReq) (ApiKeyShareResp, error) {
	id, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return ApiKeyShareResp{}, err
	}
	if req.UserID != nil || req.WorkshopID != nil || req.InstitutionID == nil {
		return ApiKeyShareResp{}, errBadRequest("a key is shared with its owner's organisation only: institutionId")
	}
	return wire.KeyShareViewOf(core.ShareKeyWithOrganisation(ctx, id, *req.InstitutionID))
}

// ShareApiKeyReq is the key, by one of its shares, and the organisation.
// userId and workshopId are refused: v1 shared with those.
type ShareApiKeyReq struct {
	ID            string     `path:"id"`
	UserID        *uuid.UUID `json:"userId,omitempty"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
	InstitutionID *uuid.UUID `json:"institutionId,omitempty"`
}

// UpdateApiKeyEndpoint renames a key, for its owner.
type UpdateApiKeyEndpoint struct{}

func (UpdateApiKeyEndpoint) Method() string             { return "PATCH" }
func (UpdateApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}" }
func (UpdateApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateApiKeyEndpoint) Handle(ctx context.Context, req UpdateApiKeyReq) (ApiKeyShareResp, error) {
	id, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return ApiKeyShareResp{}, err
	}
	return wire.KeyShareViewOf(core.RenameKey(ctx, id, req.Name))
}

// UpdateApiKeyReq is the key, by one of its shares, and its new name.
type UpdateApiKeyReq struct {
	ID   string `path:"id"`
	Name string `json:"name"`
}

// SetDefaultApiKeyEndpoint makes a key its owner's default: the one they
// play with themselves.
type SetDefaultApiKeyEndpoint struct{}

func (SetDefaultApiKeyEndpoint) Method() string             { return "PUT" }
func (SetDefaultApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}/default" }
func (SetDefaultApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetDefaultApiKeyEndpoint) Handle(ctx context.Context, req ApiKeyIDReq) (ApiKeyShareResp, error) {
	id, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return ApiKeyShareResp{}, err
	}
	if err := core.SetDefaultKey(ctx, id); err != nil {
		return ApiKeyShareResp{}, err
	}
	return wire.KeyShareViewOf(core.ReadKeyShare(ctx, id))
}

// CheckApiKeyEndpoint checks a key with its provider, for its owner, and
// answers the share as the check left it.
//
// The check is the smallest call the provider bills, a fraction of a cent on
// the key: it finds a key the provider rejects (lastErrorCode key_rejected)
// and an account with no credit left (no_credit) alike, and records the
// outcome as the key's last usage. A provider that does not answer records
// nothing: 503 ai_error.
type CheckApiKeyEndpoint struct{}

func (CheckApiKeyEndpoint) Method() string             { return "POST" }
func (CheckApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}/check" }
func (CheckApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CheckApiKeyEndpoint) Handle(ctx context.Context, req ApiKeyIDReq) (ApiKeyShareResp, error) {
	id, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return ApiKeyShareResp{}, err
	}
	return wire.KeyShareViewOf(core.CheckKey(ctx, id))
}

// DeleteApiKeyEndpoint takes a share back, stopping what draws from it: the
// owner any share of their key, a head their organisation's, head and staff
// what draws from it. With cascade=true the owner deletes the key itself,
// with every share of it. The owner's own use of a key goes only with the
// key.
type DeleteApiKeyEndpoint struct{}

func (DeleteApiKeyEndpoint) Method() string             { return "DELETE" }
func (DeleteApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}" }
func (DeleteApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteApiKeyEndpoint) Handle(ctx context.Context, req DeleteApiKeyReq) (MessageResp, error) {
	id, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return MessageResp{}, err
	}
	if opt(req.Cascade) == "true" {
		if err := core.DeleteKey(ctx, id); err != nil {
			return MessageResp{}, err
		}
		return MessageResp{Message: "API key deleted"}, nil
	}
	if err := core.RevokeKeyShare(ctx, id); err != nil {
		return MessageResp{}, err
	}
	return MessageResp{Message: "Share removed"}, nil
}

// DeleteApiKeyReq is the share, and whether the key itself goes.
type DeleteApiKeyReq struct {
	ID string `path:"id"`
	// Cascade "true" deletes the key and all its shares.
	Cascade *string `query:"cascade"`
}

// InstitutionApiKeysResp is the keys given to an organisation.
type InstitutionApiKeysResp = wire.Page[wire.KeyShareView]

// GetInstitutionApiKeysEndpoint lists the keys given to an organisation: for
// its head and staff.
type GetInstitutionApiKeysEndpoint struct{}

func (GetInstitutionApiKeysEndpoint) Method() string             { return "GET" }
func (GetInstitutionApiKeysEndpoint) Path() string               { return "/api/institutions/{id}/apikeys" }
func (GetInstitutionApiKeysEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetInstitutionApiKeysEndpoint) Handle(ctx context.Context, req ListOrganisationKeysReq) (InstitutionApiKeysResp, error) {
	org, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return InstitutionApiKeysResp{}, err
	}
	return wire.KeyShareViewPageOf(core.List(ctx, core.OrganisationKeys(org), req.query()))
}

// OrganisationKeySort orders the keys given to an organisation; the default is name.
type OrganisationKeySort string

const (
	OrganisationKeySortName     OrganisationKeySort = "name"
	OrganisationKeySortOwner    OrganisationKeySort = "owner"
	OrganisationKeySortPlatform OrganisationKeySort = "platform"
)

// ListOrganisationKeysReq names the organisation, and filters, sorts and
// pages the keys given to it.
type ListOrganisationKeysReq struct {
	ID string `path:"id"`
	ListParams[KeyFilter, OrganisationKeySort]
}

// SetInstitutionFreeUseApiKeyShareEndpoint sets or clears the key an
// organisation's members play with for free: one of the keys given to it.
// A null shareId clears it.
type SetInstitutionFreeUseApiKeyShareEndpoint struct{}

func (SetInstitutionFreeUseApiKeyShareEndpoint) Method() string { return "PATCH" }
func (SetInstitutionFreeUseApiKeyShareEndpoint) Path() string {
	return "/api/institutions/{id}/free-use-key"
}
func (SetInstitutionFreeUseApiKeyShareEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (SetInstitutionFreeUseApiKeyShareEndpoint) Handle(ctx context.Context, req SetInstitutionFreeUseKeyReq) (InstitutionResp, error) {
	id, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return InstitutionResp{}, err
	}
	return wire.OrganisationViewOf(core.SetOrganisationFreeUseKey(ctx, id, req.ShareID))
}

// SetInstitutionFreeUseKeyReq is the organisation and the share, or null to
// clear it.
type SetInstitutionFreeUseKeyReq struct {
	ID      string     `path:"id"`
	ShareID *uuid.UUID `json:"shareId"`
}

// SetWorkshopApiKeyEndpoint sets the key a workshop plays with: one of the
// keys given to its organisation; null or empty clears it, and the workshop
// plays no more until one is picked.
type SetWorkshopApiKeyEndpoint struct{}

func (SetWorkshopApiKeyEndpoint) Method() string             { return "PUT" }
func (SetWorkshopApiKeyEndpoint) Path() string               { return "/api/workshops/{id}/api-key" }
func (SetWorkshopApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetWorkshopApiKeyEndpoint) Handle(ctx context.Context, req SetWorkshopApiKeyReq) (WorkshopResp, error) {
	id, err := parseID(req.ID, "Invalid workshop ID")
	if err != nil {
		return WorkshopResp{}, err
	}
	share, err := optionalID(req.ApiKeyShareID, "Invalid API key share ID")
	if err != nil {
		return WorkshopResp{}, err
	}
	return wire.WorkshopViewOf(core.SetWorkshopKey(ctx, id, share))
}

// SetWorkshopApiKeyReq is the workshop and the share, or null to clear it.
type SetWorkshopApiKeyReq struct {
	ID            string  `path:"id"`
	ApiKeyShareID *string `json:"apiKeyShareId"`
}
