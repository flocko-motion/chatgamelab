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
- Issue: The admin-only `POST /api/restart` endpoint logs a message but never actually restarts the server. The goroutine it spawns waits for `r.Context().Done()` (which fires after the response is sent) then only logs, with a comment saying to use a channel instead of `os.Exit`.
- Files: `server/api/routes/sessions.go:703–722`
- Impact: Admin UI restart button has no effect. The server continues running.
- Fix approach: Send a signal to the main process context (e.g., close a shutdown channel) to trigger graceful shutdown and process restart.

**`organization` and `favorites` Filter Types Fall Back to `all`:**
- Issue: The `GetGames` function accepts `organization` and `favorites` as filter values but treats both as `all`.
- Files: `server/db/game.go:89`
- Impact: Filter UI options that select these values show all visible games instead of scoped results. UI may expose filters that appear to work but do not scope correctly.
- Fix approach: Implement organization and favorites filters or remove them from the public API.

**Legacy `SetCORSHeaders` Function Left in Codebase:**
- Issue: `SetCORSHeaders` (wildcard origin, no credentials) is marked deprecated but remains in the code. It returns `Access-Control-Allow-Origin: *` which would be incompatible with credential-bearing requests.
- Files: `server/api/httpx/response.go:171–175`
- Impact: Low direct risk (the deprecated function is not used by the main middleware), but future copy-paste risk.
- Fix approach: Remove the deprecated function and only keep `SetCORSHeadersWithOrigin`.

**Commented-out Debug Console Logs in Production Code:**
- Issue: Multiple commented-out `console.log('[SSE-DEBUG]')` calls remain in the SSE streaming hook.
- Files: `web/src/features/game-player-v2/hooks/useStreamingSession.ts:333, 335, 504, 511, 772`
- Impact: Noise in code review; small risk of accidental re-enabling during debugging sessions.
- Fix approach: Remove commented debug lines.

**`GetAllUsers` Returns Unbounded Results:**
- Issue: `GetAllUsersWithDetails` SQL query has no LIMIT. For admin list endpoints it fetches every non-deleted user.
- Files: `server/db/sqlc/user.sql.go:651–682`, `server/api/routes/users.go:53–59`
- Impact: As user count grows, this endpoint becomes progressively slower and memory-intensive. Currently low risk but will degrade at scale.
- Fix approach: Add pagination parameters to `GetAllUsers` and its SQL query.

**`GetGameSessionsByGameID` Returns All Sessions Without Limit:**
- Issue: The session listing query for a game has no `LIMIT` clause.
- Files: `server/db/sqlc/game.sql.go:837–838`, `server/api/routes/sessions.go:228–234`
- Impact: A game with many sessions (e.g., public game played by thousands) will return all of them in one query.
- Fix approach: Add a LIMIT or paginate this endpoint.

**`GetAllGameSessionMessages` Returns Entire History Without Limit:**
- Issue: Loading all messages for a session fetches the full message history, including `image` and `audio` binary columns.
- Files: `server/db/sqlc/game.sql.go:562–563`, `server/db/game.go:1441`
- Impact: Long game sessions with image/audio will return multi-megabyte payloads from a single DB call. This is called during the `?messages=all` path on session load.
- Fix approach: Exclude binary columns (`image`, `audio`) from the "all messages" query; serve those only via the dedicated `/messages/{id}/image` and `/messages/{id}/audio` endpoints.

---

## Security Considerations

**CORS Policy Reflects Any Origin Without Allowlist:**
- Risk: `SetCORSHeadersWithOrigin` reflects whatever `Origin` header is sent by the client. There is no origin allowlist. An attacker-controlled page can set any origin and receive credentialed responses.
- Files: `server/api/httpx/response.go:179–186`, `server/api/httpx/middleware.go:33–35`
- Current mitigation: Authentication is required for all sensitive routes; cookies are `HttpOnly`. The main practical risk is CSRF from cross-origin requests using the `cgl_session` cookie.
- Recommendations: Maintain an explicit list of allowed origins (configured from environment); reject unknown origins with a 403 on non-preflight requests.

**JWT Passed as Query Parameter for SSE Endpoints:**
- Risk: JWTs passed as `?token=...` in query parameters are logged in access logs, proxy logs, browser history, and Referer headers.
- Files: `server/api/httpx/auth.go:230–234`, `server/api/routes/workshop_events.go:19`
- Current mitigation: Tokens are only accepted from query param when the `Authorization` header is absent (SSE-only path). Auth0 tokens are short-lived.
- Recommendations: Use a short-lived one-time token exchange for SSE (mint a signed nonce server-side, exchange for the JWT in the SSE handshake) to avoid putting the bearer token in the URL.

