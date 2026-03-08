# Testing Patterns

**Analysis Date:** 2026-03-08

## Overview

Testing is **backend-only (Go integration tests)**. The frontend (`web/`) has no test framework configured — no Jest, Vitest, or any test files exist. All testing is in the `testing/` module which runs full integration tests against a live backend and real PostgreSQL database.

---

## Test Framework

**Runner:**
- Go's standard `testing` package + `github.com/stretchr/testify v1.10.0`
- Test suite organization via `github.com/stretchr/testify/suite`
- Report output via `gotestsum` (installed separately: `go install gotest.tools/gotestsum@latest`)
- Config: `testing/go.mod` (separate Go module referencing `../server` via `replace` directive)

**Assertion Library:**
- `testify/suite` embedded assertions (`s.Equal`, `s.NoError`, `s.Require().NoError`, etc.)
- `s.Require().Xxx` — fatal assertion (stops current test immediately)
- `s.Xxx` — non-fatal assertion (test continues)

**Run Commands:**
```bash
./run-test.sh                    # Default: exclude AI tests, dots-v2 output
./run-test.sh --verbose          # Full verbose output
./run-test.sh -v                 # Short: full verbose output
./run-test.sh --ai               # Include AI-tagged tests
./run-test.sh --only-ai          # Run ONLY AI tests (TestGameEngineSuite)
./run-test.sh --only-ai --verbose # AI tests with verbose output
```

**In CI:**
```bash
cd testing && go test -v ./...
```

---

## Test File Organization

**Location:**
- All integration tests live in `testing/` directory (separate Go module)
- Infrastructure helpers in `testing/testutil/` subdirectory
- Test data (e.g., game YAML files) in `testing/testdata/`

**File naming:**
- `{domain}_test.go` — test suites for a specific domain
- `{domain}_{scenario}_test.go` — test suites for specific scenarios
- `helpers.go` — package-level test utility functions
- `testutil/suite.go` — `BaseSuite` struct and lifecycle management
- `testutil/testclient.go` — `UserClient` and `PublicClient` HTTP clients

**Package:**
- All test files use `package testing` (not `_test` suffix)
- testutil helpers use `package testutil`

**Test files present:**
```
testing/
├── apikey_cascade_org_share_test.go
├── apikey_cascade_private_share_test.go
├── api_key_fallback_test.go
├── apikey_isolation_test.go
├── apikey_test.go
├── free_use_key_test.go
├── game_crud_test.go
├── game_engine_test.go         # AI tests (build tag: ai_tests)
├── game_permissions_test.go
├── helpers.go
├── institution_deletion_test.go
├── institution_invite_test.go
├── institution_membership_test.go
├── institution_workshop_test.go
├── integration_test.go
├── participant_restrictions_test.go
├── private_share_test.go
├── system_settings_test.go
├── unauthenticated_access_test.go
├── user_access_test.go
├── user_deleted_access_test.go
├── user_deletion_test.go
├── user_test.go
├── workshop_apikey_test.go
├── workshop_game_visibility_test.go
├── workshop_individual_test.go
├── workshop_settings_permissions_test.go
└── testutil/
    ├── suite.go
    └── testclient.go
```

---

## Test Suite Structure

**Each test suite embeds `testutil.BaseSuite`:**
```go
type InstitutionInviteTestSuite struct {
    testutil.BaseSuite
}

func TestInstitutionInviteSuite(t *testing.T) {
    s := &InstitutionInviteTestSuite{}
    s.SuiteName = "Institution Invite Tests"  // human-readable name in output
    suite.Run(t, s)
}
```

**Suite setup pattern:**
```go
// Optional: override SetupSuite to create shared users for the whole suite
func (s *GameEngineTestSuite) SetupSuite() {
    s.BaseSuite.SetupSuite()  // MUST call this first
    s.clientAlice = s.CreateUser("alice-game-engine")
}
```

**Individual test methods:**
```go
func (s *InstitutionInviteTestSuite) TestHeadCanInviteToInstitution() {
    admin := s.DevUser()

    inst := Must(admin.CreateInstitution("Head Invite Org"))
    head := s.CreateUser("hi-head")
    headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
    Must(head.AcceptInvite(headInvite.ID.String()))

    invite, err := head.InviteToInstitution(inst.ID.String(), "staff", newUser.ID)
    s.NoError(err, "head should be able to invite to institution")
    s.NotEmpty(invite.ID)
}
```

---

## Infrastructure — BaseSuite Lifecycle

`testutil.BaseSuite` (at `testing/testutil/suite.go`) manages the full test environment:

