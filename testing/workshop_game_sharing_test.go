package testing

import (
	"testing"

	"cgl/testing/testutil"

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
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds API key, shares with org
	keyShare := Must(head.AddApiKey("mock-ws-share-"+prefix, prefix+" Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))

	// Head creates workshop and sets API key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), prefix+" Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	// Enable allowGameSharing if requested
	Must(head.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:             prefix + " Workshop",
		Active:           true,
		AllowGameSharing: allowSharing,
	}))

	// Create workshop invite for individual
	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	s.Require().NotNil(invite.InviteToken)

	// Individual joins workshop
	individual := s.CreateUser(prefix + "-individual")
	s.Require().NoError(individual.AcceptWorkshopInviteByToken(*invite.InviteToken))
	s.T().Logf("Individual joined workshop: %s", individual.Name)

	// Individual uploads a game in the workshop and makes it public (required for sharing)
	game := Must(individual.UploadGame("alien-first-contact"))
	gameIDStr := game.Id.String()
	Must(individual.SetGamePublic(gameIDStr, true))

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
	s.NotEmpty(share.ShareUrl, "share should have a URL")
	s.NotNil(share.WorkshopId, "share should have workshopId set")
	s.T().Logf("Share created: token=%s, url=%s, workshopId=%v", share.Token, share.ShareUrl, share.WorkshopId)
}

// TestIndividualCannotShareWhenDisabled verifies that a workshop individual
// cannot create a share link when allowGameSharing is disabled.
func (s *WorkshopGameSharingTestSuite) TestIndividualCannotShareWhenDisabled() {
	_, individual, wsID, gameID := s.setupWorkshopWithSharing("share-no", false)

	_, err := individual.CreateWorkshopGameShare(gameID, wsID, nil)
	s.Forbidden(err, "individual should NOT be able to share when allowGameSharing is disabled")
}

// TestGuestCanOpenWorkshopShare verifies that a Guest can open
// a workshop share link created by an individual.
func (s *WorkshopGameSharingTestSuite) TestGuestCanOpenWorkshopShare() {
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
	s.Equal(share1.Id, share2.Id, "second call should return the same share ID")
	s.T().Logf("Both calls returned same share: id=%s, token=%s", share1.Id, share1.Token)
}
