// package: routes / dev-mode helpers
// type:    logic
// job:     dev-only endpoints to test registration locally: a token that acts like a fresh Auth0 login, and the mails sent so far.
// limits:  registered only when DEV_MODE is on (-> router.go).
package routes

import (
	"net/http"

	"cgl/api/auth"
	"cgl/api/httpx"
	"cgl/mail"

	"github.com/google/uuid"
)

// DevSignupTokenResponse is a token for a new, not yet registered person.
type DevSignupTokenResponse struct {
	Token   string `json:"token"`
	Auth0ID string `json:"auth0Id"`
}

// CreateDevSignupToken godoc
//
//	@Summary		Dev: token for a new person
//	@Description	DEV_MODE only. Returns a token that behaves like the Auth0 login of a person who has no account yet.
//	@Tags			dev
//	@Produce		json
//	@Success		200	{object}	DevSignupTokenResponse
//	@Router			/dev/signup-token [post]
func CreateDevSignupToken(w http.ResponseWriter, r *http.Request) {
	sub := httpx.DevSignupSubjectPrefix + uuid.NewString()
	token, _, err := auth.GenerateToken(sub)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, DevSignupTokenResponse{Token: token, Auth0ID: sub})
}

// GetDevMails godoc
//
//	@Summary		Dev: mails sent so far
//	@Description	DEV_MODE only. The last mails the server sent, newest first.
//	@Tags			dev
//	@Produce		json
//	@Success		200	{array}	mail.Message
//	@Router			/dev/mails [get]
func GetDevMails(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, mail.Recent())
}
