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
	mux.HandleFunc("GET /api/languages", GetLanguages)
	mux.HandleFunc("GET /api/languages/{code}", GetLocaleFile)

	mux.Handle("GET /api/platforms", httpx.RequireAuth(GetPlatforms))
	mux.Handle("GET /api/roles", httpx.RequireAuth(GetRoles))
	mux.Handle("GET /api/system/settings", httpx.RequireAuth(GetSystemSettings))

	// Games
	mux.Handle("GET /api/games", httpx.OptionalAuth(GetGames))
	mux.Handle("GET /api/games/{id}", httpx.OptionalAuth(GetGameByID))
	mux.Handle("GET /api/games/{id}/yaml", httpx.RequireAuth(GetGameYAML))
	mux.Handle("GET /api/games/{id}/sessions", httpx.RequireAuth(GetGameSessions))
	mux.Handle("POST /api/games/new", httpx.RequireAuth(CreateGame))
	mux.Handle("POST /api/games/{id}", httpx.RequireAuth(UpdateGame))
	mux.Handle("POST /api/games/{id}/clone", httpx.RequireAuth(CloneGame))
	mux.Handle("POST /api/games/{id}/sessions", httpx.RequireAuth(CreateGameSession))
	mux.Handle("PUT /api/games/{id}/yaml", httpx.RequireAuth(UpdateGameYAML))
	mux.Handle("DELETE /api/games/{id}", httpx.RequireAuth(DeleteGame))
	mux.Handle("GET /api/games/favourites", httpx.RequireAuth(GetFavouriteGames))
	mux.Handle("POST /api/games/{id}/favourite", httpx.RequireAuth(AddFavouriteGame))
	mux.Handle("DELETE /api/games/{id}/favourite", httpx.RequireAuth(RemoveFavouriteGame))
	mux.Handle("GET /api/games/{id}/available-keys", httpx.RequireAuth(GetAvailableKeys))

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
	mux.HandleFunc("POST /api/auth/logout", Logout)

	// Users
	mux.Handle("GET /api/users", httpx.RequireAuth(GetUsers))
	mux.Handle("GET /api/users/me", httpx.RequireAuth(GetCurrentUser))
	mux.Handle("GET /api/users/me/stats", httpx.RequireAuth(GetCurrentUserStats))
	mux.Handle("PATCH /api/users/me/language", httpx.RequireAuth(UpdateUserLanguage))
	mux.Handle("GET /api/users/{id}", httpx.RequireAuth(GetUserByID))
	mux.Handle("POST /api/users/{id}", httpx.RequireAuth(UpdateUserByID))
	mux.Handle("DELETE /api/users/{id}", httpx.RequireAuth(DeleteUser))
	mux.Handle("POST /api/users/{id}/role", httpx.RequireAuth(SetUserRole))
	mux.Handle("DELETE /api/users/{id}/role", httpx.RequireAuth(RemoveUserRole))
	if DevMode {
		mux.HandleFunc("POST /api/users/new", CreateUser)
		mux.HandleFunc("GET /api/users/{id}/jwt", GetUserJWT)
	}

	// Institutions
	mux.Handle("POST /api/institutions", httpx.RequireAuth(CreateInstitution))
	mux.Handle("GET /api/institutions", httpx.RequireAuth(ListInstitutions))
	mux.Handle("GET /api/institutions/{id}", httpx.RequireAuth(GetInstitution))
	mux.Handle("PATCH /api/institutions/{id}", httpx.RequireAuth(UpdateInstitution))
	mux.Handle("DELETE /api/institutions/{id}", httpx.RequireAuth(DeleteInstitution))
	mux.Handle("GET /api/institutions/{id}/members", httpx.RequireAuth(GetInstitutionMembers))
	mux.Handle("DELETE /api/institutions/{id}/members/{userID}", httpx.RequireAuth(RemoveInstitutionMember))
	mux.Handle("GET /api/institutions/{id}/apikeys", httpx.RequireAuth(GetInstitutionApiKeys))

	// Workshops
	mux.Handle("POST /api/workshops", httpx.RequireAuth(CreateWorkshop))
	mux.Handle("GET /api/workshops", httpx.RequireAuth(ListWorkshops))
	mux.Handle("GET /api/workshops/{id}", httpx.RequireAuth(GetWorkshop))
	mux.Handle("PATCH /api/workshops/{id}", httpx.RequireAuth(UpdateWorkshop))
	mux.Handle("DELETE /api/workshops/{id}", httpx.RequireAuth(DeleteWorkshop))
	mux.Handle("PUT /api/workshops/{id}/api-key", httpx.RequireAuth(SetWorkshopApiKey))
	mux.Handle("GET /api/workshops/{id}/events", httpx.RequireAuth(WorkshopEvents))
	mux.Handle("GET /api/workshops/participants/{participantId}/token", httpx.RequireAuth(GetParticipantToken))

	// Invites
	mux.Handle("GET /api/invites", httpx.RequireAuth(ListInvites))
	mux.Handle("GET /api/invites/all", httpx.RequireAuth(ListAllInvites))
	mux.Handle("GET /api/invites/institution/{institutionId}", httpx.RequireAuth(ListInvitesByInstitution))
	mux.Handle("GET /api/invites/{idOrToken}", httpx.OptionalAuth(GetInvite))            // Optional auth - allows anonymous to view invite details
	mux.Handle("POST /api/invites/{idOrToken}/accept", httpx.OptionalAuth(AcceptInvite)) // Optional auth - supports anonymous workshop invites
	mux.Handle("POST /api/invites/{id}/decline", httpx.RequireAuth(DeclineInvite))
	mux.Handle("POST /api/invites/institution", httpx.RequireAuth(CreateInstitutionInvite))
	mux.Handle("POST /api/invites/workshop", httpx.RequireAuth(CreateWorkshopInvite))
	mux.Handle("DELETE /api/invites/{id}", httpx.RequireAuth(RevokeInvite))

	// Sessions
	mux.Handle("GET /api/sessions", httpx.RequireAuth(GetUserSessions))
	mux.Handle("GET /api/sessions/{id}", httpx.RequireAuth(GetSession))
	mux.Handle("POST /api/sessions/{id}", httpx.RequireAuth(PostSessionAction))
	mux.Handle("PATCH /api/sessions/{id}", httpx.RequireAuth(UpdateSession))
	mux.Handle("DELETE /api/sessions/{id}", httpx.RequireAuth(DeleteSession))

	// Messages
	mux.Handle("GET /api/messages/{id}/stream", httpx.RequireAuth(GetMessageStream))
	// Image endpoint doesn't require auth header - browser <img> tags can't send headers
	// Access is verified internally via session ownership
	mux.HandleFunc("GET /api/messages/{id}/image", GetMessageImage)
	mux.HandleFunc("GET /api/messages/{id}/image/status", GetMessageImageStatus)

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
