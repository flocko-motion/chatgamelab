// package: routes / organisation self-registration HTTP handlers
// type:    logic
// job:     handles registering a new account as an organisation, upgrading an account, e-mail confirmation and resending the link; sends the mails.
// limits:  no route registration (-> router.go); persistence and rules in db/institution_registration.go; texts of the declaration in legal.
package routes

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"cgl/api/httpx"
	"cgl/constants"
	"cgl/db"
	"cgl/legal"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// InstitutionConsents are the checkboxes of the organisation form.
type InstitutionConsents struct {
	// ConfirmAdult: "Ich bin mindestens 18 Jahre alt".
	ConfirmAdult bool `json:"confirmAdult"`
	// AcceptDeclaration: the Einverständniserklärung, in the name of the organisation.
	AcceptDeclaration bool `json:"acceptDeclaration"`
	// DeclarationVersion is the version shown to the person; must be the current one.
	DeclarationVersion string `json:"declarationVersion"`
}

// RegisterInstitutionRequest registers a new account directly as an organisation.
type RegisterInstitutionRequest struct {
	obj.InstitutionDetails
	InstitutionConsents
	// AccountName is the display name of the new account (same rules as normal registration).
	AccountName string `json:"accountName"`
	// Language: ISO 639-1 code of the interface (optional).
	Language      string `json:"language"`
	AcceptTerms   bool   `json:"acceptTerms"`
	AcceptPrivacy bool   `json:"acceptPrivacy"`
}

// UpgradeInstitutionRequest upgrades the logged-in account to an organisation.
type UpgradeInstitutionRequest struct {
	obj.InstitutionDetails
	InstitutionConsents
}

// InstitutionApplicationResponse is returned after an application was stored.
type InstitutionApplicationResponse struct {
	User        *obj.User                   `json:"user,omitempty"`
	Application *obj.InstitutionApplication `json:"application"`
}

// MyInstitutionApplicationResponse is the open application of the logged-in account (nil if none).
type MyInstitutionApplicationResponse struct {
	Application *obj.InstitutionApplication `json:"application"`
}

// ConfirmInstitutionRequest carries the token from the confirmation link.
type ConfirmInstitutionRequest struct {
	Token string `json:"token"`
}

// ConfirmInstitutionResponse is the created organisation and the logged consent.
type ConfirmInstitutionResponse struct {
	Institution *obj.Institution        `json:"institution"`
	Consent     *obj.InstitutionConsent `json:"consent"`
}

func checkInstitutionConsents(c InstitutionConsents) *obj.AppError {
	if !c.ConfirmAdult || !c.AcceptDeclaration {
		return obj.NewAppError(obj.ErrCodeInstitutionConsentsRequired, "the 18+ confirmation and the declaration are required")
	}
	if c.DeclarationVersion != legal.CurrentDeclaration().Version {
		return obj.NewAppError(obj.ErrCodeDeclarationVersionOutdated, "the declaration was updated; please reload and read the current version")
	}
	return nil
}

// validateAccountName applies the account-name rules of the normal registration.
func validateAccountName(ctx context.Context, raw string) (string, *obj.AppError) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", obj.ErrValidation("accountName is required")
	}
	if utf8.RuneCountInString(name) > 24 {
		return "", obj.ErrNameTooLong("Name must be at most 24 characters")
	}
	if constants.IsProfane(name) {
		return "", obj.ErrProfaneName("This name is not allowed")
	}
	taken, err := db.IsNameTaken(ctx, name)
	if err != nil {
		return "", obj.ErrServerError("failed to check name availability")
	}
	if taken {
		return "", obj.NewAppError(obj.ErrCodeDuplicateName, "Name is already taken")
	}
	return name, nil
}

func writeErr(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*obj.AppError); ok {
		httpx.WriteAppError(w, appErr)
		return
	}
	httpx.WriteAppError(w, obj.ErrServerError(err.Error()))
}

// RegisterInstitution godoc
//
//	@Summary		Register a new account as an organisation
//	@Description	After the Auth0 login of a new person: creates the account (age group u18) and an organisation application, and e-mails a confirmation link. The organisation is created only when the link is confirmed. Requires the 18+ confirmation, terms, privacy policy and the current declaration.
//	@Tags			institution-registration
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterInstitutionRequest	true	"Registration form"
//	@Success		200		{object}	InstitutionApplicationResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse	"Name taken or e-mail belongs to another account"
//	@Security		BearerAuth
//	@Router			/auth/register-institution [post]
func RegisterInstitution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth0ID := httpx.Auth0IDFromContext(ctx)
	if auth0ID == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "Auth0 ID not found in token")
		return
	}
	var req RegisterInstitutionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if existing, _ := db.GetUserByAuth0ID(ctx, auth0ID); existing != nil {
		httpx.WriteAppError(w, obj.ErrConflict("account already registered; log in and use the upgrade instead"))
		return
	}
	if !req.AcceptTerms || !req.AcceptPrivacy {
		httpx.WriteAppError(w, obj.NewAppError(obj.ErrCodeInstitutionConsentsRequired, "terms of use and privacy policy must be accepted"))
		return
	}
	if appErr := checkInstitutionConsents(req.InstitutionConsents); appErr != nil {
		httpx.WriteAppError(w, appErr)
		return
	}
	name, appErr := validateAccountName(ctx, req.AccountName)
	if appErr != nil {
		httpx.WriteAppError(w, appErr)
		return
	}
	// Check the whole form before the account exists, so a rejected form leaves nothing behind.
	details, appErr := db.PrecheckInstitutionDetails(ctx, uuid.Nil, req.InstitutionDetails)
	if appErr != nil {
		httpx.WriteAppError(w, appErr)
		return
	}

	ageGroup := obj.AgeGroupU18
	email := details.Email
	user, err := db.CreateUser(ctx, name, &email, auth0ID, &ageGroup, httpx.PreferredLanguage(r, req.Language))
	if err != nil {
		log.Error("failed to create account for organisation registration", "auth0_id", auth0ID, "error", err)
		httpx.WriteAppError(w, obj.ErrServerError("failed to create account"))
		return
	}
	log.Info("account registered for organisation", "user_id", user.ID, "auth0_id", auth0ID)

	app, token, err := db.CreateInstitutionApplication(ctx, user.ID, obj.InstitutionApplicationRegister, details)
	if err != nil {
		writeErr(w, err)
		return
	}
	sendApplicationLinkMail(app, token)
	httpx.WriteJSON(w, http.StatusOK, InstitutionApplicationResponse{User: user, Application: app})
}

