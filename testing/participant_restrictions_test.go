package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// ParticipantRestrictionsTestSuite tests that anonymous workshop participants
// (share-link users) are refused on endpoints meant for regular users, heads
// or staff.
type ParticipantRestrictionsTestSuite struct {
	testutil.BaseSuite
}

func TestParticipantRestrictionsSuite(t *testing.T) {
	s := &ParticipantRestrictionsTestSuite{}
	s.SuiteName = "Participant Restrictions Tests"
	suite.Run(t, s)
}

// setupWorkshopWithParticipant creates an institution with a head, a
// workshop, and a participant who joined it by its invite link. prefix keeps
// names unique per test.
// Returns (head, participant, institution ID, workshop ID).
func (s *ParticipantRestrictionsTestSuite) setupWorkshopWithParticipant(prefix string) (*testutil.UserClient, *testutil.UserClient, string, string) {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), prefix+" Workshop"))
	invite := Must(head.CreateWorkshopInvite(workshop.Id.String()))
	participant := Must(s.JoinWorkshop(*invite.InviteToken))

	return head, participant, inst.Id.String(), workshop.Id.String()
}

// Only admins, heads and staff list institutions.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotListInstitutions() {
	_, participant, _, _ := s.setupWorkshopWithParticipant("li")

	_, err := participant.ListInstitutions()
	s.Forbidden(err)
}

func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotCreateInstitution() {
	_, participant, _, _ := s.setupWorkshopWithParticipant("ci")

	_, err := participant.CreateInstitution("Sneaky Org")
	s.Forbidden(err)
}

// Only heads and staff list workshops, all of them or their institution's.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotListWorkshops() {
	_, participant, instID, _ := s.setupWorkshopWithParticipant("lw")

	_, err := participant.ListAllWorkshops()
	s.Forbidden(err)
	_, err = participant.ListWorkshops(instID)
	s.Forbidden(err)
}

// Not even in the institution of their own workshop.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotCreateWorkshop() {
	_, participant, instID, _ := s.setupWorkshopWithParticipant("cw")

	_, err := participant.CreateWorkshop(instID, "Sneaky Workshop")
	s.Forbidden(err)
}

// Only heads and staff change a workshop, their own participants' included.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotUpdateWorkshop() {
	_, participant, _, wsID := s.setupWorkshopWithParticipant("uw")

	_, err := participant.UpdateWorkshop(wsID, testutil.WorkshopUpdate{Name: "Hacked Workshop", Active: true})
	s.Forbidden(err)
}

func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotDeleteOtherUser() {
	head, participant, _, _ := s.setupWorkshopWithParticipant("du")

	s.Forbidden(participant.DeleteUser(head.ID))
}

// A participant reads only the workshop they belong to.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotAccessOtherWorkshop() {
	admin := s.DevUser()
	_, participant, _, _ := s.setupWorkshopWithParticipant("aw")

	inst2 := Must(admin.CreateInstitution("Other Org"))
	head2 := s.CreateUser("other-head")
	head2Invite := Must(admin.InviteToInstitution(inst2.Id.String(), "head", head2.ID))
	Must(head2.AcceptInvite(head2Invite.Id.String()))
	otherWorkshop := Must(head2.CreateWorkshop(inst2.Id.String(), "Other Workshop"))

	_, err := participant.GetWorkshop(otherWorkshop.Id.String())
	s.Forbidden(err)
}

// Not even into the institution of their own workshop.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotInviteToInstitution() {
	_, participant, instID, _ := s.setupWorkshopWithParticipant("ii")

	outsider := s.CreateUser("outsider")
	_, err := participant.InviteToInstitution(instID, "staff", outsider.ID)
	s.Forbidden(err)
}

func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotDeleteWorkshop() {
	_, participant, _, wsID := s.setupWorkshopWithParticipant("dw")

	s.Forbidden(participant.DeleteWorkshop(wsID))
}

// Only heads and staff create a workshop's invite link.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotCreateWorkshopInvite() {
	_, participant, _, wsID := s.setupWorkshopWithParticipant("wi")

	_, err := participant.CreateWorkshopInvite(wsID)
	s.Forbidden(err)
}

func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotListUsers() {
	_, participant, _, _ := s.setupWorkshopWithParticipant("lu")

	_, err := participant.GetUsers()
	s.Forbidden(err)
}

func (s *ParticipantRestrictionsTestSuite) TestParticipantCannotReadOtherUserProfile() {
	_, participant, _, _ := s.setupWorkshopWithParticipant("ru")

	outsider := s.CreateUser("ru-outsider")
	_, err := participant.GetUser(outsider.ID)
	s.Forbidden(err)
}

// A participant does read their own account.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCanSeeOwnProfile() {
	_, participant, _, _ := s.setupWorkshopWithParticipant("op")

	me := Must(participant.GetMe())
	s.Equal(participant.Name, me.Name)
	s.Equal("participant", participant.GetRole())
}

// A participant lists games and sees the ones they made in their workshop.
func (s *ParticipantRestrictionsTestSuite) TestParticipantCanSeeWorkshopGames() {
	_, participant, _, _ := s.setupWorkshopWithParticipant("wg")

	game := Must(participant.UploadGame("alien-first-contact"))

	games := Must(participant.ListGames())
	found := false
	for _, g := range games {
		if g.Id == game.Id {
			found = true
			break
		}
	}
	s.True(found, "participant should see their own workshop game")
}
