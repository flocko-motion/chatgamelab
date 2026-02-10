package routes

import (
	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"
	"net/http"

	"github.com/google/uuid"
)

// CreateInstitutionRequest represents the request to create an institution
type CreateInstitutionRequest struct {
	Name string `json:"name"`
}

// UpdateInstitutionRequest represents the request to update an institution
type UpdateInstitutionRequest struct {
	Name string `json:"name"`
}

// CreateInstitution godoc
//
//	@Summary		Create institution
//	@Description	Creates a new institution
//	@Tags			institutions
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateInstitutionRequest	true	"Institution details"
//	@Success		200		{object}	obj.Institution
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions [post]
func CreateInstitution(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	var req CreateInstitutionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteAppError(w, obj.ErrInvalidInput("Invalid JSON"))
		return
	}

	if req.Name == "" {
		httpx.WriteAppError(w, obj.ErrValidation("Name is required"))
		return
	}

	institution, err := db.CreateInstitution(r.Context(), user.ID, req.Name)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, institution)
}

// ListInstitutions godoc
//
//	@Summary		List institutions
//	@Description	Lists all institutions
//	@Tags			institutions
//	@Produce		json
//	@Success		200	{array}		obj.Institution
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions [get]
func ListInstitutions(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	institutions, err := db.ListInstitutions(r.Context(), user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, institutions)
}

// GetInstitution godoc
//
//	@Summary		Get institution
//	@Description	Gets an institution by ID
//	@Tags			institutions
//	@Produce		json
//	@Param			id	path		string	true	"Institution ID"
//	@Success		200	{object}	obj.Institution
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id} [get]
func GetInstitution(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	id, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
		return
	}

	institution, err := db.GetInstitutionByID(r.Context(), user.ID, id)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, institution)
}

// UpdateInstitution godoc
//
//	@Summary		Update institution
//	@Description	Updates an institution
//	@Tags			institutions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Institution ID"
//	@Param			request	body		UpdateInstitutionRequest	true	"Institution details"
//	@Success		200		{object}	obj.Institution
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id} [patch]
func UpdateInstitution(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	id, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
		return
	}

	var req UpdateInstitutionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteAppError(w, obj.ErrInvalidInput("Invalid JSON"))
		return
	}

	if req.Name == "" {
		httpx.WriteAppError(w, obj.ErrValidation("Name is required"))
		return
	}

	institution, err := db.UpdateInstitution(r.Context(), id, user.ID, req.Name)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, institution)
}

// DeleteInstitution godoc
//
//	@Summary		Delete institution
//	@Description	Soft-deletes an institution (admin only)
//	@Tags			institutions
//	@Produce		json
//	@Param			id	path		string	true	"Institution ID"
//	@Success		200	{object}	map[string]string
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id} [delete]
func DeleteInstitution(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	id, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
		return
	}

	err = db.DeleteInstitution(r.Context(), id, user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Institution deleted",
	})
}

// GetInstitutionMembers godoc
//
//	@Summary		Get institution members
//	@Description	Returns all members of an institution
//	@Tags			institutions
//	@Produce		json
//	@Param			id	path		string	true	"Institution ID"
//	@Success		200	{array}		obj.User
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id}/members [get]
func GetInstitutionMembers(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	institutionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
		return
	}

	members, err := db.GetInstitutionMembers(r.Context(), institutionID, user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, members)
}

// GetInstitutionApiKeys godoc
//
//	@Summary		Get institution API keys
//	@Description	Returns all API keys shared with an institution (heads and staff only)
//	@Tags			institutions
//	@Produce		json
//	@Param			id	path		string	true	"Institution ID"
//	@Success		200	{array}		obj.ApiKeyShare
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id}/apikeys [get]
func GetInstitutionApiKeys(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	institutionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
		return
	}

	shares, err := db.GetApiKeySharesByInstitution(r.Context(), user.ID, institutionID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, shares)
}

// RemoveInstitutionMember godoc
//
//	@Summary		Remove member from institution
//	@Description	Removes a member from an institution (head or admin only)
//	@Tags			institutions
//	@Produce		json
//	@Param			id		path		string	true	"Institution ID"
//	@Param			userID	path		string	true	"User ID"
//	@Success		200		{object}	map[string]string
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id}/members/{userID} [delete]
func RemoveInstitutionMember(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	institutionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
		return
	}

	userID, err := httpx.PathParamUUID(r, "userID")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid user ID"))
		return
	}

	err = db.RemoveInstitutionMember(r.Context(), institutionID, userID, user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Member removed",
	})
}

// SetInstitutionFreeUseKeyRequest is the request body for setting the institution free-use API key share
type SetInstitutionFreeUseKeyRequest struct {
	ShareID *uuid.UUID `json:"shareId"`
}

// SetInstitutionFreeUseApiKeyShare godoc
//
//	@Summary		Set institution free-use API key share
//	@Description	Sets or clears the free-use API key share for an institution.
//	@Description	Any institution member can use this key to play for free.
//	@Description	Pass null shareId to clear.
//	@Tags			institutions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Institution ID (UUID)"
//	@Param			request	body		SetInstitutionFreeUseKeyRequest	true	"Share ID to set (null to clear)"
//	@Success		200		{object}	obj.Institution
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id}/free-use-key [patch]
func SetInstitutionFreeUseApiKeyShare(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	institutionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid institution ID")
		return
	}

	var req SetInstitutionFreeUseKeyRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if err := db.SetInstitutionFreeUseApiKeyShare(r.Context(), user.ID, institutionID, req.ShareID); err != nil {
		if httpErr, ok := err.(*obj.HTTPError); ok {
			httpx.WriteHTTPError(w, httpErr)
			return
		}
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update free-use key: "+err.Error())
		return
	}

	// Return updated institution
	institution, err := db.GetInstitutionByID(r.Context(), user.ID, institutionID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get updated institution: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, institution)
}
