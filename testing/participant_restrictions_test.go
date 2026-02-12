package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ParticipantRestrictionsTestSuite tests that anonymous workshop participants
// (share-link users) are restricted from accessing endpoints meant for
// regular users, heads, or staff.
type ParticipantRestrictionsTestSuite struct {
	testutil.BaseSuite
}

func TestParticipantRestrictionsSuite(t *testing.T) {
	s := &ParticipantRestrictionsTestSuite{}
	s.SuiteName = "Participant Restrictions Tests"
	suite.Run(t, s)
}

// setupWorkshopWithParticipant creates an institution, workshop, and a participant.
// prefix is used to generate unique names per test.
// Returns (head, participant, workshop ID).
func (s *ParticipantRestrictionsTestSuite) setupWorkshopWithParticipant(prefix string) (*testutil.UserClient, *testutil.UserClient, string) {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	workshop := Must(head.CreateWorkshop(inst.ID.String(), prefix+" Workshop"))
	wsIDStr := workshop.ID.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp, err := s.AcceptWorkshopInviteAnonymously(*invite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*resp.AuthToken)
	s.T().Logf("Participant joined: %s (ID: %s)", participant.Name, participant.ID)

	return head, participant, wsIDStr
}

// TestParticipantCannotListInstitutions verifies that a participant cannot
// list institutions (only admin/head/staff can).
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotListInstitutions() {
	_, participant, _ := s.setupWorkshopWithParticipant("li")

	_, err := participant.GetInstitutions()
	s.Error(err, "participant should not be able to list institutions")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotCreateInstitution verifies that a participant cannot
// create an institution.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotCreateInstitution() {
	_, participant, _ := s.setupWorkshopWithParticipant("ci")

	_, err := participant.CreateInstitution("Sneaky Org")
	s.Error(err, "participant should not be able to create an institution")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotListWorkshops verifies that a participant cannot
// list workshops (only head/staff can).
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotListWorkshops() {
	_, participant, _ := s.setupWorkshopWithParticipant("lw")

	// Participant tries to list workshops for the institution
	_, err := participant.ListWorkshops("")
	s.Error(err, "participant should not be able to list workshops")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotCreateWorkshop verifies that a participant cannot
// create a workshop.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotCreateWorkshop() {
	_, participant, _ := s.setupWorkshopWithParticipant("cw")

	_, err := participant.CreateWorkshop("fake-institution-id", "Sneaky Workshop")
	s.Error(err, "participant should not be able to create a workshop")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotUpdateWorkshop verifies that a participant
// cannot update a workshop (only head/staff can).
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotUpdateWorkshop() {
	_, participant, wsIDStr := s.setupWorkshopWithParticipant("uw")

	_, err := participant.UpdateWorkshop(wsIDStr, map[string]interface{}{"name": "Hacked Workshop"})
	s.Error(err, "participant should not be able to update a workshop")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotDeleteOtherUser verifies that a participant cannot
// delete another user.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotDeleteOtherUser() {
	head, participant, _ := s.setupWorkshopWithParticipant("du")

	err := participant.DeleteUser(head.ID)
	s.Error(err, "participant should not be able to delete another user")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotAccessOtherWorkshop verifies that a participant
// cannot read a workshop they don't belong to.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotAccessOtherWorkshop() {
	admin := s.DevUser()
	_, participant, _ := s.setupWorkshopWithParticipant("aw")

	// Create a second institution with a different workshop
	inst2 := Must(admin.CreateInstitution("Other Org"))
	head2 := s.CreateUser("other-head")
	head2Invite := Must(admin.InviteToInstitution(inst2.ID.String(), "head", head2.ID))
	Must(head2.AcceptInvite(head2Invite.ID.String()))
	otherWorkshop := Must(head2.CreateWorkshop(inst2.ID.String(), "Other Workshop"))

	// Participant tries to read the other workshop — should fail
	_, err := participant.GetWorkshop(otherWorkshop.ID.String())
	s.Error(err, "participant should not be able to access a workshop they don't belong to")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotInviteToInstitution verifies that a participant
// cannot create institution invites.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotInviteToInstitution() {
	_, participant, _ := s.setupWorkshopWithParticipant("ii")

	outsider := s.CreateUser("outsider")
	_, err := participant.InviteToInstitution("fake-id", "staff", outsider.ID)
	s.Error(err, "participant should not be able to invite to institution")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotDeleteWorkshop verifies that a participant
// cannot delete a workshop (only head/staff can).
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotDeleteWorkshop() {
	_, participant, wsIDStr := s.setupWorkshopWithParticipant("dw")

	err := participant.DeleteWorkshop(wsIDStr)
	s.Error(err, "participant should not be able to delete a workshop")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotCreateWorkshopInvite verifies that a participant
// cannot create workshop invites (only head/staff can).
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotCreateWorkshopInvite() {
	_, participant, wsIDStr := s.setupWorkshopWithParticipant("wi")

	_, err := participant.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant))
	s.Error(err, "participant should not be able to create workshop invites")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotListUsers verifies that a participant
// cannot list users.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotListUsers() {
	_, participant, _ := s.setupWorkshopWithParticipant("lu")

	_, err := participant.GetUsers()
	s.Error(err, "participant should not be able to list users")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCannotReadOtherUserProfile verifies that a participant
// cannot read another user's profile by ID.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotReadOtherUserProfile() {
	_, participant, _ := s.setupWorkshopWithParticipant("ru")

	outsider := s.CreateUser("ru-outsider")
	var profile obj.User
	err := participant.Get("users/"+outsider.ID, &profile)
	s.Error(err, "participant should not be able to read another user's profile")
	s.T().Logf("Correctly denied: %v", err)
}

// TestParticipantCanSeeOwnProfile verifies that a participant CAN
// read their own profile (basic sanity check).
func (s *ParticipantRestrictionsTestSuite) TestParticipantCanSeeOwnProfile() {
	_, participant, _ := s.setupWorkshopWithParticipant("op")

	me, err := participant.GetMe()
	s.NoError(err, "participant should be able to read own profile")
	s.Equal(participant.Name, me.Name)
	s.T().Logf("Participant can see own profile: %s", me.Name)
}

// TestParticipantCanSeeWorkshopGames verifies that a participant CAN
// list games (they see their workshop games).
func (s *ParticipantRestrictionsTestSuite) TestParticipantCanSeeWorkshopGames() {
	_, participant, _ := s.setupWorkshopWithParticipant("wg")

	// Participant creates a game in their workshop
	game := Must(participant.UploadGame("alien-first-contact"))
	s.NotEmpty(game.ID)

	// Participant can list games (sees their own workshop games)
	games, err := participant.ListGames()
	s.NoError(err, "participant should be able to list games")
	s.T().Logf("Participant sees %d game(s)", len(games))

	found := false
	for _, g := range games {
		if g.ID == game.ID {
			found = true
			break
		}
	}
	s.True(found, "participant should see their own workshop game")
}
