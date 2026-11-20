package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// ShareSourceTestSuite tests that shares are created with the correct source
// (personal, organization, workshop) depending on which API key is used.
type ShareSourceTestSuite struct {
	testutil.BaseSuite
}

func TestShareSourceSuite(t *testing.T) {
	s := &ShareSourceTestSuite{}
	s.SuiteName = "Share Source Tests"
	suite.Run(t, s)
}

// TestPersonalKeyCreatesPersonalShare verifies that using a personal API key
// share results in a share with source "personal".
func (s *ShareSourceTestSuite) TestPersonalKeyCreatesPersonalShare() {
	user := s.CreateUser("src-personal")
	keyShare := Must(user.AddApiKey("mock-src-personal", "My Personal Key", "openai"))
	shareID := keyShare.Id.String()

	game := Must(user.UploadGame("alien-first-contact"))
	gameID := game.Id.String()

	// Create share using personal key
	created := Must(user.CreateGameShare(gameID, shareID, nil))
	s.NotEmpty(created.Token, "share should have a token")

	// Check status — should be "personal"
	status := Must(user.GetPrivateShareStatus(gameID))
	s.Require().Len(status.Shares, 1, "should have exactly 1 share")
	s.Equal("personal", status.Shares[0].Source, "share source should be personal")
	s.T().Logf("Personal share created: source=%s", status.Shares[0].Source)
}

// TestOrgKeyCreatesOrgShare verifies that using an organization-shared API key
// results in a share with source "organization".
func (s *ShareSourceTestSuite) TestOrgKeyCreatesOrgShare() {
	admin := s.DevUser()

	// Create institution + head
	inst := Must(admin.CreateInstitution("src-org Org"))
	head := s.CreateUser("src-org-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds API key and shares it with the org
	keyShare := Must(head.AddApiKey("mock-src-org", "Org Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))
	orgShareID := orgShare.Id.String()

	// Head uploads a game
	game := Must(head.UploadGame("alien-first-contact"))
	gameID := game.Id.String()

	// Create share using the org-shared key
	created := Must(head.CreateGameShare(gameID, orgShareID, nil))
	s.NotEmpty(created.Token, "share should have a token")

	// Check status — should be "organization"
	status := Must(head.GetPrivateShareStatus(gameID))
	s.Require().Len(status.Shares, 1, "should have exactly 1 share")
	s.Equal("organization", status.Shares[0].Source, "share source should be organization")
	s.T().Logf("Org share created: source=%s", status.Shares[0].Source)
}

// TestWorkshopShareCreatesWorkshopSource verifies that workshop shares have
// source "workshop" and are tied to the workshop's institution.
func (s *ShareSourceTestSuite) TestWorkshopShareCreatesWorkshopSource() {
	admin := s.DevUser()

	// Create institution + head
	inst := Must(admin.CreateInstitution("src-ws Org"))
	head := s.CreateUser("src-ws-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds API key, shares with org
	keyShare := Must(head.AddApiKey("mock-src-ws", "WS Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))

	// Head creates workshop and sets API key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Source Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	// Enable game sharing
	Must(head.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:             "Source Workshop",
		Active:           true,
		AllowGameSharing: true,
	}))

	// Create workshop invite for participant
	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	s.Require().NotNil(invite.InviteToken)

	// Participant joins workshop and uploads a game (must be public for sharing)
	participant := s.CreateUser("src-ws-participant")
	s.Require().NoError(participant.AcceptWorkshopInviteByToken(*invite.InviteToken))
	game := Must(participant.UploadGame("alien-first-contact"))
	gameID := game.Id.String()
	Must(participant.SetGamePublic(gameID, true))

	// Participant creates workshop share
	created := Must(participant.CreateWorkshopGameShare(gameID, wsIDStr, nil))
	s.NotEmpty(created.Token, "share should have a token")

	// Check status — should be "workshop"
	status := Must(participant.GetPrivateShareStatus(gameID))
	s.Require().Len(status.Shares, 1, "should have exactly 1 share")
	s.Equal("workshop", status.Shares[0].Source, "share source should be workshop")
	s.NotNil(status.Shares[0].WorkshopId, "share should have workshopId set")
	s.NotNil(status.Shares[0].InstitutionId, "workshop share should also have institutionId set")
	s.T().Logf("Workshop share created: source=%v, workshopName=%v", status.Shares[0].Source, status.Shares[0].WorkshopName)
}

// TestHeadSeesWorkshopSharesOutsideWorkshopMode verifies that a head user
// can see workshop shares even when not in workshop mode.
func (s *ShareSourceTestSuite) TestHeadSeesWorkshopSharesOutsideWorkshopMode() {
	admin := s.DevUser()

	// Create institution + head
	inst := Must(admin.CreateInstitution("src-outside Org"))
	head := s.CreateUser("src-outside-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds API key, shares with org
	keyShare := Must(head.AddApiKey("mock-src-outside", "Outside Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))

	// Head creates workshop and sets API key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Outside Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	Must(head.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:             "Outside Workshop",
		Active:           true,
		AllowGameSharing: true,
	}))

	// Create workshop invite for participant
	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	s.Require().NotNil(invite.InviteToken)

	// Participant joins, uploads game (must be public for sharing), creates workshop share
	participant := s.CreateUser("src-outside-part")
	s.Require().NoError(participant.AcceptWorkshopInviteByToken(*invite.InviteToken))
	game := Must(participant.UploadGame("alien-first-contact"))
	gameID := game.Id.String()
	Must(participant.SetGamePublic(gameID, true))
	Must(participant.CreateWorkshopGameShare(gameID, wsIDStr, nil))

	// Head (outside workshop mode) should see the workshop share
	status := Must(head.GetPrivateShareStatus(gameID))
	s.Require().Len(status.Shares, 1, "head should see 1 workshop share outside workshop mode")
	s.Equal("workshop", status.Shares[0].Source, "share source should be workshop")
	s.Require().NotNil(status.Shares[0].WorkshopName, "should include workshop name")
	s.Equal("Outside Workshop", *status.Shares[0].WorkshopName)
	s.T().Logf("Head sees workshop share outside workshop mode: source=%s, workshopName=%v", status.Shares[0].Source, status.Shares[0].WorkshopName)
}
