package routes

import (
	"net/http"

	"cgl/api/httpx"
)

// DevMode controls whether dev-only routes are registered
var DevMode = false

// NewMux creates a new http.ServeMux with all API routes registered.
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()

	// Public endpoints (no auth at all) - GET
	mux.HandleFunc("GET /api/status", GetStatus)
	mux.HandleFunc("GET /api/version", GetVersion)
	mux.HandleFunc("GET /api/platforms", GetPlatforms)
	mux.HandleFunc("GET /api/languages", GetLanguages)
	mux.HandleFunc("GET /api/languages/{code}", GetLocaleFile)

	// Games
	mux.Handle("GET /api/games", httpx.OptionalAuth(GetGames))
	mux.Handle("GET /api/games/{id}", httpx.OptionalAuth(GetGameByID))
	mux.Handle("GET /api/games/{id}/yaml", httpx.RequireAuth(GetGameYAML))
	mux.Handle("GET /api/games/{id}/sessions", httpx.RequireAuth(GetGameSessions))
	mux.Handle("POST /api/games/new", httpx.RequireAuth(CreateGame))
	mux.Handle("POST /api/games/{id}", httpx.RequireAuth(UpdateGame))
	mux.Handle("POST /api/games/{id}/sessions", httpx.RequireAuth(CreateGameSession))
	mux.Handle("PUT /api/games/{id}/yaml", httpx.RequireAuth(UpdateGameYAML))
	mux.Handle("DELETE /api/games/{id}", httpx.RequireAuth(DeleteGame))

	// API Keys
	mux.Handle("GET /api/apikeys", httpx.RequireAuth(GetApiKeys))
	mux.Handle("GET /api/apikeys/{id}", httpx.RequireAuth(GetApiKeyByID))
	mux.Handle("POST /api/apikeys/new", httpx.RequireAuth(CreateApiKey))
	mux.Handle("POST /api/apikeys/{id}/shares", httpx.RequireAuth(ShareApiKey))
	// Backward compatibility: previously used POST /api/apikeys/{id} for sharing
	mux.Handle("POST /api/apikeys/{id}", httpx.RequireAuth(ShareApiKey))
	mux.Handle("PATCH /api/apikeys/{id}", httpx.RequireAuth(UpdateApiKey))
	mux.Handle("DELETE /api/apikeys/{id}", httpx.RequireAuth(DeleteApiKey))

	// Auth
	mux.HandleFunc("GET /api/auth/check-name", CheckNameAvailability)
	mux.Handle("POST /api/auth/register", httpx.RequireAuth0Token(RegisterUser))

	// Users
	mux.Handle("GET /api/users", httpx.RequireAuth(GetUsers))
	mux.Handle("GET /api/users/me", httpx.RequireAuth(GetCurrentUser))
	mux.Handle("GET /api/users/{id}", httpx.RequireAuth(GetUserByID))
	mux.Handle("POST /api/users/{id}", httpx.RequireAuth(UpdateUserByID))
	if DevMode {
		mux.HandleFunc("POST /api/users/new", CreateUser)
		mux.HandleFunc("GET /api/users/{id}/jwt", GetUserJWT)
	}

	// Sessions
	mux.Handle("GET /api/sessions/{id}", httpx.OptionalAuth(GetSession))
	mux.Handle("POST /api/sessions/{id}", httpx.OptionalAuth(PostSessionAction))

	// Messages
	mux.Handle("GET /api/messages/{id}/stream", httpx.OptionalAuth(GetMessageStream))

	// Admin
	mux.Handle("POST /api/restart", httpx.RequireAuth(PostRestart))

	return mux
}

// Handler returns the full HTTP handler with all middleware applied
func Handler() http.Handler {
	mux := NewMux()

	// Apply global middleware (outermost to innermost)
	return httpx.Chain(mux,
		httpx.Recover,
		httpx.Logging,
		httpx.CORS,
		httpx.NoCache,
	)
}
