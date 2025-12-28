package httpx

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cgl/api/auth"
	"cgl/db"
	"cgl/obj"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/google/uuid"
)

// ctxKeyUser is the context key for the authenticated user
type ctxKeyUser struct{}

// WithUser returns a new request with the user attached to its context
func WithUser(r *http.Request, user *obj.User) *http.Request {
	ctx := context.WithValue(r.Context(), ctxKeyUser{}, user)
	return r.WithContext(ctx)
}

// UserFromRequest returns the authenticated user from the request context.
//
// This function should ONLY be called in handlers wrapped with RequireAuth middleware.
// If the user is nil, it panics with a clear error message indicating a programming error
// (i.e., the handler was not properly wrapped with RequireAuth).
//
// Example usage:
//
//	func MyProtectedHandler(w http.ResponseWriter, r *http.Request) {
//		user := httpx.UserFromRequest(r) // Safe: RequireAuth guarantees user is non-nil
//		// ... use user
//	}
//
//	mux.Handle("/api/protected", httpx.RequireAuth(MyProtectedHandler))
func UserFromRequest(r *http.Request) *obj.User {
	u, _ := r.Context().Value(ctxKeyUser{}).(*obj.User)
	if u == nil {
		panic("UserFromRequest called but user is nil - handler must be wrapped with RequireAuth middleware")
	}
	return u
}

// MaybeUserFromRequest returns the authenticated user from the request context, or nil if not authenticated.
//
// Use this function in handlers wrapped with OptionalAuth where the user may or may not be present.
// Always check if the returned user is nil before using it.
//
// Example usage:
//
//	func MyOptionalAuthHandler(w http.ResponseWriter, r *http.Request) {
//		user := httpx.MaybeUserFromRequest(r)
//		if user != nil {
//			// User is authenticated, show personalized content
//		} else {
//			// User is not authenticated, show public content
//		}
//	}
//
//	mux.Handle("/api/optional", httpx.OptionalAuth(MyOptionalAuthHandler))
func MaybeUserFromRequest(r *http.Request) *obj.User {
	u, _ := r.Context().Value(ctxKeyUser{}).(*obj.User)
	return u
}

// RequireUser returns a middleware that enforces authentication.
// If no user is attached to the request context, it returns 401 Unauthorized.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if MaybeUserFromRequest(r) == nil {
			WriteError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CustomClaims contains custom data we want from the Auth0 token
type CustomClaims struct {
	Scope string `json:"scope"`
}

func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

func (c CustomClaims) HasScope(expectedScope string) bool {
	scopes := strings.Split(c.Scope, " ")
	for _, s := range scopes {
		if s == expectedScope {
			return true
		}
	}
	return false
}

// Authenticate returns a middleware that attempts to authenticate the request.
// It supports both CGL dev JWTs (HS256) and Auth0 JWTs (RS256).
// - If no Authorization header: passes through (user = nil)
// - If valid token: loads user from DB and attaches to context
// - If invalid token: returns 401
func Authenticate(next http.Handler) http.Handler {
	// Set up Auth0 validator (lazily initialized)
	var auth0Middleware *jwtmiddleware.JWTMiddleware
	var auth0MiddlewareInitialized bool

	initAuth0 := func() *jwtmiddleware.JWTMiddleware {
		if auth0MiddlewareInitialized {
			return auth0Middleware
		}
		auth0MiddlewareInitialized = true

		domain := os.Getenv("AUTH0_DOMAIN")
		if domain == "" {
			log.Println("[Auth] AUTH0_DOMAIN not set, Auth0 authentication disabled")
			return nil
		}

		issuerURL, err := url.Parse("https://" + domain + "/")
		if err != nil {
			log.Printf("[Auth] Failed to parse Auth0 issuer URL: %v", err)
			return nil
		}

		provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

		jwtValidator, err := validator.New(
			provider.KeyFunc,
			validator.RS256,
			issuerURL.String(),
			[]string{os.Getenv("AUTH0_AUDIENCE")},
			validator.WithCustomClaims(func() validator.CustomClaims {
				return &CustomClaims{}
			}),
			validator.WithAllowedClockSkew(time.Minute),
		)
		if err != nil {
			log.Printf("[Auth] Failed to set up Auth0 validator: %v", err)
			return nil
		}

		auth0Middleware = jwtmiddleware.New(
			jwtValidator.ValidateToken,
			jwtmiddleware.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
				log.Printf("[Auth] Auth0 JWT validation error: %v", err)
				WriteError(w, http.StatusUnauthorized, "Invalid token")
			}),
		)
		return auth0Middleware
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// No Authorization header - pass through without user
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Try CGL dev JWT first (HS256, signed with JWT_SECRET)
		if userId, valid := auth.ValidateToken(r); valid {
			if userId == "" {
				log.Printf("[Auth] CGL JWT has empty 'sub' claim")
				WriteError(w, http.StatusUnauthorized, "Invalid token: missing user ID")
				return
			}

			user, err := db.GetUserByID(r.Context(), uuid.MustParse(userId))
			if err != nil {
				log.Printf("[Auth] CGL JWT user not found: %s", userId)
				WriteError(w, http.StatusUnauthorized, "User not found")
				return
			}

			log.Printf("[Auth] CGL JWT authenticated user: %s (%s)", userId, user.Name)
			next.ServeHTTP(w, WithUser(r, user))
			return
		}

		// Try Auth0 JWT (RS256)
		auth0 := initAuth0()
		if auth0 == nil {
			// Auth0 not configured and CGL token invalid
			WriteError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		// Use Auth0 middleware to validate, then extract user
		auth0.CheckJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenObj := r.Context().Value(jwtmiddleware.ContextKey{})
			if tokenObj == nil {
				next.ServeHTTP(w, r)
				return
			}

			token := tokenObj.(*validator.ValidatedClaims)
			auth0ID := token.RegisteredClaims.Subject
			if auth0ID == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Load or create user by Auth0 ID
			user, err := db.GetUserByAuth0ID(r.Context(), auth0ID)
			if err != nil {
				// Auto-create user on first login
				user, err = db.CreateUser(r.Context(), "Unnamed Auth0 User", nil, auth0ID)
				if err != nil {
					log.Printf("[Auth] Failed to create user for Auth0 ID %s: %v", auth0ID, err)
					WriteError(w, http.StatusInternalServerError, "Failed to create user")
					return
				}
			}

			log.Printf("[Auth] Auth0 authenticated user: %s (%s)", auth0ID, user.Name)
			next.ServeHTTP(w, WithUser(r, user))
		})).ServeHTTP(w, r)
	})
}

// OptionalAuth wraps a handler with authentication that is optional (user may be nil)
// Use this for endpoints that work without auth but may have enhanced functionality with auth
func OptionalAuth(h http.HandlerFunc) http.Handler {
	return Authenticate(h)
}

// RequireAuth wraps a handler with authentication that is required (user guaranteed non-nil)
// Use this for endpoints that require a logged-in user
func RequireAuth(h http.HandlerFunc) http.Handler {
	return Authenticate(RequireUser(h))
}
