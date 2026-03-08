# Architecture

**Analysis Date:** 2026-03-08

## Pattern Overview

**Overall:** Full-stack monorepo — Go REST API backend + React SPA frontend, deployed via Docker/Nginx

**Key Characteristics:**
- Strict separation between `server/` (Go) and `web/` (TypeScript/React)
- Backend uses plain `net/http` with no framework; manual middleware chaining on a `http.ServeMux`
- Frontend is a feature-sliced SPA using TanStack Router (file-based routing) and TanStack Query (server state)
- AI interaction is the core domain: every game session runs a multi-phase AI pipeline (ExecuteAction → ExpandStory → GenerateImage → GenerateAudio)
- Real-time updates flow via Server-Sent Events (SSE) — backend pushes stream chunks, frontend consumes them
- Authentication supports three token types: Auth0 JWT (RS256), CGL dev JWT (HS256), and participant tokens (prefixed strings)

## Layers

**CLI Entry Point:**
- Purpose: Cobra-based CLI with subcommands for server startup, seeding, and admin tasks
- Location: `server/cmd/`
- Contains: `server/`, `ai/`, `apikey/`, `game/`, `healthcheck/`, `institution/`, `invite/`, `lang/`, `user/`, `workshop/` subcommands; `root.go`
- Depends on: `api`, `db`, `game`, `config`
- Used by: shell scripts (`run-dev.sh`, `run-prod.sh`, `run-seed.sh`, `bin/`)

**HTTP API Layer:**
- Purpose: Route registration, middleware application, JSON request/response marshaling
- Location: `server/api/`
- Contains: `server.go` (startup orchestration), `routes/router.go` (mux registration), `routes/*.go` (handlers), `httpx/` (middleware, auth helpers, response writers), `auth/` (JWT generation for dev tokens), `client/` (typed Go API client used by CLI tools), `apiclient/`
- Depends on: `db`, `game`, `obj`, `events`, `lang`
- Used by: CLI server subcommand

**Middleware Stack:**
- Location: `server/api/httpx/middleware.go`, `server/api/httpx/auth.go`
- Global order (outermost→innermost): `Recover → Logging → CORS → NoCache`
- Per-route auth: `RequireAuth(fn)`, `OptionalAuth(fn)`, `RequireAuth0Token(fn)` compose the `Authenticate` middleware with optional `RequireUser` enforcement

**Game Engine (Core Domain):**
- Purpose: Orchestrates multi-phase AI interactions for game sessions
- Location: `server/game/`
- Contains: `session_play.go` (core `DoSessionAction` + `DoSessionActionWithFallback`), `session_creation.go`, `session_lock.go` (per-session mutex), `resolve_api_key.go`, `ai/` (platform interface + implementations), `stream/` (SSE stream registry), `imagecache/`, `templates/` (AI prompts + JSON response schemas), `status/`, `theme.go`, `translate.go`
- Depends on: `db`, `obj`, `lang`, `functional`, `events`
- Used by: `api/routes` handlers

**AI Platform Abstraction:**
- Purpose: Unified interface for all AI providers; pluggable implementations
- Location: `server/game/ai/ai.go` (interface), `server/game/ai/openai/`, `server/game/ai/mistral/`, `server/game/ai/mock/`
- Interface methods: `ExecuteAction`, `ExpandStory`, `GenerateImage`, `GenerateAudio`, `Translate`, `ListModels`, `ToolQuery`, `TranscribeAudio`, `ResolveModelInfo`

**Data / Object Layer:**
- Purpose: Shared domain types and all database access
- Location: `server/obj/` (domain structs, HTTP errors, error codes), `server/db/` (all DB functions, migrations, sqlc-generated code)
- Key files: `obj/structs.go` (User, Game, GameSession, GameSessionMessage, ApiKey, Institution, Workshop, etc.), `db/init.go` (connection + auto-migration), `db/sqlc/` (generated queries), `db/queries/*.sql`, `db/migrations/*.sql`, `db/schema.sql`
- Depends on: PostgreSQL via `lib/pq`, `sqlc`
- Used by: all packages

**SSE Event Broker:**
- Purpose: Workshop-scoped SSE event fan-out (game create/update/delete, workshop updates)
- Location: `server/events/broker.go`
- Pattern: Global singleton `Broker` with per-workshop subscriber channels (map[workshopID]map[chan Event]struct{})
- Used by: `api/routes/workshop_events.go`

**Support Packages:**
- `server/functional/` — generic utilities: `Ptr`, `Deref`, `First`, `RequireEnv`, etc.
- `server/lang/` — locale file loading, language name lookup
- `server/log/` — structured `slog`-based logging wrapper
- `server/telemetry/` — Sentry SDK integration
- `server/constants/` — shared application constants

