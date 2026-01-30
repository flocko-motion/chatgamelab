package routes

import (
	"cgl/api/httpx"
	"cgl/db"
	"cgl/obj"
	"net/http"

	"github.com/google/uuid"
)

// CreateWorkshopRequest represents the request to create a workshop
type CreateWorkshopRequest struct {
	InstitutionID *string `json:"institutionId,omitempty"`
	Name          string  `json:"name"`
	Active        bool    `json:"active"`
	Public        bool    `json:"public"`
}

// UpdateWorkshopRequest represents the request to update a workshop
type UpdateWorkshopRequest struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Public bool   `json:"public"`
}

// CreateWorkshop godoc
//
//	@Summary		Create workshop
//	@Description	Creates a new workshop
//	@Tags			workshops
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateWorkshopRequest	true	"Workshop details"
//	@Success		200		{object}	obj.Workshop
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops [post]
func CreateWorkshop(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	var req CreateWorkshopRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteAppError(w, obj.ErrInvalidInput("Invalid JSON"))
		return
	}

	if req.Name == "" {
		httpx.WriteAppError(w, obj.ErrValidation("Name is required"))
		return
	}

	var institutionID *uuid.UUID
	if req.InstitutionID != nil {
		id, err := uuid.Parse(*req.InstitutionID)
		if err != nil {
			httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
			return
		}
		institutionID = &id
	}

	workshop, err := db.CreateWorkshop(r.Context(), user.ID, institutionID, req.Name, req.Active, req.Public)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, workshop)
}

// ListWorkshops godoc
//
//	@Summary		List workshops
//	@Description	Lists all workshops or workshops for a specific institution
//	@Tags			workshops
//	@Produce		json
//	@Param			institutionId	query		string	false	"Institution ID"
//	@Param			search			query		string	false	"Search by name"
//	@Param			sortBy			query		string	false	"Sort by field (name, createdAt, participantCount)"
//	@Param			sortDir			query		string	false	"Sort direction (asc, desc)"
//	@Param			activeOnly		query		bool	false	"Filter to active workshops only"
//	@Success		200				{array}		obj.Workshop
//	@Failure		500				{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops [get]
func ListWorkshops(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	institutionIDStr := httpx.QueryParam(r, "institutionId")
	search := httpx.QueryParam(r, "search")
	sortBy := httpx.QueryParam(r, "sortBy")
	sortDir := httpx.QueryParam(r, "sortDir")
	activeOnlyStr := httpx.QueryParam(r, "activeOnly")

	var institutionID *uuid.UUID
	if institutionIDStr != "" {
		id, parseErr := uuid.Parse(institutionIDStr)
		if parseErr != nil {
			httpx.WriteAppError(w, obj.ErrValidation("Invalid institution ID"))
			return
		}
		institutionID = &id
	}

	// Parse activeOnly filter
	var activeOnly *bool
	if activeOnlyStr == "true" {
		t := true
		activeOnly = &t
	}

	// Build filter options
	opts := db.ListWorkshopsOptions{
		Search:     search,
		SortBy:     sortBy,
		SortDir:    sortDir,
		ActiveOnly: activeOnly,
	}

	workshops, err := db.ListWorkshopsWithOptions(r.Context(), user.ID, institutionID, opts)

	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, workshops)
}

// GetWorkshop godoc
//
//	@Summary		Get workshop
//	@Description	Gets a workshop by ID
//	@Tags			workshops
//	@Produce		json
//	@Param			id	path		string	true	"Workshop ID"
//	@Success		200	{object}	obj.Workshop
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops/{id} [get]
func GetWorkshop(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	id, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid workshop ID"))
		return
	}

	workshop, err := db.GetWorkshopByID(r.Context(), user.ID, id)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, workshop)
}

// UpdateWorkshop godoc
//
//	@Summary		Update workshop
//	@Description	Updates a workshop
//	@Tags			workshops
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Workshop ID"
//	@Param			request	body		UpdateWorkshopRequest	true	"Workshop details"
//	@Success		200		{object}	obj.Workshop
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops/{id} [patch]
func UpdateWorkshop(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	id, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid workshop ID"))
		return
	}

	var req UpdateWorkshopRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteAppError(w, obj.ErrInvalidInput("Invalid JSON"))
		return
	}

	if req.Name == "" {
		httpx.WriteAppError(w, obj.ErrValidation("Name is required"))
		return
	}

	workshop, err := db.UpdateWorkshop(r.Context(), id, user.ID, req.Name, req.Active, req.Public)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, workshop)
}

// DeleteWorkshop godoc
//
//	@Summary		Delete workshop
//	@Description	Soft-deletes a workshop
//	@Tags			workshops
//	@Produce		json
//	@Param			id	path		string	true	"Workshop ID"
//	@Success		200	{object}	map[string]string
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops/{id} [delete]
func DeleteWorkshop(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	id, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid workshop ID"))
		return
	}

	err = db.DeleteWorkshop(r.Context(), id, user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Workshop deleted",
	})
}

// SetWorkshopApiKeyRequest represents the request to set a workshop's default API key
type SetWorkshopApiKeyRequest struct {
	ApiKeyShareID *string `json:"apiKeyShareId"`
}

// SetWorkshopApiKey godoc
//
//	@Summary		Set workshop default API key
//	@Description	Sets the default API key for workshop participants (staff/heads only)
//	@Tags			workshops
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Workshop ID"
//	@Param			request	body		SetWorkshopApiKeyRequest	true	"API key share ID"
//	@Success		200		{object}	obj.Workshop
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops/{id}/api-key [put]
func SetWorkshopApiKey(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	id, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid workshop ID"))
		return
	}

	var req SetWorkshopApiKeyRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteAppError(w, obj.ErrInvalidInput("Invalid JSON"))
		return
	}

	var apiKeyShareID *uuid.UUID
	if req.ApiKeyShareID != nil && *req.ApiKeyShareID != "" {
		parsed, err := uuid.Parse(*req.ApiKeyShareID)
		if err != nil {
			httpx.WriteAppError(w, obj.ErrValidation("Invalid API key share ID"))
			return
		}
		apiKeyShareID = &parsed
	}

	workshop, err := db.SetWorkshopDefaultApiKey(r.Context(), id, user.ID, apiKeyShareID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, workshop)
}

// GetParticipantToken godoc
//
//	@Summary		Get participant token
//	@Description	Gets the access token for a workshop participant (staff/heads only)
//	@Tags			workshops
//	@Produce		json
//	@Param			participantId	path		string	true	"Participant ID"
//	@Success		200				{object}	map[string]string
//	@Failure		403				{object}	httpx.ErrorResponse
//	@Failure		404				{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workshops/participants/{participantId}/token [get]
func GetParticipantToken(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)

	participantID, err := httpx.PathParamUUID(r, "participantId")
	if err != nil {
		httpx.WriteAppError(w, obj.ErrValidation("Invalid participant ID"))
		return
	}

	token, err := db.GetWorkshopParticipantToken(r.Context(), participantID, user.ID)
	if err != nil {
		if appErr, ok := err.(*obj.AppError); ok {
			httpx.WriteAppError(w, appErr)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"token": token,
	})
}