**Message/Image/Audio Endpoints Rely on UUID Unguessability:**
- Risk: `/messages/{id}/status`, `/messages/{id}/image`, and `/messages/{id}/audio` have no authentication check. Access is defended solely by the randomness of UUIDs.
- Files: `server/api/routes/sessions.go:383–392`, `server/api/routes/sessions.go:524–556`
- Current mitigation: Code comments explicitly acknowledge this: "No authentication required - message UUIDs are random and unguessable." UUIDs are cryptographically random.
- Recommendations: For production with sensitive game content, add optional bearer-token or session-cookie check on these endpoints. Low priority if game content is not considered sensitive.

**AI-Generated CSS Field Stored and Served Without Sanitization:**
- Risk: The `Game.CSS` field is AI-generated or user-supplied CSS stored in the database and sent to the client. The `obj/structs.go` comment explicitly flags this: "Should be validated/parsed strictly to avoid arbitrary code execution."
- Files: `server/obj/structs.go:237–238`, `server/db/game.go:592`
- Current mitigation: `NormalizeJson` is applied to validate structure against the `obj.CSS` struct (color/font fields only). Structural validation limits the attack surface significantly.
- Recommendations: Verify that CSS served to the frontend is only injected as CSS custom property values (not into a `<style>` tag verbatim). Confirm the CSS schema enforces only the safe structured fields and does not allow free-form CSS strings.

**No HTTP Request Body Size Limit:**
- Risk: The HTTP server has no `MaxBytesReader` applied and no `ReadTimeout`/`WriteTimeout` configured. Large request bodies (e.g., a very large audio base64 payload) or slow clients can hold connections and memory indefinitely.
- Files: `server/api/server.go:44–47` (no timeout fields set on `http.Server`), `server/api/httpx/middleware.go` (no body limiting middleware)
- Current mitigation: None detected.
- Recommendations: Set `ReadTimeout`, `WriteTimeout`, `IdleTimeout` on `http.Server`. Apply `http.MaxBytesReader` in the `ReadJSON` helper or in route handlers that accept file-like payloads (audio upload).

---

## Performance Bottlenecks

**Session Message Binary Columns Loaded on All-Message Fetch:**
- Problem: `GetAllGameSessionMessages` query returns `image` and `audio` columns (potentially MB each) for every message in the session history.
- Files: `server/db/sqlc/game.sql.go:562–563`
- Cause: The `SELECT *` style generated query includes all columns including the binary ones.
- Improvement path: Create a separate `GetAllGameSessionMessageHeaders` query that excludes `image` and `audio`. Binary data is already served via dedicated endpoints.

**No DB Connection Pool Configuration:**
- Problem: `sql.DB` is created with Go's default pool settings (unlimited open connections, unlimited idle connections). Under high concurrency the pool is unbounded.
- Files: `server/db/init.go:95–98`
- Cause: No `SetMaxOpenConns`, `SetMaxIdleConns`, or `SetConnMaxLifetime` calls after `sql.Open`.
- Improvement path: Set reasonable pool limits; monitor with `sqlDb.Stats()`.

**SSE Silence Timeout Activates Polling Redundantly:**
- Problem: If no SSE data arrives within 10 seconds, polling starts alongside the still-open SSE connection. Both run simultaneously until SSE completes.
- Files: `web/src/features/game-player-v2/hooks/useStreamingSession.ts:34–36, 314–327`
- Cause: The silence timer starts polling without first closing the SSE connection.
- Improvement path: When the silence timer fires, abort the SSE connection before starting polling to avoid duplicate work.

---

## Fragile Areas

**Two-Phase Session Initialization Race Condition:**
- Files: `server/api/routes/sessions.go:251–277`, `server/game/session_play.go:128–159`, `web/src/features/game-player-v2/hooks/useStreamingSession.ts:807–821`
- Why fragile: Session creation returns an empty session; the frontend is expected to call `sendAction("init")` to trigger the opening scene. The detection logic checks `messageCount == 1 && action.Message == "init"`. If the client calls something other than "init" first (e.g., page reload triggers a regular player action before init), the opening scene is skipped silently.
- Safe modification: Always test both fresh-start and mid-session-reload paths when touching session creation or the `sendAction` call in the `useEffect`.
- Test coverage: No integration test covering the two-phase init flow directly.

**Stream Registry Timeout is Global `time.Sleep`:**
- Files: `server/game/stream/stream.go:62–66`
- Why fragile: Streams auto-cleanup via a goroutine that calls `time.Sleep(5 * time.Minute)` then `registry.Remove(id)`. If a stream is legitimately removed before the sleep expires, `Remove` is called twice (the second is a no-op only if the channel was already closed, but `Remove` re-checks). There is no way to cancel the sleep goroutine early.
- Safe modification: Track whether a cleanup goroutine is already running, or use a `context.WithTimeout` approach instead of `time.Sleep`.
- Test coverage: Not covered by integration tests.