**Frontend Provider Tree:**
- Purpose: Global React context providers wrapping the entire application
- Location: `web/src/providers/`
- Mount order: `QueryClientProvider → Auth0Provider → MantineProvider → ErrorBoundary → ModalsProvider → Notifications → GlobalErrorModal → AuthProvider → WorkshopModeProvider → RouterProvider`
- Key files: `AppProviders.tsx`, `AuthProvider.tsx`, `WorkshopModeProvider.tsx`, `auth/tokenStorage.ts`, `auth/useTokenManager.ts`, `auth/useBackendUser.ts`

**Frontend Routing:**
- Purpose: File-based routing; route files map 1:1 to URL segments
- Location: `web/src/routes/`
- Root layout: `web/src/routes/__root.tsx` — auth guards, participant redirects, layout variant selection, nav construction
- Generated: `web/src/routeTree.gen.ts` (do not edit manually)
- Route constants: `web/src/common/routes/routes.ts`

**Frontend Features:**
- Purpose: Feature-sliced business logic and UI components
- Location: `web/src/features/`
- Key features: `game-player-v2/` (core gameplay), `games/` (browse/create/edit), `auth/` (login/registration), `admin/`, `dashboard/`, `my-organization/`, `my-workshop/`, `play/`, `profile/`, `settings/`, `api-keys/`, `debug/`
- Each feature has: `components/`, optionally `hooks/`, `lib/`, `context/`, `types.ts`

**Frontend API Client:**
- Purpose: Type-safe HTTP client auto-generated from OpenAPI/swagger spec
- Location: `web/src/api/`
- Key files: `generated/` (generated `Api` class), `client/http.ts` (config factory), `hooks/` (TanStack Query hooks per resource), `useAuthenticatedApi.ts` (hook returning auth-injected client), `queryKeys.ts`

## Data Flow

**Game Session Action (core flow):**

1. Player sends action: `POST /api/sessions/{id}` (authenticated) or `POST /api/play/{token}/sessions/{id}` (guest)
2. Route handler in `server/api/routes/sessions.go` loads session + resolves API key candidates, calls `game.DoSessionActionWithFallback`
3. `server/game/session_play.go::DoSessionAction` acquires per-session mutex (serializes AI calls for conversation continuity)
4. Phase 0a: Audio transcription if player sent voice input via `platform.TranscribeAudio`
5. Phase 0b: Rephrase player input in third person via `platform.ToolQuery` (fast, blocking)
6. Phase 1: `platform.ExecuteAction` — blocking, returns structured JSON (plot outline, status fields, image prompt) with game-specific JSON schema enforcement
7. Handler returns immediately with placeholder `GameSessionMessage` (Stream=true); client connects to SSE
8. Phase 2 (goroutine): `platform.ExpandStory` — streams prose narrative chunks to `stream.Registry` channel; holds session lock until complete
9. Phase 3 (goroutine, parallel to Phase 2): `platform.GenerateImage` — generates image, saves via `imagecache`, persists to DB
10. Phase 4 (after Phase 2 completes): `platform.GenerateAudio` — TTS narration, streams audio chunks
11. Frontend SSE consumer at `GET /api/messages/{id}/stream` reads from `stream.Registry` channel and writes `text/event-stream` events until text+image+audio all signal done

**Authentication Flow:**

1. User authenticates via Auth0 in browser → receives RS256 JWT (stored in localStorage via Auth0 SDK)
2. `AuthProvider` obtains token via `useAuth0`, calls `GET /api/users/me` to load backend user
3. All API calls attach `Authorization: Bearer <token>` via `createAuthenticatedApiConfig`
4. Server `Authenticate` middleware resolves token type: `participant-` prefix → DB lookup; `cgl-` prefix → HS256 validate; else → Auth0 RS256 validate via JWKS
5. Resolved user is attached to request context; handlers call `httpx.UserFromRequest(r)` (panics if missing — programming error guard)
6. SSE endpoints accept token via query param (EventSource cannot set headers)

**Workshop Events (SSE):**

1. Staff/head connects to `GET /api/workshops/{id}/events`
2. Handler subscribes a channel to `events.Broker` for the workshop UUID
3. Game mutations (create/update/delete in that workshop) call `events.GetBroker().Publish(...)`
4. Broker fans out to all subscribers; handler writes `text/event-stream` to connected client
5. Frontend EventSource receives events and invalidates TanStack Query cache for affected queries

## Key Abstractions

**AiPlatform Interface (`server/game/ai/ai.go`):**
- All AI provider code implements this interface
- Callers in `server/game/session_play.go` interact only through the interface
- `GetAiPlatform(name)` factory returns the correct implementation

**stream.Registry (`server/game/stream/stream.go`):**
- In-memory channel registry mapping `uuid.UUID` (message ID) to active `*Stream`
- Global singleton: `stream.Get()`
- Created per AI response on action submit; auto-removed after 5-minute timeout or on stream completion
- `Stream.Chunks` is a buffered channel (capacity 100) that SSE handler drains

