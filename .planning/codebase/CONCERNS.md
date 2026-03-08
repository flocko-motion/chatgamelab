# Codebase Concerns

**Analysis Date:** 2026-03-08

## Tech Debt

**SSE Reconnection Disabled:**
- Issue: On-page-reload SSE reconnect logic is commented out with a TODO. If the user reloads mid-stream, polling fallback handles it, but reconnecting to the live stream is not attempted.
- Files: `web/src/features/game-player-v2/hooks/useStreamingSession.ts:770–776`
- Impact: Users who reload mid-generation see polling fallback (3-second interval) instead of instant SSE reconnection. Minor UX regression.
- Fix approach: Re-enable the commented `connectToStream` block after verifying the two-phase init flow is stable.

**Frontend i18n Translation Stub:**
- Issue: The backend translation loader returns English text for all non-static languages. The actual translation API call is commented out with a TODO.
- Files: `web/src/i18n/backendLoader.ts:25–33`
- Impact: Only English and German are actually translated in the UI. All other languages silently fall back to English without the user being notified.
- Fix approach: Implement the `POST /translate` call or wire in a translation service.

**`PostRestart` Endpoint is a No-op:**
- Issue: The admin-only `POST /api/restart` endpoint logs a message but never actually restarts the server. The goroutine it spawns waits for `r.Context().Done()` (which fires after the response is sent) then only logs.
- Files: `server/api/routes/sessions.go:703–722`
- Impact: Admin UI restart button has no effect. The server continues running.
- Fix approach: Send a signal to the main process context (e.g., close a shutdown channel) to trigger graceful shutdown and process restart.

**`organization` and `favorites` Filter Types Fall Back to `all`:**
- Issue: The `GetGames` function accepts `organization` and `favorites` as filter values but treats both as `all`.
- Files: `server/db/game.go:89`
- Impact: Filter UI options that select these values show all visible games instead of scoped results.
- Fix approach: Implement organization and favorites filters or remove them from the public API.

**Legacy `SetCORSHeaders` Function Left in Codebase:**
- Issue: `SetCORSHeaders` (wildcard origin, no credentials) is marked deprecated but remains in the code. It returns `Access-Control-Allow-Origin: *` which would be incompatible with credential-bearing requests.
- Files: `server/api/httpx/response.go:171–175`
- Impact: Low direct risk (the deprecated function is not called by the main middleware), but future copy-paste risk.
- Fix approach: Remove the deprecated function.

**Commented-out Debug Console Logs in Production Code:**
- Issue: Multiple commented-out `console.log('[SSE-DEBUG]')` calls remain in the SSE streaming hook, plus an active `console.log` on every theme change.
- Files: `web/src/features/game-player-v2/hooks/useStreamingSession.ts:333, 335, 504, 511, 772`, `web/src/features/game-player-v2/hooks/useSessionLifecycle.ts:171`
- Impact: Active log on theme change fires in production; commented lines are noise.
- Fix approach: Remove active log; delete commented debug lines.

**`GetAllUsers` Returns Unbounded Results:**
- Issue: `GetAllUsersWithDetails` SQL query has no LIMIT. For the admin list endpoint it fetches every non-deleted user.
- Files: `server/db/sqlc/user.sql.go:651–682`, `server/api/routes/users.go:53–59`
- Impact: As user count grows, this endpoint becomes progressively slower and memory-intensive.
- Fix approach: Add pagination parameters to `GetAllUsers` and its SQL query.

**`GetGameSessionsByGameID` Returns All Sessions Without Limit:**
- Issue: The session listing query for a game has no `LIMIT` clause.
- Files: `server/db/sqlc/game.sql.go:837–838`, `server/api/routes/sessions.go:228–234`
- Impact: A game with many sessions will return all of them in one query.
- Fix approach: Add a LIMIT or paginate this endpoint.

**`GetAllGameSessionMessages` Loads Binary Columns for Full Session History:**
- Issue: Loading all messages for a session fetches the full message history including `image` and `audio` binary columns (potentially MB each per message).
- Files: `server/db/sqlc/game.sql.go:562–563`, `server/db/game.go:1441`
- Impact: Long game sessions with image/audio will return multi-megabyte payloads from a single DB call.
- Fix approach: Exclude binary columns (`image`, `audio`) from the list query; serve those only via the dedicated `/messages/{id}/image` and `/messages/{id}/audio` endpoints.

**Duplicate Auth0 JWT Validator Initialization:**
- Issue: `getAuth0Validator()` (lines 31–66) and the `initAuth0` closure inside `Authenticate()` (lines 163–209) both independently parse `AUTH0_DOMAIN`, create a JWKS caching provider, and build a `validator.Validator`. Two separate singleton flags track two separate instances.
- Files: `server/api/httpx/auth.go:24–209`
- Impact: Two JWKS caching providers active simultaneously, double memory usage.
- Fix approach: Extract a single `initAuth0Validator()` function and reference it from both locations.

