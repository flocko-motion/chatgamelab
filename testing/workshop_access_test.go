package testing

import (
	"fmt"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// WorkshopAccessTestSuite checks who sees and changes a workshop, as the
// secure core decides it (core/workshop*.go).
type WorkshopAccessTestSuite struct {
	testutil.BaseSuite
	n int
}

func TestWorkshopAccessSuite(t *testing.T) {
	s := &WorkshopAccessTestSuite{}
	s.SuiteName = "Workshop access"
	suite.Run(t, s)
}

// setup is an organisation at work with a head and staff, an individual
// (in no organisation), and a workshop the head created with two guests in
// it.
func (s *WorkshopAccessTestSuite) setup() (orgID string, head, staff, individual *testutil.UserClient, workshopID string, guests [2]*testutil.UserClient) {
	s.n++
	org := Must(s.DevUser().CreateInstitution(fmt.Sprintf("Werkstatt-Org %d", s.n)))
	orgID = org.Id.String()
	member := func(role string) *testutil.UserClient {
		s.n++
		user := s.CreateUser(fmt.Sprintf("%s%d", role, s.n))
		invite := Must(s.DevUser().InviteToInstitution(orgID, role, user.ID))
		Must(user.AcceptInviteByToken(invite.Id.String()))
		return user
	}
	head, staff = member("head"), member("staff")
	s.n++
	individual = s.CreateUser(fmt.Sprintf("individual%d", s.n))
	workshop := Must(head.CreateWorkshop(orgID, fmt.Sprintf("Werkstatt %d", s.n)))
	workshopID = workshop.Id.String()
	link := *Must(head.CreateWorkshopInvite(workshopID)).InviteToken
	guests[0] = Must(s.JoinWorkshop(link))
	guests[1] = Must(s.JoinWorkshop(link))
	return
}

// An organisation is cooperative: staff change and delete any of its
// workshops, not only their own.
func (s *WorkshopAccessTestSuite) TestStaffRunEveryWorkshop() {
	_, _, staff, individual, workshopID, _ := s.setup()

	renamed := Must(staff.UpdateWorkshop(workshopID, testutil.WorkshopUpdate{Name: fmt.Sprintf("Von Staff %d", s.n), Active: true}))
	s.True(renamed.MayDelete)

	_, err := individual.UpdateWorkshop(workshopID, testutil.WorkshopUpdate{Name: "Nein", Active: true})
	s.NotFound(err, "an individual not in it does not see it")

	s.NoError(staff.DeleteWorkshop(workshopID))
}

// The guests of a workshop see each other, but not its settings or invites.
func (s *WorkshopAccessTestSuite) TestGuestsSeeEachOther() {
	_, head, _, _, workshopID, guests := s.setup()

	w := Must(guests[0].GetWorkshop(workshopID))
	s.True(w.HasParticipants)
	ids := map[string]bool{}
	for _, p := range w.Participants {
		ids[p.Id.String()] = true
	}
	s.True(ids[guests[1].ID], "a guest sees the other guest")
	s.False(w.HasSettings, "no settings")
	s.Empty(w.Invites, "no invites")
	s.False(w.MayEditCard)

	own := Must(head.GetWorkshop(workshopID))
	s.True(own.HasSettings)
	s.NotEmpty(own.Invites, "the head sees the invite link")
}

// A workshop with a public page shows its card to any account, nothing
// more; a private one does not exist for outsiders.
func (s *WorkshopAccessTestSuite) TestAPublicWorkshopShowsItsCard() {
	_, head, _, _, workshopID, _ := s.setup()
	outsider := s.CreateUser(fmt.Sprintf("aussen%d", s.n))

	_, err := outsider.GetWorkshop(workshopID)
	s.NotFound(err, "private")

	Must(head.UpdateWorkshop(workshopID, testutil.WorkshopUpdate{Name: fmt.Sprintf("Öffentlich %d", s.n), Active: true, Public: true}))
	w := Must(outsider.GetWorkshop(workshopID))
	s.True(w.Public)
	s.False(w.HasSettings)
	s.False(w.HasParticipants)
	s.Empty(w.Participants)
}

// Staff give a guest a new login code, which ends the old one; the guest
// themselves does not.
func (s *WorkshopAccessTestSuite) TestStaffResetAGuestsCode() {
	_, _, staff, _, _, guests := s.setup()

	old := Must(staff.GetGuestToken(guests[0].ID))
	fresh := Must(staff.ResetGuestToken(guests[0].ID))
	s.NotEqual(old, fresh)

	_, err := guests[1].ResetGuestToken(guests[0].ID)
	s.Error(err, "a guest resets nobody's code")
}
