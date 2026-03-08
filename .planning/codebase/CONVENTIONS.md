# Coding Conventions

**Analysis Date:** 2026-03-08

## Overview

This is a full-stack codebase with two primary languages: TypeScript (React frontend at `web/`) and Go (backend server at `server/`). Conventions differ per language but are internally consistent within each.

---

## TypeScript / React (web/)

### Naming Patterns

**Files:**
- React components: `PascalCase.tsx` (e.g., `GameCard.tsx`, `AppHeader.tsx`)
- CSS modules: `ComponentName.module.css` co-located with component
- Hooks: `camelCase.ts` prefixed with `use` (e.g., `useResponsiveDesign.ts`, `useGames.ts`)
- Utilities/libs: `camelCase.ts` (e.g., `formatters.ts`, `userUtils.ts`)
- Barrel files: `index.ts` at directory level

**Functions:**
- React components: `PascalCase` named exports (e.g., `export function AppHeader(...)`)
- Hooks: `camelCase` named exports prefixed with `use` (e.g., `export function useApiKeys()`)
- Utility functions: `camelCase` (e.g., `getUserInitials`, `formatRelativeTime`)
- Event handlers: `handleXxx` or `onXxx` prefix (e.g., `handleLogout`, `handleReset`)

**Variables:**
- `camelCase` throughout
- Boolean variables: `isXxx`, `hasXxx` pattern (e.g., `isMobile`, `isLoading`, `hasError`)
- Module-level constants: `SCREAMING_SNAKE_CASE` (e.g., `MOBILE_BREAKPOINT`, `EXTERNAL_LINKS`)
- Mutation keys: `const MUTATION_KEY = [...] as const` pattern

**Types/Interfaces:**
- `interface` for component props: `XxxProps` suffix (e.g., `ActionButtonProps`, `AppHeaderProps`)
- `interface` for data shapes
- `type` for unions, aliases, and function types
- Use `import type` keyword for type-only imports: `import type { ReactNode } from 'react'`

### Code Style

**Formatting:**
- No Prettier config detected; ESLint is the only enforced tool
- ESLint config at `web/eslint.config.js` using flat config format
- Rules: `@eslint/js` recommended + `typescript-eslint` recommended + `react-hooks` + `react-refresh`
- Target: ECMAScript 2020

**Linting:**
- Command: `npm run lint` (runs `eslint .`)
- TypeScript strict mode via `typescript-eslint`
- `@ts-nocheck` used in generated files only (`web/src/api/generated/index.ts`)

### Import Organization

**Order (observed pattern):**
1. Third-party packages (`react`, `@mantine/core`, `@tanstack/react-query`)
2. Internal absolute imports using path aliases
3. Relative imports

**Path Aliases (defined in `web/vite.config.ts`):**
- `@` → `web/src`
- `@api` → `web/src/api`
- `@common` → `web/src/common`
- `@config` → `web/src/config`
- `@features` → `web/src/features`
- `@i18n` → `web/src/i18n`
- `@components` → `web/src/common/components`
- `@hooks` → `web/src/common/hooks`
- `@lib` → `web/src/common/lib`
- `@types` → `web/src/common/types`
- `@version` → `web/src/version.js`

### Component Design

**Props Pattern:**
- Always define a named `interface XxxProps` for component props
- Use destructuring with defaults in function signature
- Wrap third-party components to create semantic abstractions

**Semantic Component System (use these, not raw Mantine):**

Buttons in `web/src/common/components/buttons/`:
- `ActionButton` — primary CTA, never for secondary actions
- `MenuButton` — action lists/menus
- `TextButton` — secondary/subtle actions
- `DangerButton` — destructive actions

Typography in `web/src/common/components/typography/`:
- `PageTitle` (h1), `SectionTitle` (h2), `CardTitle` (h3)
- `BodyText`, `HelperText`, `ErrorText`

Always import semantic components via `@components/buttons`, `@components/typography`, etc.

**JSDoc on shared components:**
```tsx
/**
 * ActionButton - Primary call-to-action button
 *
 * USE WHEN:
 * - Main action on a page (Login, Submit, Get Started)
 *
 * DO NOT USE FOR:
 * - Secondary actions (use TextButton)
 *
 * @example
 * <ActionButton onClick={handleLogin}>Get Started</ActionButton>
 */
```

### Error Handling (Frontend)

**React component errors:**
- Wrap feature trees with `ErrorBoundary` (`web/src/common/components/ErrorBoundary.tsx`)
- Errors logged via `uiLogger` in DEV only: `if (import.meta.env.DEV) { uiLogger.error(...) }`

**API mutation errors:**
- Use `handleApiError` from `web/src/config/queryClient.ts` in `onError` handlers
- TanStack Query: no retry on 4xx, max 1 retry for other errors
- Exponential backoff: `Math.min(1000 * Math.pow(2, attemptIndex), 30_000)`

**Standard mutation pattern:**
```typescript
const MUTATION_KEY = ["mutationName"] as const;

return useMutation<ReturnType, HttpxErrorResponse, InputType>({
  mutationKey: MUTATION_KEY,
  mutationFn: async (input) => {
    if (!api) throw new Error("Not authenticated");
    const response = await api.domain.endpoint(input);
    return response.data;
  },
  onSuccess: (data) => {
    queryClient.setQueryData(queryKeys.relevant, data);
    // or: queryClient.invalidateQueries({ queryKey: queryKeys.relevant });
  },
});
```

