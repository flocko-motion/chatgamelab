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
func (s *PrivateShareTestSuite) setupGameWithKey(prefix string) (*testutil.UserClient, string, string) {
	user := s.CreateUser(prefix)
	keyShare := Must(user.AddApiKey("mock-"+prefix, prefix+" Key", "mock"))
	game := Must(user.UploadGame("alien-first-contact"))
	return user, game.ID.String(), keyShare.ID.String()
}

// --- Lifecycle ---

// TestCreateGameShare verifies creating a share returns a valid token and URL.
func (s *PrivateShareTestSuite) TestCreateGameShare() {
	user, gameID, shareID := s.setupGameWithKey("ps-enable")

	resp := Must(user.CreateGameShare(gameID, shareID, nil))
	s.NotEmpty(resp.Token, "share token should not be empty")
	s.Contains(resp.ShareURL, "/play/", "share URL should contain /play/")
	s.T().Logf("Created share: token=%s, url=%s", resp.Token, resp.ShareURL)
}

// TestGetPrivateShareStatus verifies reading back the share status matches what was set.
func (s *PrivateShareTestSuite) TestGetPrivateShareStatus() {
	user, gameID, shareID := s.setupGameWithKey("ps-status")

	created := Must(user.CreateGameShare(gameID, shareID, nil))

	status := Must(user.GetPrivateShareStatus(gameID))
	s.Len(status.Shares, 1, "should have 1 share")
	s.Equal(created.Token, status.Shares[0].Token, "token should match")
	s.T().Logf("Status matches: token=%s", status.Shares[0].Token)
}

// TestCreateWithMaxSessions verifies creating with a session limit.
func (s *PrivateShareTestSuite) TestCreateWithMaxSessions() {
	user, gameID, shareID := s.setupGameWithKey("ps-max")

	maxSessions := 5
	resp := Must(user.CreateGameShare(gameID, shareID, &maxSessions))
	s.NotNil(resp.Remaining, "remaining should not be nil")
	s.Equal(5, *resp.Remaining, "remaining should be 5")
	s.T().Logf("Created with max sessions: remaining=%d", *resp.Remaining)
}

// TestDeleteGameShare verifies deleting a share.
func (s *PrivateShareTestSuite) TestDeleteGameShare() {
	user, gameID, shareID := s.setupGameWithKey("ps-revoke")

	created := Must(user.CreateGameShare(gameID, shareID, nil))

	MustSucceed(user.DeleteGameShare(gameID, created.ID.String()))
	s.T().Logf("Deleted share")

	// Verify status
	status := Must(user.GetPrivateShareStatus(gameID))
	s.Len(status.Shares, 0, "should have no shares after delete")
}

// --- Permissions ---

// TestOnlyOwnerCanCreateShare verifies that a non-owner cannot create a share on a private game.
func (s *PrivateShareTestSuite) TestOnlyOwnerCanCreateShare() {
	_, gameID, _ := s.setupGameWithKey("ps-perm-a")
	bob := s.CreateUser("ps-perm-b")
	bobKey := Must(bob.AddApiKey("mock-ps-perm-b", "Bob Key", "mock"))

	// Bob tries to create share on alice's game -> should fail
	_, err := bob.CreateGameShare(gameID, bobKey.ID.String(), nil)
	s.Error(err, "non-owner should not be able to create share on private game")
	s.T().Logf("Correctly denied bob creating share: %v", err)
}

// TestOnlyOwnerCanDeleteShare verifies that a non-owner cannot delete a share.
func (s *PrivateShareTestSuite) TestOnlyOwnerCanDeleteShare() {
	alice, gameID, shareID := s.setupGameWithKey("ps-revperm-a")
	bob := s.CreateUser("ps-revperm-b")

	created := Must(alice.CreateGameShare(gameID, shareID, nil))

	err := bob.DeleteGameShare(gameID, created.ID.String())
	s.Error(err, "non-owner should not be able to delete share")
	s.T().Logf("Correctly denied bob deleting share: %v", err)
}

// TestNonOwnerCannotReadShareStatusOfPrivateGame verifies that a non-owner cannot
// read share status of a game they cannot see.
func (s *PrivateShareTestSuite) TestNonOwnerCannotReadShareStatusOfPrivateGame() {
	alice := s.CreateUser("ps-readperm-a")
	bob := s.CreateUser("ps-readperm-b")

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

	created := Must(user.CreateGameShare(gameID, shareID, nil))
	s.NotEmpty(created.Token)

	pub := s.Public()
	info := Must(pub.GuestGetGameInfo(created.Token))
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

	created := Must(user.CreateGameShare(gameID, shareID, nil))
	token := created.Token

	// Delete the share
	MustSucceed(user.DeleteGameShare(gameID, created.ID.String()))

	// Token should no longer work
	pub := s.Public()
	_, err := pub.GuestGetGameInfo(token)
	s.Error(err, "revoked token should return error")
	s.T().Logf("Correctly rejected revoked token: %v", err)
}

