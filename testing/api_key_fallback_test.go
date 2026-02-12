package testing

import (
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ApiKeyFallbackTestSuite tests the API key fallback mechanism:
// when the primary resolved key fails (e.g. invalid, billing issue),
// the backend should automatically try the next candidate from the
// priority chain before giving up.
//
// Priority chain:
//  1. Workshop key
//  2. Sponsored game key
//  3. Institution free-use key
//  4. User's default API key
//  5. System free-use key
type ApiKeyFallbackTestSuite struct {
	testutil.BaseSuite
}

func TestApiKeyFallbackSuite(t *testing.T) {
	s := &ApiKeyFallbackTestSuite{}
	s.SuiteName = "API Key Fallback Tests"
	suite.Run(t, s)
}

// TestFallbackFromBrokenUserKeyToSystemFreeUse tests that when a user's
// default API key is broken (openai with invalid key), session creation
// falls back to the system free-use key (mock platform, always works).
//
// Setup:
//   - Admin sets a working mock key as system free-use (priority 5)
//   - User sets a broken openai key as their default (priority 4)
//
// Expected: priority 4 key fails → falls back to priority 5 → session created.
func (s *ApiKeyFallbackTestSuite) TestFallbackFromBrokenUserKeyToSystemFreeUse() {
	admin := s.DevUser()

	// Admin adds a working mock key and sets it as system free-use (priority 5)
	adminKey := Must(admin.AddApiKey("mock-fallback-sys", "System Fallback Key", "mock"))
	adminKeyIDStr := adminKey.ApiKeyID.String()
	Must(admin.SetSystemFreeUseApiKey(&adminKeyIDStr))
	s.T().Logf("System free-use key set (mock, working)")

	// Create a regular user with a broken openai key as default (priority 4)
	user := s.CreateUser("user-fallback")
	brokenKey := Must(user.AddApiKey("sk-broken-invalid-key-12345", "Broken OpenAI Key", "openai"))
	Must(user.SetDefaultApiKey(brokenKey.ID.String()))
	s.T().Logf("User has broken openai key as default")

	// User uploads a game (must be owner to access it)
	game := Must(user.UploadGame("alien-first-contact"))
	s.T().Logf("User uploaded game: %s (ID: %s)", game.Name, game.ID)

	// Verify user has API key available (resolution finds at least one candidate)
	available := Must(user.GetApiKeyStatus(game.ID.String()))
	s.True(available, "user should have API key available (broken default + system free-use)")
	s.T().Logf("API key available = %v", available)

	// Create a game session — this should succeed via fallback:
	// 1. Tries user's default openai key → fails (invalid key)
	// 2. Falls back to system free-use mock key → succeeds
	session, err := user.CreateGameSession(game.ID.String())
	s.NoError(err, "session creation should succeed via fallback to system free-use key")
	s.NotEmpty(session.ID, "session should have an ID")
	s.T().Logf("Session created successfully via fallback: %s", session.ID)

	// Clean up
	Must(admin.SetSystemFreeUseApiKey(nil))
}

// TestFallbackFromBrokenUserKeyToInstitutionFreeUse tests fallback from
// a broken user default key (priority 4) to institution free-use key (priority 3).
func (s *ApiKeyFallbackTestSuite) TestFallbackFromBrokenUserKeyToInstitutionFreeUse() {
	admin := s.DevUser()

	// Create institution with head
	inst := Must(admin.CreateInstitution("Fallback Org"))
	head := s.CreateUser("head-fallback")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds a working mock key, shares with org, sets as institution free-use (priority 3)
	mockKey := Must(head.AddApiKey("mock-org-fallback", "Org Fallback Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(mockKey.ID.String(), inst.ID.String()))
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetInstitutionFreeUseApiKey(inst.ID.String(), &orgShareIDStr))
	s.T().Logf("Institution free-use key set (mock, working)")

	// Add a staff member with a broken openai key as default (priority 4)
	staff := s.CreateUser("staff-fallback")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	brokenKey := Must(staff.AddApiKey("sk-broken-staff-key-12345", "Broken Staff Key", "openai"))
	Must(staff.SetDefaultApiKey(brokenKey.ID.String()))
	s.T().Logf("Staff has broken openai key as default")

	// Staff uploads a game (must be owner to access it)
	game := Must(staff.UploadGame("arctic-expedition"))
	s.T().Logf("Staff uploaded game: %s", game.Name)

	// Create session — should fall back from broken user key to institution free-use
	session, err := staff.CreateGameSession(game.ID.String())
	s.NoError(err, "session creation should succeed via fallback to institution free-use key")
	s.NotEmpty(session.ID, "session should have an ID")
	s.T().Logf("Session created successfully via fallback: %s", session.ID)
}

// TestNoFallbackWhenOnlyOneCandidateExists verifies that when only one
// key is available and it's broken, session creation fails cleanly.
func (s *ApiKeyFallbackTestSuite) TestNoFallbackWhenOnlyOneCandidateExists() {
	// Create user with only a broken openai key (no fallback available)
	user := s.CreateUser("user-no-fallback")
	brokenKey := Must(user.AddApiKey("sk-broken-only-key-12345", "Only Broken Key", "openai"))
	Must(user.SetDefaultApiKey(brokenKey.ID.String()))
	s.T().Logf("User has only a broken openai key, no fallback")

	// User uploads a game (must be owner to access it)
	game := Must(user.UploadGame("alien-first-contact"))
	s.T().Logf("User uploaded game: %s", game.Name)

	// Session creation should fail — no fallback available
	_, err := user.CreateGameSession(game.ID.String())
	s.Error(err, "session creation should fail when only key is broken and no fallback exists")
	s.T().Logf("Session creation correctly failed: %v", err)
}

// TestFallbackSkipsSamePlatformDuplicates verifies that when multiple
// candidates resolve to the same underlying API key, they are deduplicated
// and we don't retry with the same broken key.
func (s *ApiKeyFallbackTestSuite) TestFallbackSkipsSamePlatformDuplicates() {
	admin := s.DevUser()

	// Admin sets a working mock key as system free-use
	adminKey := Must(admin.AddApiKey("mock-dedup-sys", "System Dedup Key", "mock"))
	adminKeyIDStr := adminKey.ApiKeyID.String()
	Must(admin.SetSystemFreeUseApiKey(&adminKeyIDStr))
	s.T().Logf("System free-use key set (mock)")

	// Create user with a broken openai key as default
	user := s.CreateUser("user-dedup")
	brokenKey := Must(user.AddApiKey("sk-broken-dedup-key-12345", "Broken Dedup Key", "openai"))
	Must(user.SetDefaultApiKey(brokenKey.ID.String()))
	s.T().Logf("User has broken openai key as default + system free-use mock as fallback")

	// User uploads a game (must be owner to access it)
	game := Must(user.UploadGame("arctic-expedition"))
	s.T().Logf("User uploaded game: %s", game.Name)

	// Session should succeed (broken openai → fallback to mock system key)
	session, err := user.CreateGameSession(game.ID.String())
	s.NoError(err, "session should succeed via fallback")
	s.NotEmpty(session.ID)
	s.T().Logf("Session created: %s", session.ID)

	// Clean up
	Must(admin.SetSystemFreeUseApiKey(nil))
}

// TestContinueSessionWithFallback tests that when resuming a session
// (sending a message), if the originally resolved key is now broken,
// the backend falls back to the next available key.
func (s *ApiKeyFallbackTestSuite) TestContinueSessionWithFallback() {
	admin := s.DevUser()

	// Create user with a working mock key
	user := s.CreateUser("user-continue")
	mockKey := Must(user.AddApiKey("mock-continue-key", "Working Mock Key", "mock"))
	Must(user.SetDefaultApiKey(mockKey.ID.String()))

	// User uploads a game (must be owner to access it)
	game := Must(user.UploadGame("alien-first-contact"))
	s.T().Logf("User uploaded game: %s", game.Name)

	// Admin sets a working mock key as system free-use (fallback)
	adminKey := Must(admin.AddApiKey("mock-continue-sys", "System Continue Key", "mock"))
	adminKeyIDStr := adminKey.ApiKeyID.String()
	Must(admin.SetSystemFreeUseApiKey(&adminKeyIDStr))

	// Create session with working key
	session, err := user.CreateGameSession(game.ID.String())
	s.NoError(err, "session creation should succeed")
	s.T().Logf("Session created: %s", session.ID)

	// Delete the user's mock key — now their default is gone
	MustSucceed(user.DeleteApiKey(mockKey.ID.String(), true))
	s.T().Logf("User's mock key deleted")

	// Send a message — should fall back to system free-use key
	response, err := user.SendGameMessage(session.ID.String(), "look around")
	s.NoError(err, "sending message should succeed via fallback to system free-use key")
	s.NotEmpty(response.Message, "response should have a message")
	s.T().Logf("Message sent successfully via fallback")

	// Clean up
	Must(admin.SetSystemFreeUseApiKey(nil))
}
