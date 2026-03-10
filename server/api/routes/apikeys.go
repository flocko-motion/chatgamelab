package routes

import (
	"net/http"
	"strings"
	"time"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// Request/Response types for API keys
type CreateApiKeyRequest struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Key      string `json:"key"`
}

type ShareRequest struct {
	UserID        *uuid.UUID `json:"userId,omitempty"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
	InstitutionID *uuid.UUID `json:"institutionId,omitempty"`
}

type UpdateApiKeyRequest struct {
	Name *string `json:"name,omitempty"`
}

type ApiKeyInfoResponse struct {
	Share        *obj.ApiKeyShare  `json:"share"`
	LinkedShares []obj.ApiKeyShare `json:"linkedShares"`
}

type ApiKeysResponse struct {
	ApiKeys []obj.ApiKey      `json:"apiKeys"`
	Shares  []obj.ApiKeyShare `json:"shares"`
}

// GetApiKeys godoc
//
//	@Summary		List API keys
//	@Description	Returns the user's API keys and all their linked shares (org shares, sponsorships)
//	@Tags			apikeys
//	@Produce		json
//	@Success		200	{object}	ApiKeysResponse
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/apikeys [get]
func GetApiKeys(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	apiKeys, shares, err := db.GetApiKeysWithShares(r.Context(), user.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get API keys: "+err.Error())
		return
	}

	if apiKeys == nil {
		apiKeys = []obj.ApiKey{}
	}
	if shares == nil {
		shares = []obj.ApiKeyShare{}
	}

	httpx.WriteJSON(w, http.StatusOK, ApiKeysResponse{
		ApiKeys: apiKeys,
		Shares:  shares,
	})
}

// CreateApiKey godoc
//
//	@Summary		Create API key
//	@Description	Creates a new API key with automatic self-share
//	@Tags			apikeys
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateApiKeyRequest	true	"API key data"
//	@Success		200		{object}	obj.ApiKeyShare
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/apikeys/new [post]
func CreateApiKey(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	var req CreateApiKeyRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteErrorWithCode(w, http.StatusBadRequest, obj.ErrCodeInvalidInput, "Invalid JSON: "+err.Error())
		return
	}

	// Trim whitespace from all inputs (users often paste keys with trailing newlines/spaces)
	req.Platform = strings.TrimSpace(req.Platform)
	req.Key = strings.TrimSpace(req.Key)

	if req.Platform == "" {
		httpx.WriteErrorWithCode(w, http.StatusBadRequest, obj.ErrCodeValidation, "Platform is required")
		return
	}
	if req.Key == "" {
		httpx.WriteErrorWithCode(w, http.StatusBadRequest, obj.ErrCodeValidation, "Key is required")
		return
	}

	// Default name: capitalize platform + today's date, e.g. "Mistral 12.02.26"
	name := strings.TrimSpace(req.Name)
	if name == "" {
		platform := strings.ToUpper(req.Platform[:1]) + req.Platform[1:]
		name = platform + " " + time.Now().Format("02.01.06")
	}

	share, err := db.CreateApiKeyWithSelfShare(r.Context(), user.ID, name, req.Platform, req.Key)
	if err != nil {
		// Check if it's a platform validation error
		if strings.Contains(err.Error(), "unknown platform") {
			httpx.WriteErrorWithCode(w, http.StatusBadRequest, obj.ErrCodeInvalidPlatform, "Invalid platform: "+req.Platform)
			return
		}
		httpx.WriteErrorWithCode(w, http.StatusInternalServerError, obj.ErrCodeServerError, "Failed to create API key: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, share)
}

// GetApiKeyByID godoc
//
//	@Summary		Get API key info
//	@Description	Returns share info and linked shares for an API key share
//	@Tags			apikeys
//	@Produce		json
//	@Param			id	path		string	true	"Share ID (UUID)"
//	@Success		200	{object}	ApiKeyInfoResponse
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid share ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/apikeys/{id} [get]
func GetApiKeyByID(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	shareID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid share ID")
		return
	}

	share, linkedShares, err := db.GetApiKeyShareInfo(r.Context(), user.ID, shareID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get share: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ApiKeyInfoResponse{
		Share:        share,
		LinkedShares: linkedShares,
	})
}

// GetApiKeyGameShares godoc
//
//	@Summary		Get game shares for API key share
//	@Description	Returns game share links that use this API key share, enriched with game name and context.
//	@Description	Use ?context=personal to see all shares (requires key ownership).
//	@Description	Use ?context=organization to see only org/workshop shares.
//	@Tags			apikeys
//	@Produce		json
//	@Param			id		path		string	true	"API Key Share ID (UUID)"
//	@Param			context	query		string	false	"Filter context: 'personal' (all, owner only) or 'organization' (org/workshop only)"
//	@Success		200		{array}		EnrichedGameShare
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/apikeys/{id}/game-shares [get]
func GetApiKeyGameShares(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	shareID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid share ID")
		return
	}

	queryContext := r.URL.Query().Get("context") // "personal" or "organization"

	// Verify access to this share
	share, err := db.GetApiKeyShareByID(r.Context(), user.ID, shareID)
	if err != nil {
		httpx.WriteError(w, http.StatusForbidden, "Not authorized to view this share")
		return
	}

	// Personal context requires key ownership
	if queryContext == "personal" && share.ApiKey.UserID != user.ID {
		httpx.WriteError(w, http.StatusForbidden, "Only the key owner can view personal shares")
		return
	}

	// Get game shares for all shares of this API key
	gameShares, err := db.GetGameSharesWithGameByApiKeyID(r.Context(), share.ApiKeyID)
	if err != nil {
		log.Warn("failed to get game shares for api key", "api_key_id", share.ApiKeyID, "error", err)
		httpx.WriteJSON(w, http.StatusOK, []EnrichedGameShare{})
		return
	}

	result := make([]EnrichedGameShare, 0, len(gameShares))
	for _, gs := range gameShares {
		source := "personal"
		workshopName := ""
		if gs.WorkshopID != nil {
			source = "workshop"
			if name, err := db.GetWorkshopName(r.Context(), *gs.WorkshopID); err == nil {
				workshopName = name
			}
		} else if gs.InstitutionID != nil {
			source = "organization"
		}

		// In organization context, skip personal shares
		if queryContext == "organization" && source == "personal" {
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

	httpx.WriteJSON(w, http.StatusOK, result)
}

// ShareApiKey godoc
//
//	@Summary		Share API key
//	@Description	Shares an API key with a user, workshop, or institution
//	@Tags			apikeys
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string			true	"Share ID (UUID)"
//	@Param			request	body		ShareRequest	true	"Share request"
//	@Success		200		{object}	obj.ApiKeyShare
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/apikeys/{id}/shares [post]
func ShareApiKey(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	shareID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid share ID")
		return
	}

	var req ShareRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if req.UserID == nil && req.WorkshopID == nil && req.InstitutionID == nil {
		httpx.WriteError(w, http.StatusBadRequest, "At least one of userId, workshopId, or institutionId is required")
		return
	}

	newShareID, err := db.CreateApiKeyShare(r.Context(), user.ID, shareID, req.UserID, req.WorkshopID, req.InstitutionID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to share: "+err.Error())
		return
	}

	createdShare, err := db.GetApiKeyShareByID(r.Context(), user.ID, *newShareID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to load created share: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, createdShare)
}

// UpdateApiKey godoc
//
//	@Summary		Update API key
//	@Description	Updates API key name
//	@Tags			apikeys
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string			true	"Share ID (UUID)"
//	@Param			request	body		UpdateApiKeyRequest	true	"Update request"
//	@Success		200		{object}	obj.ApiKeyShare
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid request"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/apikeys/{id} [patch]
func UpdateApiKey(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	shareID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid share ID")
		return
	}

	var req UpdateApiKeyRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if req.Name != nil {
		if err := db.UpdateApiKeyName(r.Context(), user.ID, shareID, *req.Name); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to update: "+err.Error())
			return
		}
	}

	share, err := db.GetApiKeyShareByID(r.Context(), user.ID, shareID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to load updated share: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, share)
}

// SetDefaultApiKey godoc
//
//	@Summary		Set default API key
//	@Description	Sets the given API key as the user's default key for session creation
//	@Tags			apikeys
//	@Produce		json
//	@Param			id	path		string	true	"Share ID (UUID)"
//	@Success		200	{object}	obj.ApiKeyShare
//	@Failure		400	{object}	httpx.ErrorResponse	"Invalid share ID"
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/apikeys/{id}/default [put]
func SetDefaultApiKey(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	shareID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid share ID")
		return
	}

	// Look up the share to get the underlying API key ID
	share, err := db.GetApiKeyShareByID(r.Context(), user.ID, shareID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Share not found")
		return
	}

	if err := db.SetDefaultApiKey(r.Context(), user.ID, share.ApiKey.ID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to set default: "+err.Error())
		return
	}

	// Reload the share to get updated IsDefault
	updated, err := db.GetApiKeyShareByID(r.Context(), user.ID, shareID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to reload share: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, updated)
}

// DeleteApiKey godoc
//
//	@Summary		Delete or unshare API key
//	@Description	If ?cascade=true, deletes key and all shares; otherwise unshares (deletes single share)
//	@Tags			apikeys
//	@Produce		json
//	@Param			id		path		string	true	"Share ID (UUID)"
//	@Param			cascade	query		bool	false	"Delete key and all shares"
//	@Success		200		{object}	obj.ApiKeyShare
//	@Failure		400		{object}	httpx.ErrorResponse	"Invalid share ID"
//	@Failure		401		{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/apikeys/{id} [delete]
func DeleteApiKey(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	shareID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid share ID")
		return
	}

	cascade := httpx.QueryParam(r, "cascade") == "true"

	// Capture the share before deleting so we can return an obj type.
	// (After deletion it may no longer be loadable.)
	share, err := db.GetApiKeyShareByID(r.Context(), user.ID, shareID)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Share not found")
		return
	}

	if cascade {
		if err := db.DeleteApiKey(r.Context(), user.ID, shareID); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete key: "+err.Error())
			return
		}
		httpx.WriteJSON(w, http.StatusOK, share)
		return
	}

	if err := db.DeleteApiKeyShare(r.Context(), user.ID, shareID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to unshare: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, share)
}
