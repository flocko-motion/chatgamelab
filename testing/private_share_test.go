package testing

import (
	"testing"

	"cgl/testing/testutil"

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
	return user, game.Id.String(), keyShare.Id.String()
}

// --- Lifecycle ---

// TestCreateGameShare verifies creating a share returns a valid token and URL.
func (s *PrivateShareTestSuite) TestCreateGameShare() {
	user, gameID, shareID := s.setupGameWithKey("ps-enable")

	resp := Must(user.CreateGameShare(gameID, shareID, nil))
	s.NotEmpty(resp.Token, "share token should not be empty")
	s.Contains(resp.ShareUrl, "/play/", "share URL should contain /play/")
	s.T().Logf("Created share: token=%s, url=%s", resp.Token, resp.ShareUrl)
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

	MustSucceed(user.DeleteGameShare(gameID, created.Id.String()))
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

	// Bob can't see alice's private game, so for him it doesn't exist
	_, err := bob.CreateGameShare(gameID, bobKey.Id.String(), nil)
	s.NotFound(err, "non-owner should not be able to create share on private game")
}

// TestOnlyOwnerCanDeleteShare verifies that a non-owner cannot delete a share.
func (s *PrivateShareTestSuite) TestOnlyOwnerCanDeleteShare() {
	alice, gameID, shareID := s.setupGameWithKey("ps-revperm-a")
	bob := s.CreateUser("ps-revperm-b")

	created := Must(alice.CreateGameShare(gameID, shareID, nil))

	// Bob can't see alice's private game, so for him neither it nor its share exists
	err := bob.DeleteGameShare(gameID, created.Id.String())
	s.NotFound(err, "non-owner should not be able to delete share")
	s.Len(Must(alice.GetPrivateShareStatus(gameID)).Shares, 1, "the share stays")
}

// TestNonOwnerCannotReadShareStatusOfPrivateGame verifies that a non-owner cannot
// read share status of a game they cannot see.
func (s *PrivateShareTestSuite) TestNonOwnerCannotReadShareStatusOfPrivateGame() {
	alice := s.CreateUser("ps-readperm-a")
	bob := s.CreateUser("ps-readperm-b")

	game := Must(alice.UploadGame("alien-first-contact"))
	gameID := game.Id.String()

	_, err := bob.GetPrivateShareStatus(gameID)
	s.NotFound(err, "non-owner should not be able to read share status of private game")
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
	s.NotFound(err, "invalid token should return error")
}

// TestRevokedTokenReturnsError verifies that a revoked token no longer works.
func (s *PrivateShareTestSuite) TestRevokedTokenReturnsError() {
	user, gameID, shareID := s.setupGameWithKey("ps-revtoken")

	created := Must(user.CreateGameShare(gameID, shareID, nil))
	token := created.Token

	// Delete the share
	MustSucceed(user.DeleteGameShare(gameID, created.Id.String()))

	// Token should no longer work
	pub := s.Public()
	_, err := pub.GuestGetGameInfo(token)
	s.NotFound(err, "revoked token should return error")
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
