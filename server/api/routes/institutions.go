// package: routes / institution HTTP handlers
// type:    logic
// job:     handles CRUD for institutions, their members, API keys, and prompt constraints
// limits:  no route registration (-> router.go); persistence delegated to db
package routes

import (
	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"
	"net/http"
	"unicode/utf8"

	"github.com/google/uuid"
)

// CreateInstitutionRequest represents the request to create an institution
type CreateInstitutionRequest struct {
	Name string `json:"name"`
	// Optional details (operator and address), as entered at self-registration.
	Operator   string `json:"operator,omitempty"`
	Street     string `json:"street,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	City       string `json:"city,omitempty"`
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
	if req.Operator != "" || req.Street != "" || req.PostalCode != "" || req.City != "" {
		if err := db.SetInstitutionDetails(r.Context(), institution.ID, req.Operator, req.Street, req.PostalCode, req.City); err != nil {
			httpx.WriteAppError(w, obj.ErrServerError("organisation created, but its details could not be saved"))
			return
		}
		institution.Operator, institution.Street = optionalString(req.Operator), optionalString(req.Street)
		institution.PostalCode, institution.City = optionalString(req.PostalCode), optionalString(req.City)
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
//	@Description	Permanently deletes an institution with all its workshops and workshop participants; heads and staff keep their accounts as individual users (admin only). To block access without losing data, lock it instead (POST /institutions/{id}/disable).
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

// SetInstitutionFreeUseTierRequest is the request body for setting the tier the
// institution's free-use key is released up to
type SetInstitutionFreeUseTierRequest struct {
	AiQualityTier *string `json:"aiQualityTier"` // low/medium/high/max; null or "" = server default
}

// SetInstitutionFreeUseAiQualityTier godoc
//
//	@Summary		Set institution free-use key tier
//	@Description	Sets the highest AI quality tier the institution's free-use key is released up to.
//	@Description	Members and workshops without a key of their own play with it at most at this tier.
//	@Description	Pass null or "" to fall back to the server's default tier.
//	@Tags			institutions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Institution ID (UUID)"
//	@Param			request	body		SetInstitutionFreeUseTierRequest	true	"Tier (null to clear)"
//	@Success		200		{object}	obj.Institution
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id}/free-use-tier [patch]
func SetInstitutionFreeUseAiQualityTier(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	institutionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid institution ID")
		return
	}

	var req SetInstitutionFreeUseTierRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if req.AiQualityTier != nil && *req.AiQualityTier == "" {
		req.AiQualityTier = nil
	}

	if err := db.SetInstitutionFreeUseAiQualityTier(r.Context(), user.ID, institutionID, req.AiQualityTier); err != nil {
		if httpErr, ok := err.(*obj.HTTPError); ok {
			httpx.WriteHTTPError(w, httpErr)
			return
		}
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update free-use tier: "+err.Error())
		return
	}

	institution, err := db.GetInstitutionByID(r.Context(), user.ID, institutionID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get updated institution: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, institution)
}

// UpdateInstitutionPromptConstraintsRequest is the request body for updating institution prompt constraints
type UpdateInstitutionPromptConstraintsRequest struct {
	PromptConstraints *string `json:"promptConstraints"`
}

// UpdateInstitutionPromptConstraints godoc
//
//	@Summary		Update institution prompt constraints
//	@Description	Sets or clears prompt constraints for an institution (head or admin only)
//	@Tags			institutions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string											true	"Institution ID"
//	@Param			request	body		UpdateInstitutionPromptConstraintsRequest		true	"Prompt constraints"
//	@Success		200		{object}	obj.Institution
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id}/prompt-constraints [patch]
func UpdateInstitutionPromptConstraints(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	institutionID, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid institution ID")
		return
	}

	var req UpdateInstitutionPromptConstraintsRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	constraint := req.PromptConstraints
	if constraint != nil && *constraint == "" {
		constraint = nil
	}
	if constraint != nil && utf8.RuneCountInString(*constraint) > obj.MaxPromptConstraintLength {
		httpx.WriteAppError(w, obj.ErrValidationf("prompt constraint must be at most %d characters", obj.MaxPromptConstraintLength))
		return
	}

	if err := db.UpdateInstitutionPromptConstraints(r.Context(), user.ID, institutionID, constraint); err != nil {
		if httpErr, ok := err.(*obj.HTTPError); ok {
			httpx.WriteHTTPError(w, httpErr)
			return
		}
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to update prompt constraints: "+err.Error())
		return
	}

	updatedInstitution, err := db.GetInstitutionByID(r.Context(), user.ID, institutionID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to get updated institution: "+err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, updatedInstitution)
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
