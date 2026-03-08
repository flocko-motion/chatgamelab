# Codebase Structure

**Analysis Date:** 2026-03-08

## Directory Layout

```
chatgamelab/                     # Monorepo root
├── server/                      # Go backend (module: cgl)
│   ├── main.go                  # Binary entry point
│   ├── api/                     # HTTP layer
│   │   ├── routes/              # Route handlers (one file per resource)
│   │   ├── httpx/               # Middleware, auth, response helpers
│   │   ├── auth/                # JWT generation helpers
│   │   └── client/              # Internal HTTP client (apiclient)
│   ├── apiclient/               # Typed Go client for the REST API
│   ├── cmd/                     # Cobra CLI subcommands
│   │   ├── server/              # `cgl server` — starts HTTP server
│   │   ├── user/                # `cgl user` — user management
│   │   ├── game/                # `cgl game` — game management
│   │   ├── ai/                  # `cgl ai` — AI test/tooling
│   │   ├── apikey/              # `cgl apikey` — API key management
│   │   ├── institution/         # `cgl institution` — institution management
│   │   ├── workshop/            # `cgl workshop` — workshop management
│   │   ├── invite/              # `cgl invite` — invite management
│   │   ├── lang/                # `cgl lang` — translation tooling
│   │   └── healthcheck/         # `cgl healthcheck` — Docker healthcheck
│   ├── config/                  # Server configuration helpers
│   ├── constants/               # Shared server constants
│   ├── db/                      # Database layer
│   │   ├── init.go              # Connection, schema init, migrations
│   │   ├── *.go                 # Query functions per domain (game.go, user.go, etc.)
│   │   ├── queries/             # Raw SQL query files (*.sql)
│   │   ├── sqlc/                # sqlc-generated Go code (do not edit)
│   │   ├── migrations/          # Sequential numbered migration files (001_*.sql, 002_*.sql, …)
│   │   ├── schema.sql           # Full baseline schema (applied to fresh DBs)
│   │   └── permissions/         # Permission helper SQL
│   ├── events/                  # SSE event broker for workshop events
│   ├── functional/              # Generic utility functions (EnvOrDefault, RequireEnv, Ptr, etc.)
│   ├── game/                    # Game engine: session lifecycle, AI orchestration
│   │   ├── session_creation.go  # CreateSession, createSessionInternal, generateSessionSetup
│   │   ├── session_play.go      # DoSessionAction, DoSessionActionWithFallback
│   │   ├── session_lock.go      # Per-session mutex to serialize AI calls
│   │   ├── resolve_api_key.go   # API key priority chain resolution
│   │   ├── guest.go             # CreateGuestSession (anonymous share-token play)
│   │   ├── theme.go             # GenerateTheme (AI visual theme for game UI)
│   │   ├── translate.go         # TranslateGame (AI translation of game content)
│   │   ├── ai/                  # AI platform abstraction + implementations
│   │   │   ├── ai.go            # AiPlatform interface, GetAiPlatform factory
│   │   │   ├── openai/          # OpenAI implementation
│   │   │   ├── mistral/         # Mistral implementation
│   │   │   └── mock/            # Mock implementation for testing
│   │   ├── imagecache/          # In-memory cache for generated image status
│   │   ├── status/              # Game status field helpers
│   │   ├── stream/              # SSE stream registry (channel-based, keyed by message UUID)
│   │   └── templates/           # AI prompt templates, response schema builder, image style constants
│   ├── lang/                    # Server-side i18n + AI-driven translation
│   │   └── locales/             # Server locale JSON files
│   ├── log/                     # Structured logger (wraps slog)
│   ├── obj/                     # Domain objects and shared types
│   │   ├── structs.go           # All domain structs: User, Game, GameSession, GameSessionMessage, ApiKey, Workshop, Institution, etc.
│   │   ├── errors.go            # Error type constants and helpers
│   │   ├── http_error.go        # HTTPError type (status + machine code + message)
│   │   └── invite.go            # Invite-specific types
│   ├── telemetry/               # Sentry initialization
│   ├── docs/                    # Generated OpenAPI/Swagger docs
│   └── Dockerfile               # Backend Docker image
├── web/                         # React/TypeScript SPA
│   ├── src/
│   │   ├── api/                 # API client layer
│   │   │   ├── client/          # Configured Axios client + re-exports (index.ts, http.ts)
│   │   │   ├── generated/       # Auto-generated TypeScript API types from OpenAPI spec
│   │   │   └── hooks/           # TanStack Query hooks per domain (useGames.ts, useSessions.ts, etc.)
│   │   ├── assets/              # Static assets (logos, images)
│   │   ├── common/              # Shared UI building blocks
│   │   │   ├── components/      # Reusable components (Layout, buttons, cards, controls, DataTable, etc.)
│   │   │   ├── contexts/        # Shared React contexts
│   │   │   ├── hooks/           # Shared custom hooks
│   │   │   ├── lib/             # Pure utility functions (roles.ts, url.ts, formatters.ts, etc.)
│   │   │   ├── routes/          # Route constants (routes.ts)
│   │   │   └── types/           # Shared TypeScript types (errorCodes.ts, etc.)
│   │   ├── config/              # App-level configuration
│   │   │   ├── env.ts           # Runtime config (window.__APP_CONFIG__ / VITE_* vars)
│   │   │   ├── auth0.ts         # Auth0 client config
│   │   │   ├── queryClient.ts   # TanStack Query client + error handling
│   │   │   ├── router.ts        # TanStack Router instance
│   │   │   ├── mantineTheme.ts  # Mantine UI theme
│   │   │   └── sentry.ts        # Sentry frontend init
│   │   ├── features/            # Feature-sliced domain modules
│   │   │   ├── game-player-v2/  # Core game player (the main interactive experience)
│   │   │   │   ├── components/  # GamePlayer.tsx, SceneCard.tsx, PlayerInput.tsx, StatusBar.tsx, etc.
│   │   │   │   ├── context/     # GamePlayerContext.tsx
│   │   │   │   ├── hooks/       # useGameSession.ts, useGuestGameSession.ts, useStreamingSession.ts, useAudioRecorder.ts, etc.
│   │   │   │   ├── lib/         # Game player utilities
│   │   │   │   ├── theme/       # Theme presets for the game UI
│   │   │   │   └── types.ts     # GamePlayerState, SceneMessage, StreamChunk types
│   │   │   ├── games/           # Game library/catalogue feature
│   │   │   ├── auth/            # Registration form
│   │   │   ├── admin/           # Admin panel components
│   │   │   ├── api-keys/        # API key management
│   │   │   ├── dashboard/       # Dashboard components
│   │   │   ├── debug/           # Debug/dev panel
│   │   │   ├── my-organization/ # Organization management
│   │   │   ├── my-workshop/     # Workshop management for staff/heads
│   │   │   ├── play/            # Play feature components
│   │   │   ├── profile/         # User profile
│   │   │   └── settings/        # Settings
│   │   ├── i18n/                # Frontend internationalisation
│   │   │   └── locales/         # Locale JSON files
│   │   ├── providers/           # App-level React providers
│   │   │   ├── AppProviders.tsx # Root provider tree (Auth0, Mantine, Query, Router, Auth, Workshop)
│   │   │   ├── AuthProvider.tsx # Auth state (Auth0 + backend user + participant tokens)
│   │   │   ├── WorkshopModeProvider.tsx # Workshop mode state (staff acting as participant)
│   │   │   └── auth/            # Auth internals (tokenStorage.ts, useTokenManager.ts, useBackendUser.ts)
│   │   └── routes/              # TanStack Router file-based routes
│   │       ├── __root.tsx       # Root route: layout, auth guards, nav
│   │       ├── _app/            # Authenticated app shell
│   │       ├── app/             # App routes
│   │       ├── auth/            # Auth routes (login, logout, Auth0 callbacks)
│   │       ├── admin/           # Admin routes (organizations, users, server-settings)
│   │       ├── games/           # Game browsing + play routes
│   │       ├── my-games/        # User's own games
│   │       ├── my-organization/ # Organization routes
│   │       ├── my-workshop/     # Workshop routes
│   │       ├── play/            # Guest play routes (anonymous, share token)
│   │       ├── sessions/        # Session history
│   │       └── invites/         # Invite acceptance
│   ├── public/                  # Static public files
│   ├── plugins/                 # Vite plugins
│   └── dist/                    # Built SPA output (committed for Docker)
├── testing/                     # Integration test infrastructure (Go module)
│   ├── testutil/                # Test suite helpers (suite.go, testclient.go)
│   └── testdata/games/          # Sample game YAML files for tests
├── docker/                      # Docker configuration
│   └── db/                      # Custom DB Docker image (schema pre-baked)
├── bin/                         # Local dev binaries
├── docker-compose.yml           # Production deployment
├── docker-compose.dev.yml       # Development deployment
├── nginx.conf                   # Nginx config (serves SPA + proxies API)
├── go.work                      # Go workspace (server + testing modules)
└── .planning/                   # GSD planning documents
    └── codebase/                # Codebase analysis documents
```