**Image Cache Cleanup Uses Creation Time, Not Last Access:**
- Files: `server/game/imagecache/cache.go:195–205`
- Why fragile: Cache entries are evicted after `MaxEntryAge` (5 minutes) from creation time. If a large image generation takes close to 5 minutes, its cache entry may be cleaned up before the DB persist goroutine runs (which also sleeps 30 seconds before removing the entry).
- Safe modification: Use `UpdatedAt` instead of `CreatedAt` for expiry, or extend `MaxEntryAge`.
- Test coverage: Not covered.

**Error String Matching for AI Error Classification:**
- Files: `server/game/session_play.go:25–53`, `server/game/ai/openai/types.go:161–162`
- Why fragile: AI errors are classified by substring matching on the lowercased error message string (e.g., `strings.Contains(errStr, "invalid_api_key")`). Any AI provider that changes their error message wording will silently fall back to the generic `ai_error` code, causing incorrect key-retry and sponsorship-removal logic to not trigger.
- Safe modification: Where possible, parse structured error responses from the AI API (HTTP status codes, JSON error bodies) instead of relying on string content.
- Test coverage: Covered by mock AI tests in `server/game/ai/` but not with real provider responses.

---

## Scaling Limits

**In-Memory Stream Registry and Image Cache:**
- Current capacity: Single process; streams and images are stored in Go heap maps.
- Limit: Multi-instance deployment is not possible without shared state. Each server instance has independent stream registries and image caches. SSE clients must connect to the same instance that is processing the AI request.
- Scaling path: Add sticky-session routing (e.g., via Nginx upstream hash by session ID) for SSE endpoints, or replace in-memory stores with a Redis-backed pub/sub for streams and Redis/S3 for image cache.

**Session History Unbounded Growth:**
- Current capacity: Each session accumulates messages indefinitely. Binary image/audio columns exist on every message row.
- Limit: Long-running sessions or sessions with high message counts and images will have very large DB rows. The `GetAllGameSessionMessages` fetch time grows linearly.
- Scaling path: Implement soft pagination on message history; archive or compress old messages; consider moving image/audio blobs out of the `game_session_message` table into an object store.

---

## Dependencies at Risk

**Hard-coded AI Model Names (e.g., `gpt-5.2`, `gpt-image-1.5`):**
- Risk: OpenAI and Mistral model names are hard-coded in the platform structs. OpenAI regularly renames and deprecates models.
- Impact: When a model is retired, sessions referencing it will fail at the `ExecuteAction` call with an unclear error.
- Files: `server/game/ai/openai/openai.go:32–65`
- Migration plan: Make model IDs configurable via the server config or database, so they can be updated without a code deploy.

---

## Test Coverage Gaps

**No Frontend Tests:**
- What's not tested: All React components, hooks (including `useStreamingSession`), and UI logic have zero test coverage.
- Files: `web/src/features/game-player-v2/hooks/useStreamingSession.ts`, `web/src/providers/AuthProvider.tsx`, all `web/src/features/*/components/*.tsx`
- Risk: Regressions in the streaming state machine, auth flow, or UI rendering go undetected until manual QA.
- Priority: High for `useStreamingSession` and `AuthProvider`; medium for UI components.

**No Tests for Session Streaming Pipeline:**
- What's not tested: The `DoSessionAction` → `ExpandStory` goroutine → SSE stream chain is not integration-tested.
- Files: `server/game/session_play.go`, `server/game/stream/stream.go`
- Risk: Race conditions in the goroutine handoff (session lock transfer, channel closure) may only appear under load.
- Priority: High.

**No Tests for Image/Audio Cache Lifecycle:**
- What's not tested: `imagecache` cleanup, concurrent updates, the 30-second post-persist removal delay, and `stream.SendImage`/`SendAudio` persistence paths.
- Files: `server/game/imagecache/cache.go`, `server/game/stream/stream.go`
- Risk: Silent image loss or double-close panics.
- Priority: Medium.

**No Tests for Two-Phase Session Init:**
- What's not tested: The `init` trigger in `DoSessionAction`, the `openingSceneInitiatedRef` guard in the frontend, and error escalation when the init action fails.
- Files: `server/game/session_play.go:128–159`, `web/src/features/game-player-v2/hooks/useStreamingSession.ts:807–821`
- Risk: A regression here prevents any game from starting.
- Priority: High.

---

*Concerns audit: 2026-03-08*
