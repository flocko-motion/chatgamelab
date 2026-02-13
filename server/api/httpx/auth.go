package httpx

import (
	"cgl/api/auth"
	"cgl/db"
	"cgl/log"
	"cgl/obj"
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/google/uuid"
)

// ctxKeyUser is the context key for the authenticated user
type ctxKeyUser struct{}

// Package-level Auth0 validator (lazily initialized)
var (
	auth0Validator            *validator.Validator
	auth0ValidatorInitialized bool
)

// getAuth0Validator returns the Auth0 JWT validator, initializing it if needed
func getAuth0Validator() *validator.Validator {
	if auth0ValidatorInitialized {
		return auth0Validator
	}
	auth0ValidatorInitialized = true

	domain := os.Getenv("AUTH0_DOMAIN")
	if domain == "" {
		log.Debug("auth0 authentication disabled", "reason", "AUTH0_DOMAIN not set")
		return nil
	}

	issuerURL, err := url.Parse("https://" + domain + "/")
	if err != nil {
		log.Error("failed to parse auth0 issuer URL", "domain", domain, "error", err)
		return nil
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	auth0Validator, err = validator.New(
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
		log.Error("failed to set up auth0 validator", "error", err)
		return nil
	}
	return auth0Validator
}

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
	Scope    string `json:"scope"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
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
// It supports three authentication methods (checked in order):
// 1. Participant tokens (Bearer participant-xxx) - for workshop participants
// 2. CGL dev JWTs (HS256, signed with JWT_SECRET) - for development
// 3. Auth0 JWTs (RS256) - for production users
//
// Behavior:
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
			log.Debug("auth0 authentication disabled", "reason", "AUTH0_DOMAIN not set")
			return nil
		}

		issuerURL, err := url.Parse("https://" + domain + "/")
		if err != nil {
			log.Error("failed to parse auth0 issuer URL", "domain", domain, "error", err)
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
			log.Error("failed to set up auth0 validator", "error", err)
			return nil
		}
		auth0Middleware = jwtmiddleware.New(
			jwtValidator.ValidateToken,
			jwtmiddleware.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
				log.Debug("auth0 JWT validation failed", "error", err, "path", r.URL.Path)
				WriteError(w, http.StatusUnauthorized, "Invalid token")
			}),
		)
		return auth0Middleware
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Collect token from various sources
		var tokenString string
		var tokenSource string

		// 1. Check Authorization header
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			tokenSource = "header"
		}

		// 2. Check for session cookie as fallback
		if tokenString == "" {
			if cookie, err := r.Cookie("cgl_session"); err == nil && cookie.Value != "" {
				tokenString = cookie.Value
				tokenSource = "cookie"
			}
		}

		// 3. Check for token query parameter (for SSE endpoints where EventSource can't send headers)
		if tokenString == "" {
			if token := r.URL.Query().Get("token"); token != "" {
				tokenString = token
				tokenSource = "query"
			}
		}

		// No token found - pass through without user
		if tokenString == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for participant token (prefixed with "participant-")
		if strings.HasPrefix(tokenString, "participant-") {
			// Lookup user by participant token
			// SQL query validates: user exists, has participant role, linked to active workshop
			user, err := db.GetUserByParticipantToken(r.Context(), tokenString)
			if err != nil {
				// Check for specific error codes
				if authErr, ok := err.(*db.ParticipantAuthError); ok {
					log.Debug("participant auth failed", "code", authErr.Code, "error", authErr.Message)
					if authErr.Code == "workshop_inactive" {
						WriteErrorWithCode(w, http.StatusForbidden, "auth_workshop_inactive", "Workshop is inactive")
						return
					}
				}
				log.Debug("participant token invalid", "error", err)
				// For OptionalAuth, continue without user instead of returning 401
				// This allows invite acceptance to work with invalid/old tokens
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, WithUser(r, user))
			return
		}

		// Try CGL dev JWT first (HS256, signed with JWT_SECRET)
		// Use ValidateTokenString to handle tokens from any source (header, cookie, query param)
		if userId, valid := auth.ValidateTokenString(tokenString); valid {
			_ = tokenSource // silence unused warning in production
			if userId == "" {
				log.Warn("CGL JWT has empty sub claim")
				WriteError(w, http.StatusUnauthorized, "Invalid token: missing user ID")
				return
			}

			user, err := db.GetUserByID(r.Context(), uuid.MustParse(userId))
			if err != nil {
				log.Debug("CGL JWT user not found", "user_id", userId, "error", err)
				WriteError(w, http.StatusUnauthorized, "User not found")
				return
			}

			user = db.CheckAndPromoteAdmin(r.Context(), user)
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

		// Auth0 CheckJWT reads from Authorization header. If the token came from
		// a query param (e.g. SSE EventSource), inject it into the header so
		// Auth0 middleware can find it.
		if tokenSource == "query" && r.Header.Get("Authorization") == "" {
			r = r.Clone(r.Context())
			r.Header.Set("Authorization", "Bearer "+tokenString)
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

			// Load user by Auth0 ID - do NOT auto-create
			user, err := db.GetUserByAuth0ID(r.Context(), auth0ID)
			if err != nil {
				// User not registered - frontend will get email/name from Auth0 directly
				log.Debug("auth0 user not registered", "auth0_id", auth0ID)
				WriteUserNotRegistered(w, auth0ID)
				return
			}

			user = db.CheckAndPromoteAdmin(r.Context(), user)
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

// RequireAuth0Token validates Auth0 token but allows unregistered users.
// Use this for endpoints like registration where user is authenticated via Auth0
// but may not yet exist in our database. Sets Auth0ID in context.
func RequireAuth0Token(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract and validate Auth0 token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			WriteError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Check if it's a CGL dev JWT first
		if strings.HasPrefix(tokenString, "cgl-") {
			WriteError(w, http.StatusForbidden, "Dev tokens cannot be used for registration")
			return
		}

		// Validate Auth0 token
		jwtValidator := getAuth0Validator()
		if jwtValidator == nil {
			WriteError(w, http.StatusInternalServerError, "Auth0 not configured")
			return
		}

		tokenObj, err := jwtValidator.ValidateToken(r.Context(), tokenString)
		if err != nil {
			log.Debug("auth0 token validation failed", "error", err)
			WriteError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		claims, ok := tokenObj.(*validator.ValidatedClaims)
		if !ok {
			WriteError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		auth0ID := claims.RegisteredClaims.Subject
		if auth0ID == "" {
			WriteError(w, http.StatusUnauthorized, "Invalid token: missing subject")
			return
		}

		// Store Auth0 ID in context for the handler to use
		ctx := context.WithValue(r.Context(), auth0IDContextKey, auth0ID)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// auth0IDContextKey is the context key for storing Auth0 ID
type auth0IDContextKeyType string

const auth0IDContextKey auth0IDContextKeyType = "auth0_id"

// Auth0IDFromContext retrieves the Auth0 ID from the request context
func Auth0IDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(auth0IDContextKey).(string); ok {
		return id
	}
	return ""
}