// TestGuestCanCreateSession verifies the full guest play flow: create session and get a response.
func (s *PrivateShareTestSuite) TestGuestCanCreateSession() {
	user, gameID, shareID := s.setupGameWithKey("ps-session")

	created := Must(user.CreateGameShare(gameID, shareID, nil))

	pub := s.Public()
	resp, streamResult, err := pub.GuestCreateSessionWithStream(created.Token)
	s.NoError(err, "guest should be able to create session")
	s.NotNil(resp.GameSession, "session should not be nil")
	s.NotEmpty(resp.GameSession.ID, "session ID should not be empty")
	s.GreaterOrEqual(len(resp.Messages), 1, "should have at least 1 initial message")
	s.NotNil(streamResult, "stream result should not be nil")
	s.Greater(len(streamResult.Text), 10, "opening scene should have substantial text")
	s.T().Logf("Guest created session: %s with %d messages", resp.GameSession.ID, len(resp.Messages))
}

// TestGuestCanReloadSession verifies that a guest can reload a session via GET.
func (s *PrivateShareTestSuite) TestGuestCanReloadSession() {
	user, gameID, shareID := s.setupGameWithKey("ps-reload")

	created := Must(user.CreateGameShare(gameID, shareID, nil))

	pub := s.Public()
	createResp := Must(pub.GuestCreateSession(created.Token))
	sessionID := createResp.GameSession.ID.String()

	// Reload session
	reloaded := Must(pub.GuestGetSession(created.Token, sessionID))
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
	created := Must(user.CreateGameShare(gameID, shareID, &maxSessions))

	pub := s.Public()

	// First session — should succeed
	_, err := pub.GuestCreateSession(created.Token)
	s.NoError(err, "first session should succeed")

	// Second session — should succeed
	_, err = pub.GuestCreateSession(created.Token)
	s.NoError(err, "second session should succeed")

	// Third session — should fail (limit reached)
	_, err = pub.GuestCreateSession(created.Token)
	s.Error(err, "third session should fail — limit reached")
	s.T().Logf("Correctly rejected 3rd session: %v", err)
}

// TestUnlimitedSessions verifies that nil maxSessions allows unlimited sessions.
func (s *PrivateShareTestSuite) TestUnlimitedSessions() {
	user, gameID, shareID := s.setupGameWithKey("ps-unlimited")

	created := Must(user.CreateGameShare(gameID, shareID, nil))

	pub := s.Public()

	// Create 3 sessions — all should succeed
	for i := 0; i < 3; i++ {
		_, err := pub.GuestCreateSession(created.Token)
		s.NoError(err, "unlimited session %d should succeed", i+1)
	}
	s.T().Logf("Created 3 unlimited sessions successfully")
}

// --- Cascade ---

// TestDeleteSponsorKeyClearsShare verifies that deleting the sponsor API key disables the share.
func (s *PrivateShareTestSuite) TestDeleteSponsorKeyClearsShare() {
	user, gameID, shareID := s.setupGameWithKey("ps-cascade")

	Must(user.CreateGameShare(gameID, shareID, nil))

	// Verify share exists
	status := Must(user.GetPrivateShareStatus(gameID))
	s.Len(status.Shares, 1)

	// Delete the API key (cascade) — this should clean up all shares including game-scoped
	MustSucceed(user.DeleteApiKey(shareID, true))
	s.T().Logf("Deleted sponsor API key")

	// Share should now be gone (sponsor key gone)
	status = Must(user.GetPrivateShareStatus(gameID))
	s.Len(status.Shares, 0, "shares should be empty after sponsor key deletion")
	s.T().Logf("Share correctly removed after key deletion")
}

// TestRevokeCleanupGuestData verifies that deleting a share cleans up guest users/sessions.
func (s *PrivateShareTestSuite) TestRevokeCleanupGuestData() {
	admin := s.DevUser()
	user, gameID, shareID := s.setupGameWithKey("ps-cleanup")

	created := Must(user.CreateGameShare(gameID, shareID, nil))

	// Create a guest session
	pub := s.Public()
	_, err := pub.GuestCreateSession(created.Token)
	s.NoError(err)
	s.T().Logf("Guest session created")

	// Count users before delete
	usersBefore := Must(admin.GetUsers())

	// Delete the share — should clean up guest data
	MustSucceed(user.DeleteGameShare(gameID, created.ID.String()))
	s.T().Logf("Deleted share")

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
	createdA := Must(userA.CreateGameShare(gameAID, shareAID, nil))

	// Setup game B (separate owner)
	userB, gameBID, shareBID := s.setupGameWithKey("ps-iso-b")
	createdB := Must(userB.CreateGameShare(gameBID, shareBID, nil))

	pub := s.Public()

	// Create session on game B
	respB := Must(pub.GuestCreateSession(createdB.Token))
	sessionBID := respB.GameSession.ID.String()

	// Try to access game B's session using game A's token — should fail
	_, err := pub.GuestGetSession(createdA.Token, sessionBID)
	s.Error(err, "token for game A should not access game B's session")
	s.T().Logf("Correctly denied cross-game access: %v", err)
}