## Directory Purposes

**`server/obj/`:**
- Purpose: Single source of truth for all domain types shared across server layers
- Contains: Structs only — no DB code, no HTTP code, no business logic
- Key files: `server/obj/structs.go` (all domain types), `server/obj/http_error.go`

**`server/api/routes/`:**
- Purpose: HTTP handler functions, one file per resource domain
- Contains: Request/response struct definitions, handler implementations, Swagger annotations
- Key files: `router.go` (all route registrations), `sessions.go`, `games.go`, `users.go`, `guest_play.go`

**`server/api/httpx/`:**
- Purpose: Reusable HTTP infrastructure, not tied to specific routes
- Key files: `auth.go` (authentication middleware), `middleware.go` (logging, CORS, panic recovery), `response.go` (WriteJSON, WriteError, WriteErrorWithCode), `params.go` (PathParamUUID, QueryParam)

**`server/game/`:**
- Purpose: All game-related business logic — the core of the application
- No direct HTTP dependencies (uses `obj` types, `db` functions, `game/ai` interfaces)

**`server/db/`:**
- Purpose: All database access; the only layer that imports `database/sql` and `cgl/db/sqlc`
- Pattern: Domain-specific `.go` files (e.g. `game.go`, `user.go`) wrap sqlc-generated functions and map to `obj` types