**`SetupSuite()` (runs once per suite):**
1. Acquires a global `sync.Mutex` (only one suite runs at a time)
2. Finds two free TCP ports (Postgres + backend)
3. Starts a `postgres:18` Docker container on a random port
4. Waits for Postgres to be ready (`pg_isready` polling)
5. Starts the backend in-process via `api.RunServer(ctx, port, devMode, readyChan)`
6. Waits for backend HTTP readiness via `/api/status`
7. Creates a dev admin user via the dev-mode endpoint

**`TearDownSuite()` (runs once per suite):**
1. Cancels the backend context (shuts down server)
2. Releases the mutex
3. Postgres container is left running for test speed (cleaned up on next suite start)

**Global mutex:** Only one suite runs at a time due to `var testLock sync.Mutex` — suites are NOT parallelized.

---

## Test Clients

**`UserClient`** (`testing/testutil/testclient.go`): Represents an authenticated user.

Fields: `Name`, `ID`, `Email`, `Token string`, `t *testing.T`

**Methods — must succeed (panic on error):**
```go
alice.MustGet("games", &result)           // GET, panics if error
alice.MustPost("games/new", payload, &result) // POST, panics if error
```

**Methods — expect failure:**
```go
alice.FailPost("games/new", payload)      // expects HTTP error, panics if success
alice.FailGet("games/"+id, &result)
```

**Methods — return error:**
```go
err := alice.Get("games", &result)        // returns error (don't panic)
err := alice.Post("games/new", payload, &result)
```

**`PublicClient`** — unauthenticated client, same method signatures.

**Obtaining clients in tests:**
```go
admin := s.DevUser()                      // pre-seeded admin user
alice := s.CreateUser("alice")            // creates user, returns UserClient
alice := s.CreateUser("alice", "alice@example.com") // with email
public := s.Public()                      // unauthenticated
```

---

## Helper Functions

In `testing/helpers.go`:

```go
// Must — panics if err != nil, returns value (use for "must succeed" calls)
result := Must(alice.CreateInstitution("My Org"))

// Fail — panics if no error (use for "must fail" calls)
Fail(alice.CreateInstitution("Bad Name"))

// MustSucceed — panics if err != nil (use for void operations)
MustSucceed(alice.DeleteGame(gameID))

// MustFail — panics if err == nil (use for void operations expected to fail)
MustFail(alice.DeleteGame(unauthorizedGameID))
```

These panics are caught by the test framework and reported as test failures with location.

---

## AI Tests (Build Tag)

AI tests that call external AI APIs require the `ai_tests` build tag:

```go
//go:build ai_tests

package testing
```

These are in `testing/game_engine_test.go` and run only via:
```bash
./run-test.sh --ai        # include AI tests
./run-test.sh --only-ai   # only AI tests
```

AI tests require real API keys available in the environment.

---

## Assertion Patterns

**Fatal vs non-fatal assertions:**
```go
// Fatal — stops current test, use for preconditions
s.Require().NoError(err, "should create game")
s.Require().NotNil(result, "result must not be nil")
s.Require().Len(items, 3, "should have 3 items")

// Non-fatal — reports failure but continues, use for independent checks
s.NoError(err, "optional operation should succeed")
s.Equal("expected", actual, "should match")
s.NotEmpty(result.ID)
s.Greater(len(msg), 10, "message should be substantial")
```

**Logging in tests (use for diagnostics):**
```go
s.T().Logf("Created user %q (ID: %s)", name, id)
s.T().Logf("Found %d games", len(games))
```

**Error validation (from testutil):**
```go
// Validate error type without failing test
testutil.ErrorPrefix("forbidden")(err)  // true if error starts with "forbidden"
testutil.ErrorContains("not found")(err) // true if error contains "not found"
```

---

## Error Testing Pattern

To assert that an action correctly fails:
```go
// Pattern 1: use Fail helper (panics if no error)
Fail(alice.CreateGame(invalidPayload))

// Pattern 2: use FailPost/FailGet on test client
alice.FailPost("admin/users", payload)

// Pattern 3: manual check
_, err := alice.Get("unauthorized/resource", &result)
s.Error(err, "should get an error")
```

---

## Async / Streaming Tests

For SSE streaming (game session responses):
```go
sessionResponse, initialStream, err := s.clientAlice.CreateGameSessionWithStream(gameID)
s.Require().NoError(err, "CreateGameSessionWithStream failed")

// streamResult contains accumulated SSE data
s.Greater(len(streamResult.AudioData), 0, "audio data should be received")
```

---

## Coverage

**Requirements:** None enforced.

**Scope:** Integration tests only — no unit tests. All tests make real HTTP calls to a real in-process server with a real Postgres database.

---

## Frontend Testing

**Status: Not configured.**

No test framework (Jest, Vitest, Playwright, etc.) is installed in `web/`. No `*.test.*` or `*.spec.*` files exist. No test scripts in `web/package.json`.

If adding frontend tests, use Vitest (aligns with Vite toolchain already in use).

---

*Testing analysis: 2026-03-08*
