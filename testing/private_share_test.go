package testing

import (
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// PrivateShareTestSuite tests private share lifecycle, permissions, guest play, and cascade cleanup.
type PrivateShareTestSuite struct {
	testutil.BaseSuite
}

func TestPrivateShareSuite(t *testing.T) {
	s := &PrivateShareTestSuite{}
	s.SuiteName = "Private Share Tests"
	suite.Run(t, s)
}

// setupGameWithKey creates a user with a mock API key and an uploaded game.
// Returns (user, gameID, personalShareID).
// EnablePrivateShare will internally create a game-scoped share for guest play.
func (s *PrivateShareTestSuite) setupGameWithKey(prefix string) (*testutil.UserClient, string, string) {
	user := s.CreateUser(prefix)
	keyShare := Must(user.AddApiKey("mock-"+prefix, prefix+" Key", "mock"))
	game := Must(user.UploadGame("alien-first-contact"))
	return user, game.ID.String(), keyShare.ID.String()
}

// --- Lifecycle ---

// TestEnablePrivateShare verifies enabling private sharing returns a valid token and URL.
func (s *PrivateShareTestSuite) TestEnablePrivateShare() {
	user, gameID, shareID := s.setupGameWithKey("ps-enable")

	status := Must(user.EnablePrivateShare(gameID, shareID, nil))
	s.True(status.Enabled, "share should be enabled")
	s.NotEmpty(status.Token, "share token should not be empty")
	s.Contains(status.ShareURL, "/play/", "share URL should contain /play/")
	s.Nil(status.Remaining, "remaining should be nil for unlimited")
	s.T().Logf("Enabled share: token=%s, url=%s", status.Token, status.ShareURL)
}

// TestGetPrivateShareStatus verifies reading back the share status matches what was set.
func (s *PrivateShareTestSuite) TestGetPrivateShareStatus() {
	user, gameID, shareID := s.setupGameWithKey("ps-status")

	enabled := Must(user.EnablePrivateShare(gameID, shareID, nil))
	s.True(enabled.Enabled)

	status := Must(user.GetPrivateShareStatus(gameID))
	s.True(status.Enabled)
	s.Equal(enabled.Token, status.Token, "token should match")
	s.T().Logf("Status matches: token=%s", status.Token)
}

// TestEnableWithMaxSessions verifies enabling with a session limit.
func (s *PrivateShareTestSuite) TestEnableWithMaxSessions() {
	user, gameID, shareID := s.setupGameWithKey("ps-max")

	maxSessions := 5
	status := Must(user.EnablePrivateShare(gameID, shareID, &maxSessions))
	s.True(status.Enabled)
	s.NotNil(status.Remaining, "remaining should not be nil")
	s.Equal(5, *status.Remaining, "remaining should be 5")
	s.T().Logf("Enabled with max sessions: remaining=%d", *status.Remaining)
}

// TestRevokePrivateShare verifies revoking clears the share.
func (s *PrivateShareTestSuite) TestRevokePrivateShare() {
	user, gameID, shareID := s.setupGameWithKey("ps-revoke")

	Must(user.EnablePrivateShare(gameID, shareID, nil))

	revoked := Must(user.RevokePrivateShare(gameID))
	s.False(revoked.Enabled, "share should be disabled after revoke")
	s.T().Logf("Revoked share")

	// Verify status
	status := Must(user.GetPrivateShareStatus(gameID))
	s.False(status.Enabled, "status should show disabled")
	s.Empty(status.Token, "token should be empty after revoke")
}

// --- Permissions ---

// TestOnlyOwnerCanEnableShare verifies that a non-owner cannot enable private sharing.
func (s *PrivateShareTestSuite) TestOnlyOwnerCanEnableShare() {
	_, gameID, _ := s.setupGameWithKey("ps-perm-a")
	bob := s.CreateUser("ps-perm-b")
	bobKey := Must(bob.AddApiKey("mock-ps-perm-b", "Bob Key", "mock"))

	// Bob tries to enable on alice's game → should fail
	_, err := bob.EnablePrivateShare(gameID, bobKey.ID.String(), nil)
	s.Error(err, "non-owner should not be able to enable private share")
	s.T().Logf("Correctly denied bob enabling share: %v", err)
}

// TestOnlyOwnerCanRevokeShare verifies that a non-owner cannot revoke private sharing.
func (s *PrivateShareTestSuite) TestOnlyOwnerCanRevokeShare() {
	alice, gameID, shareID := s.setupGameWithKey("ps-revperm-a")
	bob := s.CreateUser("ps-revperm-b")

	Must(alice.EnablePrivateShare(gameID, shareID, nil))

	_, err := bob.RevokePrivateShare(gameID)
	s.Error(err, "non-owner should not be able to revoke private share")
	s.T().Logf("Correctly denied bob revoking share: %v", err)
}

// TestNonOwnerCannotReadShareStatusOfPrivateGame verifies that a non-owner cannot
// read share status of a game they cannot see.
// Note: GetPrivateShareStatus uses GetGameByID which allows access to public games,
// so this test uses a private game that bob cannot see.
func (s *PrivateShareTestSuite) TestNonOwnerCannotReadShareStatusOfPrivateGame() {
	alice := s.CreateUser("ps-readperm-a")
	bob := s.CreateUser("ps-readperm-b")

	// Create a private game (no public sponsor setup needed — just test visibility)
	game := Must(alice.UploadGame("alien-first-contact"))
	gameID := game.ID.String()

	_, err := bob.GetPrivateShareStatus(gameID)
	s.Error(err, "non-owner should not be able to read share status of private game")
	s.T().Logf("Correctly denied bob reading status: %v", err)
}

// --- Guest play ---

// TestGuestCanGetGameInfo verifies that a guest can get game info via the share token.
func (s *PrivateShareTestSuite) TestGuestCanGetGameInfo() {
	user, gameID, shareID := s.setupGameWithKey("ps-info")

	status := Must(user.EnablePrivateShare(gameID, shareID, nil))
	s.NotEmpty(status.Token)

	pub := s.Public()
	info := Must(pub.GuestGetGameInfo(status.Token))
	s.NotEmpty(info.Name, "game name should not be empty")
	s.T().Logf("Guest got game info: name=%s", info.Name)
}

// TestInvalidTokenReturnsError verifies that a bogus token returns an error.
func (s *PrivateShareTestSuite) TestInvalidTokenReturnsError() {
	pub := s.Public()
	_, err := pub.GuestGetGameInfo("bogus-token-12345")
	s.Error(err, "invalid token should return error")
	s.T().Logf("Correctly rejected bogus token: %v", err)
}

// TestRevokedTokenReturnsError verifies that a revoked token no longer works.
func (s *PrivateShareTestSuite) TestRevokedTokenReturnsError() {
	user, gameID, shareID := s.setupGameWithKey("ps-revtoken")

	status := Must(user.EnablePrivateShare(gameID, shareID, nil))
	token := status.Token

	// Revoke
	Must(user.RevokePrivateShare(gameID))

	// Token should no longer work
	pub := s.Public()
	_, err := pub.GuestGetGameInfo(token)
	s.Error(err, "revoked token should return error")
	s.T().Logf("Correctly rejected revoked token: %v", err)
}

// TestGuestCanCreateSession verifies the full guest play flow: create session and get a response.
func (s *PrivateShareTestSuite) TestGuestCanCreateSession() {
	user, gameID, shareID := s.setupGameWithKey("ps-session")

	status := Must(user.EnablePrivateShare(gameID, shareID, nil))

	pub := s.Public()
	resp, err := pub.GuestCreateSession(status.Token)
	s.NoError(err, "guest should be able to create session")
	s.NotNil(resp.GameSession, "session should not be nil")
	s.NotEmpty(resp.GameSession.ID, "session ID should not be empty")
	s.GreaterOrEqual(len(resp.Messages), 1, "should have at least 1 initial message")
	s.T().Logf("Guest created session: %s with %d messages", resp.GameSession.ID, len(resp.Messages))
}

// TestGuestCanReloadSession verifies that a guest can reload a session via GET.
func (s *PrivateShareTestSuite) TestGuestCanReloadSession() {
	user, gameID, shareID := s.setupGameWithKey("ps-reload")

	status := Must(user.EnablePrivateShare(gameID, shareID, nil))

	pub := s.Public()
	createResp := Must(pub.GuestCreateSession(status.Token))
	sessionID := createResp.GameSession.ID.String()

	// Reload session
	reloaded := Must(pub.GuestGetSession(status.Token, sessionID))
	s.NotNil(reloaded.GameSession)
	s.Equal(createResp.GameSession.ID, reloaded.GameSession.ID)
	s.GreaterOrEqual(len(reloaded.Messages), 1, "reloaded session should have messages")
	s.T().Logf("Guest reloaded session with %d messages", len(reloaded.Messages))
}

// --- Session limit ---

// TestMaxSessionsEnforcement verifies that the session limit is enforced.
func (s *PrivateShareTestSuite) TestMaxSessionsEnforcement() {
	user, gameID, shareID := s.setupGameWithKey("ps-limit")

	maxSessions := 2
	status := Must(user.EnablePrivateShare(gameID, shareID, &maxSessions))

	pub := s.Public()

	// First session — should succeed
	_, err := pub.GuestCreateSession(status.Token)
	s.NoError(err, "first session should succeed")

	// Second session — should succeed
	_, err = pub.GuestCreateSession(status.Token)
	s.NoError(err, "second session should succeed")

	// Third session — should fail (limit reached)
	_, err = pub.GuestCreateSession(status.Token)
	s.Error(err, "third session should fail — limit reached")
	s.T().Logf("Correctly rejected 3rd session: %v", err)
}

// TestUnlimitedSessions verifies that nil maxSessions allows unlimited sessions.
func (s *PrivateShareTestSuite) TestUnlimitedSessions() {
	user, gameID, shareID := s.setupGameWithKey("ps-unlimited")

	Must(user.EnablePrivateShare(gameID, shareID, nil))

	pub := s.Public()

	// Create 3 sessions — all should succeed
	for i := 0; i < 3; i++ {
		_, err := pub.GuestCreateSession(Must(user.GetPrivateShareStatus(gameID)).Token)
		s.NoError(err, "unlimited session %d should succeed", i+1)
	}
	s.T().Logf("Created 3 unlimited sessions successfully")
}

// --- Cascade ---

// TestDeleteSponsorKeyClearsShare verifies that deleting the sponsor API key disables the share.
func (s *PrivateShareTestSuite) TestDeleteSponsorKeyClearsShare() {
	user, gameID, shareID := s.setupGameWithKey("ps-cascade")

	Must(user.EnablePrivateShare(gameID, shareID, nil))

	// Verify share is enabled
	status := Must(user.GetPrivateShareStatus(gameID))
	s.True(status.Enabled)

	// Delete the API key (cascade) — this should clean up all shares including game-scoped
	MustSucceed(user.DeleteApiKey(shareID, true))
	s.T().Logf("Deleted sponsor API key")

	// Share should now be disabled (sponsor key gone)
	status = Must(user.GetPrivateShareStatus(gameID))
	s.False(status.Enabled, "share should be disabled after sponsor key deletion")
	s.T().Logf("Share correctly disabled after key deletion")
}

// TestRevokeCleanupGuestData verifies that revoking a share cleans up guest users/sessions.
func (s *PrivateShareTestSuite) TestRevokeCleanupGuestData() {
	admin := s.DevUser()
	user, gameID, shareID := s.setupGameWithKey("ps-cleanup")

	status := Must(user.EnablePrivateShare(gameID, shareID, nil))

	// Create a guest session
	pub := s.Public()
	_, err := pub.GuestCreateSession(status.Token)
	s.NoError(err)
	s.T().Logf("Guest session created")

	// Count users before revoke
	usersBefore := Must(admin.GetUsers())

	// Revoke the share — should clean up guest data
	Must(user.RevokePrivateShare(gameID))
	s.T().Logf("Revoked share")

	// Guest user should be cleaned up (fewer users)
	usersAfter := Must(admin.GetUsers())
	s.Less(len(usersAfter), len(usersBefore), "guest user should be cleaned up after revoke")
	s.T().Logf("Users before: %d, after: %d", len(usersBefore), len(usersAfter))
}

// --- Cross-game isolation ---

// TestTokenCannotAccessOtherGameSession verifies that a token for game A
// cannot be used to access a session belonging to game B.
func (s *PrivateShareTestSuite) TestTokenCannotAccessOtherGameSession() {
	// Setup game A
	userA, gameAID, shareAID := s.setupGameWithKey("ps-iso-a")
	statusA := Must(userA.EnablePrivateShare(gameAID, shareAID, nil))

	// Setup game B (separate owner)
	userB, gameBID, shareBID := s.setupGameWithKey("ps-iso-b")
	statusB := Must(userB.EnablePrivateShare(gameBID, shareBID, nil))

	pub := s.Public()

	// Create session on game B
	respB := Must(pub.GuestCreateSession(statusB.Token))
	sessionBID := respB.GameSession.ID.String()

	// Try to access game B's session using game A's token — should fail
	_, err := pub.GuestGetSession(statusA.Token, sessionBID)
	s.Error(err, "token for game A should not access game B's session")
	s.T().Logf("Correctly denied cross-game access: %v", err)
}