**`web/src/features/`:**
- Purpose: Self-contained feature modules — each feature owns its components, hooks, and local types
- `game-player-v2/` is the most complex; treat it as its own mini-app

**`web/src/common/`:**
- Purpose: Shared UI building blocks used across multiple features — no feature-specific logic
- Components in `web/src/common/components/` must remain generic

**`web/src/api/hooks/`:**
- Purpose: TanStack Query hooks for server state, grouped by domain
- Examples: `web/src/api/hooks/useGames.ts`, `web/src/api/hooks/useSessions.ts`

## Key File Locations

**Entry Points:**
- `server/main.go`: Go binary entry point
- `server/api/server.go`: HTTP server startup (`RunServer`)
- `server/api/routes/router.go`: All route registrations + middleware chain
- `web/src/providers/AppProviders.tsx`: React app root (all providers)
- `web/src/routes/__root.tsx`: TanStack Router root route (layout, auth guards)

**Domain Types:**
- `server/obj/structs.go`: All server-side domain types

**Authentication:**
- `server/api/httpx/auth.go`: Token validation middleware (`RequireAuth`, `OptionalAuth`, `Authenticate`)
- `web/src/providers/AuthProvider.tsx`: Frontend auth context
- `web/src/providers/auth/tokenStorage.ts`: Token persistence (participant token, dev token)

**Database:**
- `server/db/init.go`: Connection + migration runner
- `server/db/schema.sql`: Full baseline schema
- `server/db/migrations/`: Sequential numbered SQL migrations
- `server/db/queries/`: Raw SQL files (input to sqlc)
- `server/db/sqlc/`: Generated query code (do not edit manually)

**AI Integration:**
- `server/game/ai/ai.go`: `AiPlatform` interface definition + factory
- `server/game/ai/openai/`: OpenAI implementation
- `server/game/ai/mistral/`: Mistral implementation
- `server/game/templates/templates.go`: AI prompt templates

**Game Player (Frontend):**
- `web/src/features/game-player-v2/hooks/useStreamingSession.ts`: Core game player state machine (SSE + polling)
- `web/src/features/game-player-v2/hooks/useGameSession.ts`: Authenticated game session adapter
- `web/src/features/game-player-v2/hooks/useGuestGameSession.ts`: Guest game session adapter
- `web/src/features/game-player-v2/components/GamePlayer.tsx`: Main game player UI component

