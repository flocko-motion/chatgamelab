# Technology Stack

**Analysis Date:** 2026-03-08

## Languages

**Primary:**
- Go 1.25 - Backend server (`server/`) and CLI tool (`cgl`)
- TypeScript ~5.9.3 - Frontend web app (`web/src/`)

**Secondary:**
- SQL - Database schema and queries (`server/db/schema.sql`, `server/db/queries/`, `server/db/migrations/`)

## Runtime

**Environment:**
- Go 1.25 (backend) - `server/go.mod`
- Node.js 24 (frontend build only) - `web/Dockerfile`
- nginx 1.27-alpine (frontend serving in production)
- Debian 13-slim (backend runtime container)
- PostgreSQL 18 (database container)

**Package Manager:**
- Go modules (`go.work` workspace at repo root with `server/` and `testing/` modules)
- npm (frontend) - lockfile: `web/package-lock.json` present

## Frameworks

**Backend:**
- Standard library `net/http` - HTTP server and routing via `http.ServeMux` (no external HTTP framework); see `server/api/routes/router.go`
- `github.com/spf13/cobra` v1.10.2 - CLI command structure (`server/cmd/`)
- `github.com/joho/godotenv` v1.5.1 - `.env` file loading in dev
- `gopkg.in/yaml.v2` v2.4.0 + `gopkg.in/yaml.v3` v3.0.1 - YAML parsing (game definitions stored as YAML)
- `github.com/swaggo/swag` v1.16.3 - OpenAPI/Swagger doc generation from annotations

**Frontend:**
- React 19.2.0 - UI framework
- `@tanstack/react-router` v1.144.0 - File-based routing; routes in `web/src/routes/`, tree generated to `web/src/routeTree.gen.ts`
- `@tanstack/react-query` v5.90.16 - Server state and data fetching
- `@mantine/core` v8.3.10 - UI component library (forced light mode in `web/src/providers/AppProviders.tsx`)
- `@mantine/hooks`, `@mantine/modals`, `@mantine/notifications`, `@mantine/dates` - Mantine ecosystem
- `react-hook-form` v7.62.0 + `@hookform/resolvers` v5.2.0 + `zod` v4.1.9 - Form handling and validation
- `i18next` v25.7.3 + `react-i18next` v16.5.1 + `i18next-http-backend` + `i18next-browser-languagedetector` - i18n (40+ languages)
- `three` v0.182.0 + `@tsparticles/react` v3.0.0 + `tsparticles` v3.9.1 - 3D/particle effects for game UI
- `dayjs` v1.11.13 - Date handling

**Auth (both layers):**
- `github.com/auth0/go-jwt-middleware/v2` v2.2.2 - Backend Auth0 RS256 JWT validation via JWKS
- `github.com/golang-jwt/jwt/v4` v4.5.2 - Backend dev JWT signing/validation (HS256, `DEV_JWT_SECRET`)
- `@auth0/auth0-react` v2.11.0 - Frontend Auth0 SPA SDK (token cache in `localstorage`, refresh tokens enabled)

**Testing:**
- `github.com/stretchr/testify` v1.10.0 - Go test assertions (backend unit + integration tests in `testing/`)

**Build/Dev:**
- Vite (via `rolldown-vite` v7.3.0) - Frontend dev server and production build; config at `web/vite.config.ts`
- `@vitejs/plugin-react` v5.1.1 - React JSX transform
- `@tanstack/router-plugin` v1.145.2 - Vite plugin for automatic route tree generation
- TypeScript compiler (`tsc`) - Type checking
- ESLint v9.39.1 + `typescript-eslint` v8.46.4 + `eslint-plugin-react-hooks` + `eslint-plugin-react-refresh` - Linting
- `swagger-typescript-api` v13.2.7 - Generates TypeScript API client (`web/src/api/generated/index.ts`) from `web/swagger.json`
- `sqlc` (external tool) - SQL-to-Go code generation; config at `server/sqlc.yaml`, output to `server/db/sqlc/`

## Key Dependencies

**Critical:**
- `github.com/lib/pq` v1.10.9 - PostgreSQL driver (sole DB dependency, used with `database/sql`)
- `github.com/google/uuid` v1.6.0 - UUID generation (primary ID type throughout)
- `github.com/sqlc-dev/pqtype` v0.3.0 - Nullable types for sqlc-generated code
- `github.com/getsentry/sentry-go` v0.42.0 - Backend error tracking (Sentry/GlitchTip)
- `@sentry/react` v10.38.0 - Frontend error tracking

**Infrastructure:**
- `@tabler/icons-react` v3.36.1 - Icon library for Mantine UI
- `js-yaml` v4.1.1 - YAML parsing in frontend (game YAML editing)
- `github.com/dillonstreator/go-unique-name-generator` v1.0.2 - Participant name generation
- `github.com/drhodes/golorem` - Lorem ipsum for DB seeding
- `github.com/olekukonko/tablewriter` v1.1.2 - CLI table output for `cgl` tool

## Configuration

**Environment:**
- Single `.env` file at repo root, used by both backend and frontend
- Backend loads it via `godotenv.Load("../.env")` in `server/main.go`
- Frontend loads it via Vite with `envDir: '../'` set in `web/vite.config.ts`
- Production: env vars injected at Docker container start; frontend vars written into `window.__APP_CONFIG__` by nginx entrypoint script `web/docker/entrypoint.sh`
- See `.env.example` for full variable reference

**Required backend env vars:**
- `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_DATABASE`, `PORT_POSTGRES`
- `PORT_BACKEND`
- `AUTH0_DOMAIN`, `AUTH0_AUDIENCE`
- `DEV_JWT_SECRET` (development only)
- `SENTRY_DSN_BACKEND` (optional)
- `ADMIN_EMAILS` (optional, comma-separated — auto-promoted to admin on sign-in)

**Required frontend env vars (VITE_ prefix in dev, bare in production Docker):**
- `VITE_API_BASE_URL` / `API_BASE_URL`
- `VITE_AUTH0_DOMAIN` / `AUTH0_DOMAIN`
- `VITE_AUTH0_CLIENT_ID` / `AUTH0_CLIENT_ID`
- `VITE_AUTH0_AUDIENCE` / `AUTH0_AUDIENCE`
- `VITE_SENTRY_DSN_FRONTEND` / `SENTRY_DSN_FRONTEND` (optional)

**Build:**
- `web/vite.config.ts` - Vite config; path aliases: `@`, `@api`, `@common`, `@config`, `@features`, `@i18n`, `@components`, `@hooks`, `@lib`, `@types`
- `web/tsconfig.app.json` - TypeScript strict mode, `ES2022` target, all strict checks enabled
- `server/sqlc.yaml` - sqlc config (PostgreSQL, schema + queries → `server/db/sqlc/`)
- `server/generate-openapi.sh` - regenerates `web/swagger.json` from Go annotations

## Platform Requirements

**Development:**
- Go 1.25+, Node.js 24+, npm
- PostgreSQL via Docker: `docker compose -f docker-compose.dev.yml --profile db up`
- Copy `.env.example` to `.env` and configure Auth0 credentials

**Production:**
- Docker Compose (`docker-compose.yml`) pulling from GHCR
- Three services: `db` (postgres:18), `backend` (Go binary), `web` (nginx)
- Deployed to Coolify; CI/CD via GitHub Actions (`.github/workflows/docker-image.yml`)
- Backend listens on port 3000 internally; frontend nginx proxies `/api/` to backend

---

*Stack analysis: 2026-03-08*
