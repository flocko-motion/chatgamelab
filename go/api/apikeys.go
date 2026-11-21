// package: api / schemaf endpoints
// type:    logic
// job:     lists, creates, updates, shares and deletes AI provider API keys
// limits:  key storage and its permission checks live in db
package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cgl/api/wire"
	"cgl/core"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// ApiKeyShareResp is an API key share.
type ApiKeyShareResp = obj.ApiKeyShare

// ApiKeyIDReq names an API key share by its id.
type ApiKeyIDReq struct {
	ID string `path:"id"`
}

// ListApiKeysEndpoint lists the caller's API keys and all their linked shares
// (organization shares, sponsorships).
type ListApiKeysEndpoint struct{}

func (ListApiKeysEndpoint) Method() string             { return "GET" }
func (ListApiKeysEndpoint) Path() string               { return "/api/apikeys" }
func (ListApiKeysEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListApiKeysEndpoint) Handle(ctx context.Context, _ ListApiKeysReq) (ApiKeysResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return ApiKeysResp{}, err
	}
	apiKeys, shares, err := db.GetApiKeysWithShares(ctx, user.ID)
	if err != nil {
		return ApiKeysResp{}, fmt.Errorf("failed to get API keys: %w", err)
	}
	if apiKeys == nil {
		apiKeys = []obj.ApiKey{}
	}
	if shares == nil {
		shares = []obj.ApiKeyShare{}
	}
	return ApiKeysResp{ApiKeys: apiKeys, Shares: shares}, nil
}

// ListApiKeysReq takes no parameters.
type ListApiKeysReq struct{}

// ApiKeysResp holds the API keys the caller owns and the shares granted to them.
type ApiKeysResp struct {
	ApiKeys []obj.ApiKey      `json:"apiKeys"`
	Shares  []obj.ApiKeyShare `json:"shares"`
}

// CreateApiKeyEndpoint creates an API key and shares it with the caller.
//
// Platform and key are required; surrounding whitespace is trimmed. Without a
// name the key is named after its platform and today's date, e.g.
// "Mistral 12.02.26".
type CreateApiKeyEndpoint struct{}

func (CreateApiKeyEndpoint) Method() string             { return "POST" }
func (CreateApiKeyEndpoint) Path() string               { return "/api/apikeys/new" }
func (CreateApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (CreateApiKeyEndpoint) Handle(ctx context.Context, req CreateApiKeyReq) (ApiKeyShareResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.ApiKeyShare{}, err
	}
	// Users often paste keys with trailing newlines or spaces.
	platform := strings.TrimSpace(req.Platform)
	key := strings.TrimSpace(req.Key)
	if platform == "" {
		return obj.ApiKeyShare{}, coded(http.StatusBadRequest, obj.ErrCodeValidation, "Platform is required")
	}
	if key == "" {
		return obj.ApiKeyShare{}, coded(http.StatusBadRequest, obj.ErrCodeValidation, "Key is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.ToUpper(platform[:1]) + platform[1:] + " " + time.Now().Format("02.01.06")
	}
	share, err := db.CreateApiKeyWithSelfShare(ctx, user.ID, name, platform, key)
	if err != nil {
		if strings.Contains(err.Error(), "unknown platform") {
			return obj.ApiKeyShare{}, coded(http.StatusBadRequest, obj.ErrCodeInvalidPlatform, "Invalid platform: "+platform)
		}
		return obj.ApiKeyShare{}, coded(http.StatusInternalServerError, obj.ErrCodeServerError, "Failed to create API key: "+err.Error())
	}
	return *share, nil
}

// CreateApiKeyReq is a new API key's name, platform and secret.
type CreateApiKeyReq struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Key      string `json:"key"`
}

// GetApiKeyEndpoint returns an API key share and the shares linked to it.
type GetApiKeyEndpoint struct{}

