package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WorkshopGameSharingTestSuite tests that workshop individuals can share games
// when allowGameSharing is enabled, and that the workshop API key is used.
type WorkshopGameSharingTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopGameSharingSuite(t *testing.T) {
	s := &WorkshopGameSharingTestSuite{}
	s.SuiteName = "Workshop Game Sharing Tests"
	suite.Run(t, s)
}

// setupWorkshopWithSharing creates a full workshop setup with sharing enabled.
// Returns (head, individual, workshopID, gameID).
// The individual has joined the workshop and uploaded a game.
func (s *WorkshopGameSharingTestSuite) setupWorkshopWithSharing(prefix string, allowSharing bool) (*testutil.UserClient, *testutil.UserClient, string, string) {
	admin := s.DevUser()

	// Create institution + head
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds API key, shares with org
	keyShare := Must(head.AddApiKey("mock-ws-share-"+prefix, prefix+" Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))

	// Head creates workshop and sets API key
	workshop := Must(head.CreateWorkshop(inst.ID.String(), prefix+" Workshop"))
	wsIDStr := workshop.ID.String()
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	// Enable allowGameSharing if requested
	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":             prefix + " Workshop",
		"active":           true,
		"public":           false,
		"allowGameSharing": allowSharing,
		"isPaused":         false,
	}))

	// Create workshop invite for individual
	invite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	s.Require().NotNil(invite.InviteToken)

	// Individual joins workshop
	individual := s.CreateUser(prefix + "-individual")
	s.Require().NoError(individual.AcceptWorkshopInviteByToken(*invite.InviteToken))
	s.T().Logf("Individual joined workshop: %s", individual.Name)

	// Individual uploads a game in the workshop
	game := Must(individual.UploadGame("alien-first-contact"))
	gameIDStr := game.ID.String()

	s.T().Logf("Setup: head=%s, individual=%s, ws=%s, game=%s", head.Name, individual.Name, wsIDStr, gameIDStr)
	return head, individual, wsIDStr, gameIDStr
}

// TestIndividualCanShareWhenEnabled verifies that a workshop individual can create
// a share link when allowGameSharing is enabled.
func (s *WorkshopGameSharingTestSuite) TestIndividualCanShareWhenEnabled() {
	_, individual, wsID, gameID := s.setupWorkshopWithSharing("share-ok", true)

	share, err := individual.CreateWorkshopGameShare(gameID, wsID, nil)
	s.NoError(err, "individual should be able to create workshop share")
	s.NotEmpty(share.Token, "share should have a token")
	s.NotEmpty(share.ShareURL, "share should have a URL")
	s.NotNil(share.WorkshopID, "share should have workshopId set")
	s.T().Logf("Share created: token=%s, url=%s, workshopId=%v", share.Token, share.ShareURL, share.WorkshopID)
}

// TestIndividualCannotShareWhenDisabled verifies that a workshop individual
// cannot create a share link when allowGameSharing is disabled.
func (s *WorkshopGameSharingTestSuite) TestIndividualCannotShareWhenDisabled() {
	_, individual, wsID, gameID := s.setupWorkshopWithSharing("share-no", false)

	_, err := individual.CreateWorkshopGameShare(gameID, wsID, nil)
	s.Error(err, "individual should NOT be able to share when allowGameSharing is disabled")
	s.Contains(err.Error(), "403", "should be a 403 forbidden error")
	s.T().Logf("Correctly rejected: %v", err)
}

// TestGuestCanPlayViaWorkshopShare verifies that an anonymous guest can play
// a game via a workshop share link created by an individual.
func (s *WorkshopGameSharingTestSuite) TestGuestCanPlayViaWorkshopShare() {
	_, individual, wsID, gameID := s.setupWorkshopWithSharing("share-play", true)

	// Individual creates share
	share, err := individual.CreateWorkshopGameShare(gameID, wsID, nil)
	s.Require().NoError(err)
	s.Require().NotEmpty(share.Token)

	// Guest can get game info
	pub := s.Public()
	info, err := pub.GuestGetGameInfo(share.Token)
	s.NoError(err, "guest should be able to get game info")
	s.NotEmpty(info.Name, "game info should have a name")
	s.T().Logf("Guest game info: name=%s", info.Name)

	// Guest can create session
	session, err := pub.GuestCreateSession(share.Token)
	s.NoError(err, "guest should be able to create session")
	s.NotNil(session.GameSession, "session should not be nil")
	s.T().Logf("Guest session created: %s", session.ID)
}

// TestWorkshopShareReusesExistingLink verifies that creating the same workshop
// share twice returns the same token (reuse, not duplicate).
func (s *WorkshopGameSharingTestSuite) TestWorkshopShareReusesExistingLink() {
	_, individual, wsID, gameID := s.setupWorkshopWithSharing("share-reuse", true)

	// Create share twice
	share1, err := individual.CreateWorkshopGameShare(gameID, wsID, nil)
	s.Require().NoError(err)

	share2, err := individual.CreateWorkshopGameShare(gameID, wsID, nil)
	s.Require().NoError(err)

	s.Equal(share1.Token, share2.Token, "second call should return the same token")
	s.Equal(share1.ID, share2.ID, "second call should return the same share ID")
	s.T().Logf("Both calls returned same share: id=%s, token=%s", share1.ID, share1.Token)
}
