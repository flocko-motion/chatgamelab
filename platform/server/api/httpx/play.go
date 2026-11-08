// package: httpx / the playback-engine's player cookie
// type:    logic
// job:     sets and clears the cgl_play cookie and authenticates requests to the engine's routes from it
// limits:  token signing lives in api/auth; who may reach which session is decided in api/routes
package httpx

import (
	"context"
	"net/http"
	"time"

	"cgl/api/auth"
	"cgl/db"
	"cgl/obj"

	"github.com/google/uuid"
)

// PlayCookie names the cookie that carries a signed-in player's identity to
// the playback-engine, whose requests send cookies and never an
// Authorization header.
const PlayCookie = "cgl_play"

// EngineMount is where the engine's routes are mounted below the API's path.
const EngineMount = "/engine"

// playCookiePath scopes the cookie to the engine's routes, so the browser
// sends it to playback calls only.
func playCookiePath() string {
	return getCookiePath() + EngineMount
}

// SetPlayCookie sets a short-lived, engine-scoped play token for the user.
// SameSite=Strict keeps another website from sending it with the engine's POSTs.
func SetPlayCookie(w http.ResponseWriter, r *http.Request, userID uuid.UUID) (time.Time, error) {
	token, expiresAt, err := auth.GeneratePlayToken(userID.String())
	if err != nil {
		return time.Time{}, err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     PlayCookie,
		Value:    token,
		Path:     playCookiePath(),
		MaxAge:   int(auth.PlayTokenLifetime.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteStrictMode,
	})
	return expiresAt, nil
}

// ClearPlayCookie removes the play token, as signing out does.
func ClearPlayCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     PlayCookie,
		Value:    "",
		Path:     playCookiePath(),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// PlayAuth authenticates a request to the engine's routes: a valid play
// cookie names the player; otherwise the platform's usual authentication
// applies, as for a participant's cookie or a headless client's header. A
// stale or forged play cookie counts as no identity.
func PlayAuth(next http.Handler) http.Handler {
	authenticated := Authenticate(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(PlayCookie); err == nil {
			if sub, ok := auth.ValidatePlayToken(cookie.Value); ok {
				if id, err := uuid.Parse(sub); err == nil {
					if user, err := db.GetUserByID(r.Context(), id); err == nil {
						next.ServeHTTP(w, WithUser(r, user))
						return
					}
				}
			}
		}
		authenticated.ServeHTTP(w, r)
	})
}

// UserFromContext is the authenticated user a request's context carries, or
// nil; for code that is handed the context rather than the request.
func UserFromContext(ctx context.Context) *obj.User {
	u, _ := ctx.Value(ctxKeyUser{}).(*obj.User)
	return u
}
