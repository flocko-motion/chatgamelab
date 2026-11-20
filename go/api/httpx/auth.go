// package: httpx / HTTP auth middleware
// type:    wiring
// job:     attaches the account behind a request, resolved from schemaf's subject, to the context of an http.Handler
// limits:  schemaf authenticates the request (its endpoint's AuthLevel); resolution and the registration lock live in api/auth
package httpx

import (
	"cgl/api/auth"
	"cgl/obj"
	"context"
	"errors"
	"net/http"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// ctxKeyUser is the context key for the resolved user
type ctxKeyUser struct{}

// WithUser returns a new request with the user attached to its context
func WithUser(r *http.Request, user *obj.User) *http.Request {
	ctx := context.WithValue(r.Context(), ctxKeyUser{}, user)
	return r.WithContext(ctx)
}

// UserFromRequest returns the user RequireAuth attached. It panics when the
// handler is not wrapped with RequireAuth: a programming error.
func UserFromRequest(r *http.Request) *obj.User {
	u, _ := r.Context().Value(ctxKeyUser{}).(*obj.User)
	if u == nil {
		panic("UserFromRequest called but user is nil - handler must be wrapped with RequireAuth middleware")
	}
	return u
}

// MaybeUserFromRequest returns the user OptionalAuth attached, or nil for an
// anonymous caller.
func MaybeUserFromRequest(r *http.Request) *obj.User {
	return UserFromContext(r.Context())
}

// UserFromContext is the user a request's context carries, or nil; for code
// that is handed the context rather than the request.
func UserFromContext(ctx context.Context) *obj.User {
	u, _ := ctx.Value(ctxKeyUser{}).(*obj.User)
	return u
}

// RequireAuth resolves the signed-in account (-> auth.User) and refuses the
// request without one.
func RequireAuth(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := auth.User(r.Context())
		if err != nil {
			writeAuthError(w, err)
			return
		}
		h(w, WithUser(r, user))
	})
}

// OptionalAuth resolves the signed-in account if there is one (-> auth.MaybeUser);
// an anonymous caller passes with no user.
func OptionalAuth(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := auth.MaybeUser(r.Context())
		if err != nil {
			writeAuthError(w, err)
			return
		}
		if user != nil {
			r = WithUser(r, user)
		}
		h(w, r)
	})
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrNoCaller):
		WriteError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, schemafapi.ErrForbidden):
		WriteError(w, http.StatusForbidden, err.Error())
	default:
		WriteError(w, http.StatusInternalServerError, err.Error())
	}
}