**`BackgroundAnimation.tsx` is a 1302-line Monolith:**
- Issue: Single file contains all particle system configurations and rendering logic for every animation variant.
- Files: `web/src/features/game-player-v2/components/BackgroundAnimation.tsx`
- Impact: Hard to add new animation types; no separation of config from component logic.
- Fix approach: Extract each animation config into its own file under `game-player-v2/theme/animations/`.

**`WorkshopsTab.tsx` / `SingleWorkshopSettings.tsx` Oversized Components:**
- Issue: 1544-line and 926-line components respectively mix data-fetching, business logic, and presentation in a single file.
- Files: `web/src/features/my-organization/components/WorkshopsTab.tsx`, `web/src/features/my-organization/components/SingleWorkshopSettings.tsx`
- Impact: Difficult to test, slow to navigate.
- Fix approach: Split into sub-components and custom hooks.

---

## Security Considerations

**CORS Policy Reflects Any Origin Without Allowlist:**
- Risk: `SetCORSHeadersWithOrigin` reflects whatever `Origin` header is sent by the client without validation. An attacker-controlled page can set any origin and receive credentialed responses.
- Files: `server/api/httpx/response.go:179–186`, `server/api/httpx/middleware.go:33–35`, `server/api/routes/sessions.go:626–631`
- Current mitigation: Authentication is required for all sensitive routes; cookies are `HttpOnly`. The main practical risk is CSRF via the `cgl_session` cookie.
- Recommendations: Maintain an explicit list of allowed origins (configured from an `ALLOWED_ORIGINS` env var); reject unknown origins with 403 on non-preflight requests.

**JWT Passed as Query Parameter for SSE Endpoints:**
- Risk: JWTs passed as `?token=...` are logged in access logs, proxy logs, browser history, and Referer headers.
- Files: `server/api/httpx/auth.go:230–234`, `server/api/routes/workshop_events.go:19`
- Current mitigation: Only accepted from query param when the `Authorization` header is absent (SSE-only path). Auth0 tokens are short-lived.
- Recommendations: Use a short-lived one-time token exchange for SSE (mint a signed nonce server-side) to avoid putting the bearer token in the URL.

**Message Image/Audio/Status Endpoints Unauthenticated — Rely on UUID Unguessability:**
- Risk: `/messages/{id}/status`, `/messages/{id}/image`, `/messages/{id}/audio`, and `/messages/{id}/stream` have no authentication check. Access is defended solely by the randomness of UUIDs. For long-lived stored images (1-year cache) this is a permanent exposure.
- Files: `server/api/routes/router.go:132–138`, `server/api/routes/sessions.go:383–392`, `server/api/routes/sessions.go:524–556`
- Current mitigation: Code explicitly acknowledges this design; UUIDs are cryptographically random.
- Recommendations: For sessions with sensitive content, add optional bearer-token or session-cookie check. At minimum audit that stored images are not cacheable by shared caches (`Cache-Control: private`).

**No HTTP Request Body Size Limit or Server Timeouts:**
- Risk: The `http.Server` has no `ReadTimeout`, `WriteTimeout`, or `IdleTimeout` set. No `http.MaxBytesReader` applied anywhere. A large audio base64 payload or slow client can exhaust connections and memory.
- Files: `server/api/server.go` (http.Server initialization), `server/api/httpx/middleware.go` (no body-limiting middleware)
- Current mitigation: None detected.
- Recommendations: Set `ReadTimeout`, `WriteTimeout`, `IdleTimeout` on `http.Server`. Apply `http.MaxBytesReader` in `ReadJSON` helper or on routes that accept audio/file payloads.

**Internal Error Messages Leaked to API Clients:**
- Risk: `err.Error()` is appended verbatim to 500-level response messages throughout the route handlers, exposing DB schema details, query structure, or internal paths.
- Files: `server/api/routes/sessions.go:56`, `server/api/routes/apikeys.go:63`, `server/api/routes/apikeys.go:130`, and approximately 100 additional occurrences across all route files.
- Current mitigation: Sentry captures error responses internally.
- Recommendations: Wrap all DB errors in generic user-facing messages; log the detailed error server-side only; never pass `err.Error()` directly into `WriteError` for 5xx responses.

**AI-Generated CSS Field Served Without Verified Sanitization:**
- Risk: `Game.CSS` is AI-generated or user-supplied CSS stored in the database and sent to the client. A code comment flags that it "Should be validated/parsed strictly to avoid arbitrary code execution."
- Files: `server/obj/structs.go:237–238`, `server/db/game.go:592`
- Current mitigation: Structural validation via `NormalizeJson` limits fields to typed color/font values. Risk depends entirely on how the CSS is injected in the frontend.
- Recommendations: Confirm the frontend only injects CSS as custom property values (not into a `<style>` tag verbatim). Add a server-side schema check that rejects any free-form CSS string.

