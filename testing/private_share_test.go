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
	keyShare := Must(user.AddApiKey("mock-"+prefix, prefix+" Key", "openai"))
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
	bobKey := Must(bob.AddApiKey("mock-ps-perm-b", "Bob Key", "openai"))

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
