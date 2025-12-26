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

	// Public endpoints (no auth at all)
	mux.HandleFunc("GET /api/status", GetStatus)
	mux.HandleFunc("GET /api/version", GetVersion)

	// Games - list (auth optional for enhanced results)
	mux.Handle("GET /api/games", httpx.OptionalAuth(GetGames))

	// Games by ID - GET is optional auth, POST/DELETE require auth
	mux.Handle("GET /api/games/{id}", httpx.OptionalAuth(GetGameByID))
	mux.Handle("POST /api/games/{id}", httpx.RequireAuth(UpdateGame))
	mux.Handle("DELETE /api/games/{id}", httpx.RequireAuth(DeleteGame))

	// Games YAML export/import
	mux.Handle("GET /api/games/{id}/yaml", httpx.RequireAuth(GetGameYAML))
	mux.Handle("PUT /api/games/{id}/yaml", httpx.RequireAuth(UpdateGameYAML))

	// Games - create new
	mux.Handle("POST /api/games/new", httpx.RequireAuth(CreateGame))

	// Game sessions
	mux.Handle("GET /api/games/{id}/sessions", httpx.RequireAuth(GetGameSessions))
	mux.Handle("POST /api/games/{id}/sessions", httpx.RequireAuth(CreateGameSession))

	// API Keys - all require auth
	mux.Handle("GET /api/apikeys", httpx.RequireAuth(GetApiKeys))
	mux.Handle("POST /api/apikeys/new", httpx.RequireAuth(CreateApiKey))
	mux.Handle("GET /api/apikeys/{id}", httpx.RequireAuth(GetApiKeyByID))
	mux.Handle("POST /api/apikeys/{id}", httpx.RequireAuth(ShareApiKey))
	mux.Handle("PATCH /api/apikeys/{id}", httpx.RequireAuth(UpdateApiKey))
	mux.Handle("DELETE /api/apikeys/{id}", httpx.RequireAuth(DeleteApiKey))

	// Users - all require auth
	mux.Handle("GET /api/users", httpx.RequireAuth(GetUsers))
	mux.Handle("GET /api/users/me", httpx.RequireAuth(GetCurrentUser))
	mux.Handle("GET /api/users/{id}", httpx.RequireAuth(GetUserByID))
	mux.Handle("POST /api/users/{id}", httpx.RequireAuth(UpdateUserByID))

	// Sessions - optional auth (can play without login)
	mux.Handle("GET /api/sessions/{id}", httpx.OptionalAuth(GetSession))
	mux.Handle("POST /api/sessions/{id}", httpx.OptionalAuth(PostSessionAction))

	// SSE streaming - optional auth
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
