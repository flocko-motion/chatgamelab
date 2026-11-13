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
	"cgl/mail"
	"cgl/obj"
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

// InstitutionApplicationLinkResponse is a fresh confirmation link plus a ready-to-send mail text.
type InstitutionApplicationLinkResponse struct {
	Application *obj.InstitutionApplication `json:"application"`
	Link        string                      `json:"link"`
	To          string                      `json:"to"`
	Subject     string                      `json:"subject"`
	Body        string                      `json:"body"`
	// MailConfigured is false while the server has no mail server (SMTP_HOST empty), so the link must be sent by hand.
	MailConfigured bool `json:"mailConfigured"`
}

// CreateInstitutionApplicationLink godoc
//
//	@Summary		Create a fresh confirmation link for an application
//	@Description	Admin only. Gives an open or lapsed application a new confirmation link (valid 48 hours; the previous link stops working) and returns it with a ready-to-send mail text, so the admin can send it by hand. Nothing is mailed.
//	@Tags			institution-registration
//	@Produce		json
//	@Param			id	path		string	true	"Application ID"
//	@Success		200	{object}	InstitutionApplicationLinkResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse	"Already confirmed, anonymised or replaced by a newer application"
//	@Security		BearerAuth
//	@Router			/institution-applications/{id}/link [post]
func CreateInstitutionApplicationLink(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	id, err := httpx.PathParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}
	app, token, err := db.RenewInstitutionApplicationLinkAsAdmin(r.Context(), user.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	log.Info("admin created confirmation link for organisation application", "application_id", app.ID, "admin_id", user.ID)
	link := applicationLink(token)
	httpx.WriteJSON(w, http.StatusOK, InstitutionApplicationLinkResponse{
		Application:    app,
		Link:           link,
		To:             *app.Email,
		Subject:        applicationLinkSubject,
		Body:           applicationLinkMailBody(app, link),
		MailConfigured: mail.Configured(),
	})
}