### Logging (Frontend)

**Framework:** Custom `Logger` class at `web/src/common/lib/logger.ts` with pluggable transport abstraction.

**Named loggers** (from `web/src/config/logger.ts`):
- `uiLogger` — UI component errors and events
- `apiLogger` — API call logging and retry tracking
- `navigationLogger` — navigation events

**Transports:**
- `ConsoleTransport` — always active in dev
- `SentryTransport` — forwards `Warning` and above to Sentry/GlitchTip in production

**Usage:**
```typescript
import { uiLogger } from '@/config/logger';
uiLogger.debug('Rendering', { props });
uiLogger.warning('Retrying request', { failureCount });
uiLogger.error('Failed to load data', { error });
```

Log levels: `Debug`, `Info`, `Warning`, `Error`, `Fatal`.

### Module Design

**Barrel files:** Every domain directory has an `index.ts`. Always import from the barrel:
```typescript
// Correct
import { useApiKeys, useGames } from '@/api/hooks';

// Avoid
import { useApiKeys } from '@/api/hooks/useApiKeys';
```

**Barrel export style:** Named exports with explicit type re-exports:
```typescript
export { ActionButton, type ActionButtonProps } from './ActionButton';
```

**Query keys:** Centralized in `web/src/api/queryKeys.ts`. Always use `queryKeys.xxx` — never inline strings.

---

## Go (server/)

### Naming Patterns

**Files:**
- `snake_case.go` (e.g., `resolve_api_key.go`, `session_creation.go`)
- Test files: `xxx_test.go` in the `testing/` module
- Generated files: `*.sql.go` in `server/db/sqlc/` — never edit manually

**Packages:**
- Short, lowercase, single-word (`routes`, `httpx`, `auth`, `obj`, `db`, `lang`)
- Package name matches directory name

**Functions:**
- Exported: `PascalCase` (e.g., `WriteJSON`, `GetLanguages`, `PathParamUUID`)
- Unexported: `camelCase` (e.g., `findAvailablePort`, `getCookiePath`, `getAuth0Validator`)

**Types/Structs:**
- Exported: `PascalCase` (e.g., `HTTPError`, `StatusResponse`, `TokenUsage`)
- Context keys: unexported struct type pattern `type ctxKeyUser struct{}`

**Constants:**
- Exported: `PascalCase` (e.g., `ProjectName`)
- Error codes: `ErrCodeXxx` string constants in `server/obj/errors.go`

### HTTP Handler Pattern

All route handlers follow this signature and godoc annotation pattern:
```go
// GetLanguages godoc
//
//   @Summary      Short description
//   @Description  Longer description
//   @Tags         tagname
//   @Produce      json
//   @Success      200  {object}  ResponseType
//   @Failure      404  {object}  httpx.ErrorResponse
//   @Security     BearerAuth
//   @Router       /path [get]
func GetLanguages(w http.ResponseWriter, r *http.Request) {
    // Extract authenticated user (only in RequireAuth-wrapped handlers)
    user := httpx.UserFromRequest(r)

    // Parse path parameters
    id, err := httpx.PathParamUUID(r, "id")
    if err != nil {
        httpx.WriteError(w, http.StatusBadRequest, "invalid id")
        return
    }

    // Business logic...

    httpx.WriteJSON(w, http.StatusOK, result)
}
```

### Error Handling (Backend)

**Write errors in handlers:**
```go
httpx.WriteError(w, http.StatusNotFound, "game not found")
httpx.WriteErrorWithCode(w, http.StatusForbidden, obj.ErrCodeForbidden, "access denied")
httpx.WriteErrorf(w, http.StatusBadRequest, "invalid value: %q", value)
```

**Create typed errors in business logic:**
```go
return obj.ErrValidation("name is required")
return obj.NewHTTPError(http.StatusConflict, "name already exists")
return obj.NewHTTPErrorf(http.StatusBadRequest, "unknown platform: %q", platform)
```

**Error codes** are defined in `server/obj/errors.go` and consumed by the frontend for typed error handling (e.g., `ErrCodeNoApiKey`, `ErrCodeRateLimitExceeded`).

**Middleware auto-reporting:** `httpx.Logging` middleware auto-reports 4xx/5xx responses to Sentry with request body context.

### Logging (Backend)

**Package:** Custom `log` package at `server/log/`.

**Usage (structured key-value pairs):**
```go
import "cgl/log"
log.Info("server started", "port", port)
log.Error("database error", "error", err, "query", queryName)
log.Debug("auth disabled", "reason", "AUTH0_DOMAIN not set")
```

### Comments

**Exported functions:** Always add a godoc comment starting with the function name.
```go
// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) { ... }
```

**Swagger annotations:** All route handlers must have complete `// FunctionName godoc` annotations covering Summary, Tags, Produce, Success, Failure, and Router.

**Generated files:** Files starting with `// Code generated by sqlc. DO NOT EDIT.` must never be modified manually.

### Import Organization (Go)

Standard gofmt order:
1. Standard library
2. Internal packages (`cgl/...`)
3. Third-party packages

### Database

- All DB queries generated via sqlc from SQL files
- Generated output: `server/db/sqlc/*.sql.go` — read-only
- SQL source: inferred from `server/sqlc.yaml`
- Domain structs separate from DB structs live in `server/obj/`
- DB operations return `(value, error)` — always check the error

---

*Convention analysis: 2026-03-08*
