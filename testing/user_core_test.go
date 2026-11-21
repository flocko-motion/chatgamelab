package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// UserCoreTestSuite tests the user through the secure core: sections by who
// may see them (card, account, contact, secrets), and what each may change.
type UserCoreTestSuite struct {
	testutil.BaseSuite
}

func TestUserCoreSuite(t *testing.T) {
	s := &UserCoreTestSuite{}
	s.SuiteName = "User Core Tests"
	suite.Run(t, s)
}

// workshopWithGuest is an organisation with a head and a staff member, one
// workshop, and a guest who joined it through its invite link.
func (s *UserCoreTestSuite) workshopWithGuest(tag string) (head, staff, guest *testutil.UserClient) {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("User Core Org " + tag))
	head = s.CreateUser("uc-head-" + tag)
	Must(head.AcceptInvite(Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID)).Id.String()))
	staff = s.CreateUser("uc-staff-" + tag)
	Must(staff.AcceptInvite(Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID)).Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "User Core Workshop "+tag))
	guest = Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshop.Id.String())).InviteToken))
	return head, staff, guest
}

// /me is a shortcut: the same view as the user read by their id.
func (s *UserCoreTestSuite) TestMeIsTheOwnUser() {
	alice := s.CreateUser("uc-me")
	me := Must(alice.GetMe())
	byID := Must(alice.GetUser(alice.ID))
	s.Equal(byID, me)
	s.True(me.HasAccount)
	s.True(me.HasContact)
	s.True(me.MayEditCard)
	s.False(me.MayDelete, "nobody deletes their own account")
}

// Everyone changes their own email, except a guest, who has none.
func (s *UserCoreTestSuite) TestOwnEmailButNotAGuests() {
	alice := s.CreateUser("uc-email")
	email := "uc-email-new@example.org"
	updated := Must(alice.UpdateUser(alice.ID, testutil.UserUpdate{Email: &email}))
	s.Require().NotNil(updated.Email)
	s.Equal(email, *updated.Email)

	_, _, guest := s.workshopWithGuest("email")
	me := Must(guest.GetMe())
	s.False(me.MayEditContact)
	guestEmail := "guest@example.org"
	_, err := guest.UpdateUser(guest.ID, testutil.UserUpdate{Email: &guestEmail})
	s.Forbidden(err)
}

// A guest's login code: the guest's own, and their organisation's heads and
// staff may read it, to help a guest who lost theirs; nobody else.
func (s *UserCoreTestSuite) TestWhoReadsAGuestsLoginCode() {
	head, staff, guest := s.workshopWithGuest("code")
	own := Must(guest.GetGuestCode())
	s.Require().NotNil(own.Token)

	s.Equal(own, Must(head.GetUserSecrets(guest.ID)))
	s.Equal(own, Must(staff.GetUserSecrets(guest.ID)))

	_, err := s.DevUser().GetUserSecrets(guest.ID)
	s.Forbidden(err, "the admin sees the guest, but does not log in as them")

	outsider := s.CreateUser("uc-code-outsider")
	_, err = outsider.GetUserSecrets(guest.ID)
	s.NotFound(err)
}

// Heads and staff change and delete the guests in their workshops, never
// their organisation's registered members.
func (s *UserCoreTestSuite) TestHeadsManageGuestsNotMembers() {
	head, staff, guest := s.workshopWithGuest("manage")

	view := Must(head.GetUser(guest.ID))
	s.True(view.MayEditCard)
	s.True(view.MayDelete)
	Must(head.UpdateUserName(guest.ID, "Renamed Guest"))

	member := Must(head.GetUser(staff.ID))
	s.False(member.MayEditCard)
	s.False(member.MayDelete)
	_, err := head.UpdateUserName(staff.ID, "Renamed Staff")
	s.Forbidden(err)
	s.Forbidden(head.DeleteUser(staff.ID))

	s.Require().NoError(head.DeleteUser(guest.ID))
	_, err = head.GetUser(guest.ID)
	s.NotFound(err, "a deleted guest is gone")
}

// Heads list their organisation's members, not the guests in its
// workshops, though they may read those one by one.
func (s *UserCoreTestSuite) TestHeadsListMembersNotGuests() {
	head, staff, guest := s.workshopWithGuest("list")
	var ids []string
	for _, u := range Must(head.GetUsers()) {
		ids = append(ids, u.Id.String())
	}
	s.Contains(ids, head.ID)
	s.Contains(ids, staff.ID)
	s.NotContains(ids, guest.ID)
	Must(head.GetUser(guest.ID))
}

// An organisation's last head is never deleted, not even by the admin.
func (s *UserCoreTestSuite) TestTheLastHeadStays() {
	head, _, _ := s.workshopWithGuest("lasthead")
	s.Refused(s.DevUser().DeleteUser(head.ID), testutil.CodeLastHead)
}
