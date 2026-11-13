// package: routes / admin handlers for organisations
// type:    logic
// job:     lets admins lock and unlock organisations, list the self-registration applications and create fresh confirmation links to send by hand.
// limits:  no route registration (-> router.go); rules and SQL in db/institution_admin.go.
package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/log"
)

// LockInstitutionRequest carries an optional note on why an organisation is locked.
type LockInstitutionRequest struct {
	Reason string `json:"reason"`
}

// LockInstitution godoc
//
//	@Summary		Lock an organisation
//	@Description	Admin only. Heads, staff and participants of the organisation can no longer use it, its share links and public pages stop working. Nothing is deleted; unlock reverses it.
//	@Tags			institutions
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Institution ID"
//	@Param			request	body		LockInstitutionRequest	false	"Reason"
//	@Success		200		{object}	obj.Institution
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id}/disable [post]
func LockInstitution(w http.ResponseWriter, r *http.Request) {
	setInstitutionLock(w, r, true)
}

// UnlockInstitution godoc
//
//	@Summary		Unlock an organisation
//	@Description	Admin only. Reverses a lock; members can use the organisation again.
//	@Tags			institutions
//	@Produce		json
//	@Param			id	path		string	true	"Institution ID"
//	@Success		200	{object}	obj.Institution
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institutions/{id}/enable [post]
func UnlockInstitution(w http.ResponseWriter, r *http.Request) {
	setInstitutionLock(w, r, false)
}

func setInstitutionLock(w http.ResponseWriter, r *http.Request, disabled bool) {
	user := httpx.UserFromRequest(r)
	id, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid institution ID")
		return
	}
	var req LockInstitutionRequest
	if disabled && r.ContentLength > 0 {
		if err := httpx.ReadJSON(r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
	}
	if err := db.SetInstitutionLock(r.Context(), user.ID, id, disabled, req.Reason); err != nil {
		writeErr(w, err)
		return
	}
	log.Info("organisation lock changed", "institution_id", id, "disabled", disabled, "admin_id", user.ID)
	inst, err := db.GetInstitutionByID(r.Context(), user.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, inst)
}

// ListInstitutionApplications godoc
//
//	@Summary		List organisation applications
//	@Description	Admin only. All self-registration and upgrade applications, newest first (at most 500), including unconfirmed and expired ones. Personal data of old unconfirmed applications is removed by the retention job.
//	@Tags			institution-registration
//	@Produce		json
//	@Success		200	{array}		obj.InstitutionApplication
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/institution-applications [get]
func ListInstitutionApplications(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	apps, err := db.ListInstitutionApplications(r.Context(), user.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, apps)
}