// UpgradeToInstitution godoc
//
//	@Summary		Upgrade the logged-in account to an organisation
//	@Description	For logged-in adult accounts without an organisation: stores an application and e-mails a confirmation link. Requires the 18+ confirmation and the current declaration.
//	@Tags			institution-registration
//	@Accept			json
//	@Produce		json
//	@Param			request	body		UpgradeInstitutionRequest	true	"Organisation form"
//	@Success		200		{object}	InstitutionApplicationResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse	"Account is not 18+"
//	@Failure		409		{object}	httpx.ErrorResponse	"Already in an organisation, name taken or e-mail of another account"
//	@Security		BearerAuth
//	@Router			/institution-applications [post]
func UpgradeToInstitution(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	var req UpgradeInstitutionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if appErr := db.CheckInstitutionEligibility(user); appErr != nil {
		httpx.WriteAppError(w, appErr)
		return
	}
	if appErr := checkInstitutionConsents(req.InstitutionConsents); appErr != nil {
		httpx.WriteAppError(w, appErr)
		return
	}
	app, token, err := db.CreateInstitutionApplication(r.Context(), user.ID, obj.InstitutionApplicationUpgrade, req.InstitutionDetails)
	if err != nil {
		writeErr(w, err)
		return
	}
	sendApplicationLinkMail(app, token)
	httpx.WriteJSON(w, http.StatusOK, InstitutionApplicationResponse{Application: app})
}

// GetMyInstitutionApplication godoc
//
//	@Summary		Get my open organisation application
//	@Description	Returns the open (unconfirmed) application of the logged-in account, or null.
//	@Tags			institution-registration
//	@Produce		json
//	@Success		200	{object}	MyInstitutionApplicationResponse
//	@Security		BearerAuth
//	@Router			/institution-applications/mine [get]
func GetMyInstitutionApplication(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	app, err := db.GetPendingInstitutionApplication(r.Context(), user.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, MyInstitutionApplicationResponse{Application: app})
}

// ResendInstitutionApplication godoc
//
//	@Summary		Resend the confirmation link
//	@Description	Creates a new confirmation link for the open application of the logged-in account and e-mails it. The previous link stops working.
//	@Tags			institution-registration
//	@Produce		json
//	@Success		200	{object}	MyInstitutionApplicationResponse
//	@Failure		404	{object}	httpx.ErrorResponse	"No open application"
//	@Security		BearerAuth
//	@Router			/institution-applications/resend [post]
func ResendInstitutionApplication(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	app, token, err := db.RenewInstitutionApplicationToken(r.Context(), user.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	sendApplicationLinkMail(app, token)
	httpx.WriteJSON(w, http.StatusOK, MyInstitutionApplicationResponse{Application: app})
}

// ConfirmInstitutionApplication godoc
//
//	@Summary		Confirm an organisation application
//	@Description	Called with the token from the e-mail link. Creates the organisation, makes the applicant its head, sets the youth-protection prompt restriction, logs the consent and e-mails the confirmation (with a copy to the operator).
//	@Tags			institution-registration
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ConfirmInstitutionRequest	true	"Token from the link"
//	@Success		200		{object}	ConfirmInstitutionResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"Unknown or used link"
//	@Failure		410		{object}	httpx.ErrorResponse	"Link expired"
//	@Router			/institution-applications/confirm [post]
func ConfirmInstitutionApplication(w http.ResponseWriter, r *http.Request) {
	var req ConfirmInstitutionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	result, err := db.ConfirmInstitutionApplication(r.Context(), req.Token)
	if err != nil {
		writeErr(w, err)
		return
	}
	log.Info("organisation registered", "institution_id", result.Institution.ID, "name", result.Institution.Name,
		"user_id", result.Consent.UserID, "declaration_version", result.Consent.DeclarationVersion)
	sendConsentConfirmationMails(result.Consent)
	httpx.WriteJSON(w, http.StatusOK, ConfirmInstitutionResponse{Institution: result.Institution, Consent: result.Consent})
}
