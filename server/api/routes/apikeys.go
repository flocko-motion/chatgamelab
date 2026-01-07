package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
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
	AllowPublic   bool       `json:"allowPublicSponsoredPlays"`
}

type UpdateApiKeyRequest struct {
	Name *string `json:"name,omitempty"`
}

type ApiKeyInfoResponse struct {
	Share        *obj.ApiKeyShare  `json:"share"`
	LinkedShares []obj.ApiKeyShare `json:"linkedShares"`
}

// GetApiKeys godoc
//
//	@Summary		List API keys
//	@Description	Returns all API key shares accessible to the current user
//	@Tags			apikeys
//	@Produce		json
//	@Success		200	{array}		obj.ApiKeyShare
//	@Failure		401	{object}	httpx.ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/apikeys [get]
func GetApiKeys(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	keys, err := db.GetApiKeySharesByUser(r.Context(), user.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get API keys: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, keys)
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
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if req.Platform == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Platform is required")
		return
	}
	if req.Key == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Key is required")
		return
	}

	share, err := db.CreateApiKeyWithSelfShare(r.Context(), user.ID, req.Name, req.Platform, req.Key)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create API key: "+err.Error())
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

	newShareID, err := db.CreateApiKeyShare(r.Context(), user.ID, shareID, req.UserID, req.WorkshopID, req.InstitutionID, req.AllowPublic)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to share: "+err.Error())
		return
	}

	createdShare, err := db.GetApiKeyShareByID(r.Context(), *newShareID)
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

	share, err := db.GetApiKeyShareByID(r.Context(), shareID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to load updated share: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, share)
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
	share, err := db.GetApiKeyShareByID(r.Context(), shareID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to load share: "+err.Error())
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