**obj.GameSession (`server/obj/structs.go`):**
- Central struct carrying all context for AI calls: session ID, game definition (scenario, status fields, image style, workshop prompt constraints), active API key, AI platform/model, and `AiSession` (conversation state for continuity across turns)

**httpx Auth Wrappers (`server/api/httpx/auth.go`):**
- `RequireAuth(fn http.HandlerFunc) http.Handler` — requires authenticated user
- `OptionalAuth(fn http.HandlerFunc) http.Handler` — user may be nil (invite/guest endpoints)
- `RequireAuth0Token(fn http.HandlerFunc) http.Handler` — requires Auth0 RS256 JWT but user may be unregistered (registration endpoint)

**useStreamingSession Hook (`web/src/features/game-player-v2/hooks/useStreamingSession.ts`):**
- Core game session state machine on the frontend
- Uses `SessionAdapter` interface for auth-specific API calls (supports both authenticated and guest sessions)
- SSE primary delivery with polling fallback if SSE is silent for 10 seconds
- State phases: `idle → starting → playing → error`

**GamePlayerContext (`web/src/features/game-player-v2/context/GamePlayerContext.tsx`):**
- React context exposing game player state and actions to all child components
- Actions: `startSession`, `sendAction`, `loadExistingSession`, `retryLastAction`, `resetGame`
- Display controls: `fontSize`, `debugMode`, `textEffectsEnabled`, `isImageGenerationDisabled`

## Entry Points

**Server Binary:**
- Location: `server/cmd/root.go` (Cobra root) → `server/cmd/server/server.go`
- Triggers: `./cgl server` or via `run-dev.sh`/`run-prod.sh`
- Responsibilities: Parse CLI flags, call `api.RunServer(ctx, port, devMode, readyChan)`

**api.RunServer (`server/api/server.go`):**
- Triggers: Server subcommand
- Responsibilities: Init Sentry telemetry, JWT key generation, DB init + preseed, admin email promotions, build HTTP handler via `routes.Handler()`, start server, handle graceful shutdown (SIGINT/SIGTERM with 5-second timeout)

**Web App Root (`web/src/providers/AppProviders.tsx`):**
- Triggers: Browser page load (mounted from `main.tsx`)
- Responsibilities: Mount full provider tree including QueryClient, Auth0, Mantine, Auth, WorkshopMode, Router

**Root Route (`web/src/routes/__root.tsx`):**
- Triggers: Every navigation, on auth state changes
- Responsibilities: Auth guards via `useEffect` redirects, participant route restrictions, workshop mode detection, layout variant selection (`public`/`authenticated`/game-player dark mode/guest), role-based nav item construction

## Error Handling

**Strategy:** Structured error responses with typed string error codes; client maps codes to user-visible messages

**Server Patterns:**
- `obj.HTTPError{Code, Message}` for domain errors
- `httpx.WriteError(w, status, message)` — simple 4xx/5xx
- `httpx.WriteErrorWithCode(w, status, code, message)` — typed, e.g. `auth_workshop_inactive`
- AI errors normalized to codes in `server/game/session_play.go::extractAIErrorCode` (maps error strings to: `invalid_api_key`, `billing_not_active`, `rate_limit_exceeded`, etc.)
- Key-related errors trigger automatic session API key clearance and optional sponsorship removal
- Panics caught by `httpx.Recover`, reported to Sentry with fatal level

**Client Patterns:**
- Error codes extracted via `extractRawErrorCode` in `web/src/common/types/errorCodes.ts`
- TanStack Query surfaces errors to component error states
- Global `ErrorBoundary` at `web/src/common/components/ErrorBoundary.tsx` wraps entire app
- `GlobalErrorModal` at `web/src/common/components/GlobalErrorModal.tsx` for critical auth/session errors

## Cross-Cutting Concerns

**Logging:** `server/log/` wraps structured `slog`; `log.Info/Debug/Warn/Error` with key-value pairs; all HTTP requests logged by `httpx.Logging` including duration and status code; 4xx/5xx automatically reported to Sentry with request/response context

**Validation:** Inline in route handlers — UUID path params via `httpx.PathParamUUID`, JSON decode with error check, manual field validation; no schema validation library used

**Authentication:** Three-tier token resolution in single `Authenticate` middleware; participant token → direct DB lookup; CGL dev JWT → HS256 `auth.ValidateTokenString`; Auth0 JWT → RS256 JWKS validation with 5-minute cache; token sources: Authorization header, `cgl_session` cookie, `?token=` query param (SSE)

**i18n:** Backend: `server/lang/` loads locale JSON files from `server/lang/locales/`; game sessions translate scenario text to session language; Frontend: `react-i18next` with namespaced locale files in `web/src/i18n/locales/`

**Telemetry:** Sentry in `server/telemetry/`; initialized at startup with version tag; `httpx.Logging` middleware reports 4xx/5xx; `httpx.Recover` reports panics; flushed on shutdown

---

*Architecture analysis: 2026-03-08*