---

## Performance Bottlenecks

**Session Message Binary Columns Loaded on Full-History Fetch:**
- Problem: `GetAllGameSessionMessages` returns `image` and `audio` columns (potentially MB each) for every message.
- Files: `server/db/sqlc/game.sql.go:562–563`
- Cause: Generated query includes all columns.
- Improvement path: Create a separate query excluding `image` and `audio`; serve binary data via the existing dedicated endpoints only.

**No DB Connection Pool Configuration:**
- Problem: `sql.DB` is created with Go's defaults (unlimited open connections, unlimited idle connections).
- Files: `server/db/init.go:95–98`
- Cause: No `SetMaxOpenConns`, `SetMaxIdleConns`, or `SetConnMaxLifetime` calls after `sql.Open`.
- Improvement path: Set reasonable pool limits; monitor with `sqlDb.Stats()`.

**SSE Silence Timeout Activates Polling While SSE Remains Open:**
- Problem: If no SSE data arrives within 10 seconds, polling starts alongside the still-open SSE connection, running both simultaneously.
- Files: `web/src/features/game-player-v2/hooks/useStreamingSession.ts:34–36`
- Cause: Silence timer starts polling without first closing the SSE connection.
- Improvement path: Abort the SSE connection before starting polling to avoid duplicate work.

**`loadFull` tsparticles Bundle Loaded for Every Game View:**
- Problem: `loadFull` imports the entire tsparticles feature set even when the selected animation is `none`.
- Files: `web/src/features/game-player-v2/components/BackgroundAnimation.tsx:3`
- Cause: `initParticlesEngine` called unconditionally in component.
- Improvement path: Use `loadSlim` instead of `loadFull`; lazy-import the engine only when a non-null animation is active.

---

## Fragile Areas

**Two-Phase Session Initialization Protocol:**
- Files: `server/api/routes/sessions.go:251–277`, `server/game/session_play.go:128–159`
- Why fragile: Session creation returns an empty session; the frontend must call `sendAction("init")` to trigger the opening scene. Detection relies on `messageCount == 1 && action.Message == "init"`. Any legitimate player action that arrives before "init" (e.g., on rapid double-click or race condition) skips the opening scene silently.
- Safe modification: Always test both fresh-start and reload paths when touching session creation or the `useEffect` that calls `sendAction("init")`. The two files must be changed together.
- Test coverage: No integration test directly covering the two-phase init flow.

**Error String Comparison for Session Deletion Access Control:**
- Files: `server/api/routes/sessions.go:304–312`
- Why fragile: The handler gates the 403 response on `err.Error() == "access denied: not the owner of this session"` — a plain string match. If the error message changes (spelling, punctuation, refactoring), the forbidden case degrades silently to a 500.
- Safe modification: Return a typed sentinel error from `db.DeleteGameSession` and use a type switch in the handler, as is done with `obj.ErrForbidden` / `obj.ErrNotFound` elsewhere.
- Test coverage: No dedicated test for the forbidden path.

**Stream Channel Drops Chunks Silently on Full Buffer:**
- Files: `server/game/stream/stream.go:98–110`
- Why fragile: `Send()` discards chunks when the 100-item buffer is full (`default` branch is a no-op). A slow SSE consumer or a disconnected client causes chunks to be dropped permanently with no error signal.
- Safe modification: Log a warning on drop; consider increasing buffer or implementing backpressure.
- Test coverage: Not covered.

**Stream Auto-Cleanup Goroutine Cannot Be Cancelled:**
- Files: `server/game/stream/stream.go:62–66`
- Why fragile: A goroutine calls `time.Sleep(5 * time.Minute)` then `Remove(id)`. If a stream completes and is removed earlier, `Remove` is called a second time. Currently safe (map key already absent), but any refactor that changes the delete-before-check ordering would introduce a double-close panic on the channel.
- Safe modification: Use a `context.WithTimeout` with a cancel function stored on the stream, so normal completion can cancel the cleanup goroutine.
- Test coverage: Not covered by concurrency tests.

**AI Error Classification by String Substring Matching:**
- Files: `server/game/session_play.go:25–53`, `server/game/ai/openai/types.go:161–162`
- Why fragile: AI errors are classified by `strings.Contains` on the lowercased error message. Any AI provider that changes their error message wording causes key-retry and sponsorship-removal logic to fail to trigger, with no observable error.
- Safe modification: Where possible, parse structured error responses from AI APIs (HTTP status codes, JSON error bodies) rather than relying on string content.
- Test coverage: Covered by mock AI tests but not with real provider responses.

