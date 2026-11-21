package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GuestRestrictionsTestSuite tests that anonymous workshop guests
// (share-link users) are refused on endpoints meant for regular users, heads
// or staff.
type GuestRestrictionsTestSuite struct {
	testutil.BaseSuite
}

func TestGuestRestrictionsSuite(t *testing.T) {
	s := &GuestRestrictionsTestSuite{}
	s.SuiteName = "Guest Restrictions Tests"
	suite.Run(t, s)
}

// setupWorkshopWithGuest creates an institution with a head, a
// workshop, and a guest who joined it by its invite link. prefix keeps
// names unique per test.
// Returns (head, guest, institution ID, workshop ID).
func (s *GuestRestrictionsTestSuite) setupWorkshopWithGuest(prefix string) (*testutil.UserClient, *testutil.UserClient, string, string) {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), prefix+" Workshop"))
	invite := Must(head.CreateWorkshopInvite(workshop.Id.String()))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))

	return head, guest, inst.Id.String(), workshop.Id.String()
}

// Only admins, heads and staff list organisations: a guest's list is empty.
func (s *GuestRestrictionsTestSuite) TestGuestCannotListInstitutions() {
	_, guest, _, _ := s.setupWorkshopWithGuest("li")

	s.Empty(Must(guest.ListInstitutions()))
}

func (s *GuestRestrictionsTestSuite) TestGuestCannotCreateInstitution() {
	_, guest, _, _ := s.setupWorkshopWithGuest("ci")

	_, err := guest.CreateInstitution("Sneaky Org")
	s.Forbidden(err)
}

// Only heads and staff list workshops, all of them or their institution's.
func (s *GuestRestrictionsTestSuite) TestGuestCannotListWorkshops() {
	_, guest, instID, _ := s.setupWorkshopWithGuest("lw")

	_, err := guest.ListAllWorkshops()
	s.Forbidden(err)
	_, err = guest.ListWorkshops(instID)
	s.Forbidden(err)
}

// Not even in the institution of their own workshop.
func (s *GuestRestrictionsTestSuite) TestGuestCannotCreateWorkshop() {
	_, guest, instID, _ := s.setupWorkshopWithGuest("cw")

	_, err := guest.CreateWorkshop(instID, "Sneaky Workshop")
	s.Forbidden(err)
}

// Only heads and staff change a workshop, their own guests' included.
func (s *GuestRestrictionsTestSuite) TestGuestCannotUpdateWorkshop() {
	_, guest, _, wsID := s.setupWorkshopWithGuest("uw")

	_, err := guest.UpdateWorkshop(wsID, testutil.WorkshopUpdate{Name: "Hacked Workshop", Active: true})
	s.Forbidden(err)
}

func (s *GuestRestrictionsTestSuite) TestGuestCannotDeleteOtherUser() {
	head, guest, _, _ := s.setupWorkshopWithGuest("du")

	s.NotFound(guest.DeleteUser(head.ID), "a user the guest cannot see does not exist for them")
}

// A guest reads only the workshop they belong to.
func (s *GuestRestrictionsTestSuite) TestGuestCannotAccessOtherWorkshop() {
	admin := s.DevUser()
	_, guest, _, _ := s.setupWorkshopWithGuest("aw")

	inst2 := Must(admin.CreateInstitution("Other Org"))
	head2 := s.CreateUser("other-head")
	head2Invite := Must(admin.InviteToInstitution(inst2.Id.String(), "head", head2.ID))
	Must(head2.AcceptInvite(head2Invite.Id.String()))
	otherWorkshop := Must(head2.CreateWorkshop(inst2.Id.String(), "Other Workshop"))

	_, err := guest.GetWorkshop(otherWorkshop.Id.String())
	s.Forbidden(err)
}

// Not even into the institution of their own workshop.
func (s *GuestRestrictionsTestSuite) TestGuestCannotInviteToInstitution() {
	_, guest, instID, _ := s.setupWorkshopWithGuest("ii")

	outsider := s.CreateUser("outsider")
	_, err := guest.InviteToInstitution(instID, "staff", outsider.ID)
	s.Forbidden(err)
}

func (s *GuestRestrictionsTestSuite) TestGuestCannotDeleteWorkshop() {
	_, guest, _, wsID := s.setupWorkshopWithGuest("dw")

	s.Forbidden(guest.DeleteWorkshop(wsID))
}

// Only heads and staff create a workshop's invite link.
func (s *GuestRestrictionsTestSuite) TestGuestCannotCreateWorkshopInvite() {
	_, guest, _, wsID := s.setupWorkshopWithGuest("wi")

	_, err := guest.CreateWorkshopInvite(wsID)
	s.Forbidden(err)
}

func (s *GuestRestrictionsTestSuite) TestGuestCannotListUsers() {
	_, guest, _, _ := s.setupWorkshopWithGuest("lu")

	users := Must(guest.GetUsers())
	s.Empty(users, "a list shows only what the caller may list: nothing for a guest")
}

func (s *GuestRestrictionsTestSuite) TestGuestCannotReadOtherUserProfile() {
	_, guest, _, _ := s.setupWorkshopWithGuest("ru")

	outsider := s.CreateUser("ru-outsider")
	_, err := guest.GetUser(outsider.ID)
	s.NotFound(err, "a user the guest cannot see does not exist for them")
}

// A guest does read their own account.
func (s *GuestRestrictionsTestSuite) TestGuestCanSeeOwnProfile() {
	_, guest, _, _ := s.setupWorkshopWithGuest("op")

	me := Must(guest.GetMe())
	s.Equal(guest.Name, me.Name)
	s.Equal("guest", guest.GetRole())
}

// A guest lists games and sees the ones they made in their workshop.
func (s *GuestRestrictionsTestSuite) TestGuestCanSeeWorkshopGames() {
	_, guest, _, _ := s.setupWorkshopWithGuest("wg")

	game := Must(guest.UploadGame("alien-first-contact"))

	games := Must(guest.ListGames())
	found := false
	for _, g := range games {
		if g.Id == game.Id {
			found = true
			break
		}
	}
	s.True(found, "guest should see their own workshop game")
}