func (GetApiKeyEndpoint) Method() string             { return "GET" }
func (GetApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}" }
func (GetApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetApiKeyEndpoint) Handle(ctx context.Context, req ApiKeyIDReq) (ApiKeyInfoResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return ApiKeyInfoResp{}, err
	}
	shareID, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return ApiKeyInfoResp{}, err
	}
	share, linkedShares, err := db.GetApiKeyShareInfo(ctx, user.ID, shareID)
	if err != nil {
		return ApiKeyInfoResp{}, fmt.Errorf("failed to get share: %w", err)
	}
	return ApiKeyInfoResp{Share: share, LinkedShares: linkedShares}, nil
}

// ApiKeyInfoResp holds an API key share and the shares linked to it.
type ApiKeyInfoResp struct {
	Share        *obj.ApiKeyShare  `json:"share"`
	LinkedShares []obj.ApiKeyShare `json:"linkedShares"`
}

// ListApiKeyGameSharesEndpoint lists the game share links that use an API key,
// with each game's name and where the share comes from.
//
// Query: context=personal lists all shares and requires owning the key;
// context=organization lists only organization and workshop shares.
type ListApiKeyGameSharesEndpoint struct{}

func (ListApiKeyGameSharesEndpoint) Method() string { return "GET" }
func (ListApiKeyGameSharesEndpoint) Path() string   { return "/api/apikeys/{id}/game-shares" }
func (ListApiKeyGameSharesEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (ListApiKeyGameSharesEndpoint) Handle(ctx context.Context, req ApiKeyGameSharesReq) (ApiKeyGameSharesResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	shareID, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return nil, err
	}
	share, err := db.GetApiKeyShareByID(ctx, user.ID, shareID)
	if err != nil {
		return nil, errForbidden("Not authorized to view this share")
	}
	if opt(req.Context) == "personal" && share.ApiKey.UserID != user.ID {
		return nil, errForbidden("Only the key owner can view personal shares")
	}

	gameShares, err := db.GetGameSharesWithGameByApiKeyID(ctx, share.ApiKeyID)
	if err != nil {
		log.Warn("failed to get game shares for api key", "api_key_id", share.ApiKeyID, "error", err)
		return ApiKeyGameSharesResp{}, nil
	}

	result := make(ApiKeyGameSharesResp, 0, len(gameShares))
	for _, gs := range gameShares {
		source := "personal"
		workshopName := ""
		if gs.WorkshopID != nil {
			source = "workshop"
			if name, err := db.GetWorkshopName(ctx, *gs.WorkshopID); err == nil {
				workshopName = name
			}
		} else if gs.InstitutionID != nil {
			source = "organization"
		}
		if opt(req.Context) == "organization" && source == "personal" {
			continue
		}
		result = append(result, EnrichedGameShare{
			GameShare:    gs.GameShare,
			ShareURL:     "/play/" + gs.Token,
			Source:       source,
			WorkshopName: workshopName,
			GameName:     gs.GameName,
		})
	}
	return result, nil
}

// ApiKeyGameSharesReq names an API key share and filters its game shares.
type ApiKeyGameSharesReq struct {
	ID string `path:"id"`
	// Context is "personal" or "organization".
	Context *string `query:"context"`
}

// ApiKeyGameSharesResp is a list of game shares that use an API key.
type ApiKeyGameSharesResp = []EnrichedGameShare

// ShareApiKeyEndpoint shares an API key with a user, workshop or institution.
type ShareApiKeyEndpoint struct{}

func (ShareApiKeyEndpoint) Method() string             { return "POST" }
func (ShareApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}/shares" }
func (ShareApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ShareApiKeyEndpoint) Handle(ctx context.Context, req ShareApiKeyReq) (ApiKeyShareResp, error) {
	return shareApiKey(ctx, req)
}

// ShareApiKeyLegacyEndpoint shares an API key with a user, workshop or
// institution, at the path older clients use.
type ShareApiKeyLegacyEndpoint struct{}