**Participant Token Invalid → Silently Falls Through as Unauthenticated:**
- Files: `server/api/httpx/auth.go:244–267`
- Why fragile: When a participant token fails validation (but is not `workshop_inactive`), the middleware calls `next.ServeHTTP` with a nil user instead of returning 401. This is intentional for invite-acceptance flows but means any `OptionalAuth` endpoint accepts broken participant tokens as anonymous requests.
- Safe modification: Document exactly which endpoints depend on this behavior; add a test that confirms broken participant tokens are rejected by `RequireUser`-protected routes.
- Test coverage: `testing/participant_restrictions_test.go` exists but may not cover the invalid-token fallthrough case.

---

## Scaling Limits

**In-Memory Stream Registry and Image Cache:**
- Current capacity: Single Go process; streams map and image cache are Go heap allocations.
- Limit: Multi-instance (horizontal) deployment is impossible without shared state. SSE clients must connect to the same instance processing the AI request.
- Scaling path: Add sticky-session routing (e.g., Nginx upstream hash by session ID) for SSE endpoints, or replace with Redis-backed pub/sub for streams and Redis/S3 for image cache.

**Session Message History Unbounded Growth:**
- Current capacity: Each session accumulates messages indefinitely; binary `image` and `audio` columns exist on every message row.
- Limit: Long-running sessions or sessions with many images will have very large DB rows; `GetAllGameSessionMessages` fetch time grows linearly.
- Scaling path: Paginate message history; archive old messages; move image/audio blobs to an object store with URL references in the DB.

**Per-Session Lock Map Grows Under Load:**
- Current capacity: `sessionLocks.locks` is a `map[uuid.UUID]*lockEntry` that grows with concurrent sessions.
- Limit: Under high load with many concurrent sessions, the map accumulates entries until each `ExpandStory` goroutine finishes. Entries are short-lived in practice but could spike under load.
- Scaling path: Add a cap or periodic eviction; monitor map size in production.

---

## Dependencies at Risk

**Hard-coded AI Model Names:**
- Risk: OpenAI and Mistral model names are hard-coded in platform structs. Providers regularly rename and deprecate models.
- Files: `server/game/ai/openai/openai.go:32–65`
- Impact: When a model is retired, sessions referencing it fail at `ExecuteAction` with an unclear error.
- Migration plan: Make model IDs configurable via the database or server config so they can be updated without a code deploy.

**`gopkg.in/yaml.v2` and `gopkg.in/yaml.v3` Both Required:**
- Risk: Two major YAML library versions are required simultaneously, producing inconsistent parsing behavior and increased binary size.
- Files: `server/go.mod`
- Impact: Minor inconsistency risk; added binary size.
- Migration plan: Audit all `yaml` imports and consolidate on `v3`.

---

## Test Coverage Gaps

**No Frontend Tests:**
- What's not tested: All React components, hooks (`useStreamingSession`, `AuthProvider`, `useTokenManager`), and UI logic have zero test coverage.
- Files: `web/src/features/game-player-v2/hooks/useStreamingSession.ts` (843 lines), `web/src/providers/AuthProvider.tsx` (455 lines), all `web/src/features/*/components/*.tsx`
- Risk: Regressions in the streaming state machine, auth flow, or UI rendering go undetected until manual QA.
- Priority: High for `useStreamingSession` and `AuthProvider`; medium for UI components.

**No Tests for Session Streaming Pipeline:**
- What's not tested: The `DoSessionAction` → `ExpandStory` goroutine → SSE stream → client polling chain.
- Files: `server/game/session_play.go`, `server/game/stream/stream.go`
- Risk: Race conditions in the goroutine handoff (session lock transfer, channel closure) may only appear under load.
- Priority: High.

**No Tests for Two-Phase Session Initialization:**
- What's not tested: The `init` trigger in `DoSessionAction`, the `openingSceneInitiatedRef` guard in the frontend, and error escalation when the init action fails.
- Files: `server/game/session_play.go:128–159`
- Risk: A regression here prevents any game from starting.
- Priority: High.

**No Tests for Image/Audio Cache Lifecycle:**
- What's not tested: `imagecache` cleanup, concurrent updates, the 30-second post-persist removal delay, and `stream.SendImage`/`SendAudio` persistence paths.
- Files: `server/game/imagecache/cache.go`, `server/game/stream/stream.go`
- Risk: Silent image loss or double-close panics under concurrency.
- Priority: Medium.

**No CORS Origin Allowlist Test:**
- What's not tested: Whether the server rejects requests from untrusted origins or reflects any origin back.
- Files: `server/api/httpx/middleware.go:31–44`
- Risk: The wildcard-origin security concern cannot be validated or regression-tested.
- Priority: Medium.

---

*Concerns audit: 2026-03-08*