**Configuration:**
- `web/src/config/env.ts`: Runtime config (supports `window.__APP_CONFIG__` for Docker)
- `server/functional/tools.go`: `RequireEnv`, `EnvOrDefault` helpers

## Naming Conventions

**Server (Go):**
- Files: `snake_case.go` matching the primary domain (e.g. `session_creation.go`, `api_key_shares.go`)
- Packages: Short lowercase, matching the directory name (`routes`, `httpx`, `game`, `db`, `obj`)
- Types: `PascalCase` (e.g. `GameSession`, `ApiKeyShare`)
- Functions: `PascalCase` for exported, `camelCase` for unexported
- Route handlers: `VerbNoun` pattern (e.g. `GetGameByID`, `CreateGameSession`, `PostSessionAction`)
- DB functions: Named after the operation (e.g. `GetGameByID`, `CreateGameSession`, `UpdateApiKeyLastUsageSuccess`)

**Frontend (TypeScript):**
- Components: `PascalCase.tsx` (e.g. `GamePlayer.tsx`, `SceneCard.tsx`)
- Hooks: `use` prefix, `camelCase` (e.g. `useGameSession.ts`, `useStreamingSession.ts`)
- Types: `PascalCase` (e.g. `GamePlayerState`, `SessionAdapter`)
- Utilities: `camelCase.ts` (e.g. `roles.ts`, `formatters.ts`)
- Route files: TanStack Router file-based convention (`route.tsx`, `index.tsx`, `$paramName.tsx`)
- CSS modules: Component name + `.module.css` (e.g. `GamePlayer.module.css`)
- Query hooks: `use` + noun (e.g. `useGames`, `useSessions`)

## Where to Add New Code

**New API endpoint (server):**
1. Add handler function to existing file in `server/api/routes/` (or new file if new resource domain)
2. Register route in `server/api/routes/router.go`
3. Add SQL query to `server/db/queries/` if needed, run sqlc to generate, add wrapper in `server/db/`
4. Use `server/obj/structs.go` for request/response types if they are domain objects

**New domain object / field:**
1. Add to `server/obj/structs.go`
2. Add DB column via migration: new file in `server/db/migrations/` with next sequential number
3. Update `server/db/schema.sql` with the new column
4. Add/update sqlc query in `server/db/queries/*.sql`, regenerate sqlc
5. Update DB wrapper function in `server/db/`

**New feature (frontend):**
1. Create directory under `web/src/features/{feature-name}/`
2. Add sub-directories as needed: `components/`, `hooks/`, `lib/`
3. Add route file under `web/src/routes/` following TanStack Router conventions
4. Register in `web/src/common/routes/routes.ts` if it needs a named constant
5. Add nav item in `web/src/routes/__root.tsx` if it needs top-level navigation

**New shared UI component:**
- Place in `web/src/common/components/` — must be generic, no feature-specific data fetching

**New TanStack Query hook:**
- Add to `web/src/api/hooks/` and export from `web/src/api/hooks/index.ts`

**New AI platform:**
1. Create directory under `server/game/ai/` implementing the `ai.AiPlatform` interface
2. Register in `server/game/ai/ai.go` `getAiPlatform()` switch and `ApiKeyPlatforms` list

**New CLI command:**
1. Create package under `server/cmd/{name}/`
2. Register in `server/cmd/root.go`

## Special Directories

**`server/db/sqlc/`:**
- Purpose: Auto-generated Go query code from sqlc
- Generated: Yes (run `server/sqlc.sh`)
- Committed: Yes
- Do NOT edit manually

**`web/src/api/generated/`:**
- Purpose: Auto-generated TypeScript API types from OpenAPI spec
- Generated: Yes (run `server/generate-openapi.sh` + frontend codegen)
- Committed: Yes
- Do NOT edit manually

**`web/dist/`:**
- Purpose: Built SPA output
- Generated: Yes (Vite build)
- Committed: Yes (used directly in Docker image)

**`.planning/`:**
- Purpose: GSD planning documents
- Generated: By GSD agent commands
- Committed: Yes

---

*Structure analysis: 2026-03-08*