func (ShareApiKeyLegacyEndpoint) Method() string             { return "POST" }
func (ShareApiKeyLegacyEndpoint) Path() string               { return "/api/apikeys/{id}" }
func (ShareApiKeyLegacyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ShareApiKeyLegacyEndpoint) Handle(ctx context.Context, req ShareApiKeyReq) (ApiKeyShareResp, error) {
	return shareApiKey(ctx, req)
}

// shareApiKey shares the key behind req's share with req's target.
func shareApiKey(ctx context.Context, req ShareApiKeyReq) (obj.ApiKeyShare, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.ApiKeyShare{}, err
	}
	shareID, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return obj.ApiKeyShare{}, err
	}
	if req.UserID == nil && req.WorkshopID == nil && req.InstitutionID == nil {
		return obj.ApiKeyShare{}, errBadRequest("At least one of userId, workshopId, or institutionId is required")
	}
	newShareID, err := db.CreateApiKeyShare(ctx, user.ID, shareID, req.UserID, req.WorkshopID, req.InstitutionID)
	if err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("failed to share: %w", err)
	}
	created, err := db.GetApiKeyShareByID(ctx, user.ID, *newShareID)
	if err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("failed to load created share: %w", err)
	}
	return *created, nil
}

// ShareApiKeyReq is the share from the path and the target from the body: a
// user, a workshop or an institution.
type ShareApiKeyReq struct {
	ID            string     `path:"id"`
	UserID        *uuid.UUID `json:"userId,omitempty"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
	InstitutionID *uuid.UUID `json:"institutionId,omitempty"`
}

// UpdateApiKeyEndpoint renames an API key.
type UpdateApiKeyEndpoint struct{}

func (UpdateApiKeyEndpoint) Method() string             { return "PATCH" }
func (UpdateApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}" }
func (UpdateApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (UpdateApiKeyEndpoint) Handle(ctx context.Context, req UpdateApiKeyReq) (ApiKeyShareResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.ApiKeyShare{}, err
	}
	shareID, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return obj.ApiKeyShare{}, err
	}
	if req.Name != nil {
		if err := db.UpdateApiKeyName(ctx, user.ID, shareID, *req.Name); err != nil {
			return obj.ApiKeyShare{}, fmt.Errorf("failed to update: %w", err)
		}
	}
	share, err := db.GetApiKeyShareByID(ctx, user.ID, shareID)
	if err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("failed to load updated share: %w", err)
	}
	return *share, nil
}

// UpdateApiKeyReq is the share from the path and the key's new name.
type UpdateApiKeyReq struct {
	ID   string  `path:"id"`
	Name *string `json:"name,omitempty"`
}

// SetDefaultApiKeyEndpoint makes an API key the caller's default for new sessions.
type SetDefaultApiKeyEndpoint struct{}

func (SetDefaultApiKeyEndpoint) Method() string             { return "PUT" }
func (SetDefaultApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}/default" }
func (SetDefaultApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (SetDefaultApiKeyEndpoint) Handle(ctx context.Context, req ApiKeyIDReq) (ApiKeyShareResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.ApiKeyShare{}, err
	}
	shareID, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return obj.ApiKeyShare{}, err
	}
	share, err := db.GetApiKeyShareByID(ctx, user.ID, shareID)
	if err != nil {
		return obj.ApiKeyShare{}, errNotFound("Share not found")
	}
	if err := db.SetDefaultApiKey(ctx, user.ID, share.ApiKey.ID); err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("failed to set default: %w", err)
	}
	updated, err := db.GetApiKeyShareByID(ctx, user.ID, shareID)
	if err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("failed to reload share: %w", err)
	}
	return *updated, nil
}

// DeleteApiKeyEndpoint removes an API key share and answers with it as it was.
//
// With cascade=true it deletes the key and all its shares.
type DeleteApiKeyEndpoint struct{}

func (DeleteApiKeyEndpoint) Method() string             { return "DELETE" }
func (DeleteApiKeyEndpoint) Path() string               { return "/api/apikeys/{id}" }
func (DeleteApiKeyEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (DeleteApiKeyEndpoint) Handle(ctx context.Context, req DeleteApiKeyReq) (ApiKeyShareResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return obj.ApiKeyShare{}, err
	}
	shareID, err := parseID(req.ID, "Invalid share ID")
	if err != nil {
		return obj.ApiKeyShare{}, err
	}
	// Loaded first: after the delete it may no longer be loadable.
	share, err := db.GetApiKeyShareByID(ctx, user.ID, shareID)
	if err != nil {
		return obj.ApiKeyShare{}, errNotFound("Share not found")
	}
	if opt(req.Cascade) == "true" {
		if err := db.DeleteApiKey(ctx, user.ID, shareID); err != nil {
			return obj.ApiKeyShare{}, fmt.Errorf("failed to delete key: %w", err)
		}
		return *share, nil
	}
	if err := db.DeleteApiKeyShare(ctx, user.ID, shareID); err != nil {
		return obj.ApiKeyShare{}, fmt.Errorf("failed to unshare: %w", err)
	}
	return *share, nil
}

// DeleteApiKeyReq is the share from the path and whether to delete the whole key.
type DeleteApiKeyReq struct {
	ID string `path:"id"`
	// Cascade "true" deletes the key and all its shares.
	Cascade *string `query:"cascade"`
}

// InstitutionApiKeysResp is the API key shares of an institution.
type InstitutionApiKeysResp = []obj.ApiKeyShare

// GetInstitutionApiKeysEndpoint lists the API keys shared with an institution;
// heads and staff only.
type GetInstitutionApiKeysEndpoint struct{}

func (GetInstitutionApiKeysEndpoint) Method() string             { return "GET" }
func (GetInstitutionApiKeysEndpoint) Path() string               { return "/api/institutions/{id}/apikeys" }
func (GetInstitutionApiKeysEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetInstitutionApiKeysEndpoint) Handle(ctx context.Context, req InstitutionIDReq) (InstitutionApiKeysResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	institutionID, err := parseIDValidation(req.ID, "Invalid institution ID")
	if err != nil {
		return nil, err
	}
	shares, err := db.GetApiKeySharesByInstitution(ctx, user.ID, institutionID)
	if err != nil {
		return nil, appError(err)
	}
	return shares, nil
}

// SetInstitutionFreeUseApiKeyShareEndpoint sets or clears an institution's
// free-use API key share and answers with the institution.
//
// Any member of the institution plays for free with this key. A null shareId
// clears it.
type SetInstitutionFreeUseApiKeyShareEndpoint struct{}

func (SetInstitutionFreeUseApiKeyShareEndpoint) Method() string { return "PATCH" }
func (SetInstitutionFreeUseApiKeyShareEndpoint) Path() string {
	return "/api/institutions/{id}/free-use-key"
}
func (SetInstitutionFreeUseApiKeyShareEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (SetInstitutionFreeUseApiKeyShareEndpoint) Handle(ctx context.Context, req SetInstitutionFreeUseKeyReq) (InstitutionResp, error) {
	user, err := caller(ctx)
	if err != nil {
		return InstitutionResp{}, err
	}
	institutionID, err := parseID(req.ID, "Invalid institution ID")
	if err != nil {
		return InstitutionResp{}, err
	}
	if err := db.SetInstitutionFreeUseApiKeyShare(ctx, user.ID, institutionID, req.ShareID); err != nil {
		return InstitutionResp{}, appError(err)
	}
	return wire.OrganisationViewOf(core.ReadOrganisation(ctx, institutionID))
}

// SetInstitutionFreeUseKeyReq is the institution from the path and the share
// to set, or null to clear it.
type SetInstitutionFreeUseKeyReq struct {
	ID      string     `path:"id"`
	ShareID *uuid.UUID `json:"shareId"`
}
