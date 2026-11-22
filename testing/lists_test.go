package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// ListsTestSuite: every list endpoint filters, searches, sorts and pages on
// the server and refuses what it does not know; what the caller may do with
// an item comes with it.
type ListsTestSuite struct {
	testutil.BaseSuite
}

func TestListsSuite(t *testing.T) {
	s := &ListsTestSuite{}
	s.SuiteName = "Lists"
	suite.Run(t, s)
}

// organisation is an organisation with a head and two staff members.
func (s *ListsTestSuite) organisation(name string) (inst testutil.Institution, head, anna, bert *testutil.UserClient) {
	admin := s.DevUser()
	inst = Must(admin.CreateInstitution(name))
	head = s.CreateUser(name + "-head")
	anna = s.CreateUser(name + "-anna")
	bert = s.CreateUser(name + "-bert")
	Must(admin.SetUserRoleIn(head.ID, "head", inst.Id.String()))
	Must(admin.SetUserRoleIn(anna.ID, "staff", inst.Id.String()))
	Must(admin.SetUserRoleIn(bert.ID, "staff", inst.Id.String()))
	return inst, head, anna, bert
}

func (s *ListsTestSuite) TestMemberListPagesSortsAndSearchesOnTheServer() {
	inst, head, _, _ := s.organisation("pages")
	id := inst.Id.String()

	page := Must(head.ListMembersPage(id, testutil.ListQuery{SortBy: "name", SortDir: "asc", Limit: 2}))
	s.Equal(3, page.Total, "all members count")
	s.Len(page.Items, 2, "one page")
	s.Equal("pages-anna", page.Items[0].Name)
	s.Equal("pages-bert", page.Items[1].Name)

	rest := Must(head.ListMembersPage(id, testutil.ListQuery{SortBy: "name", SortDir: "asc", Limit: 2, Offset: 2}))
	s.Len(rest.Items, 1)
	s.Equal("pages-head", rest.Items[0].Name)

	found := Must(head.ListMembersPage(id, testutil.ListQuery{Search: "BERT"}))
	s.Equal(1, found.Total, "search is case-insensitive and partial")

	byName := Must(head.ListMembersPage(id, testutil.ListQuery{SortBy: "name", SortDir: "desc"}))
	s.Equal("pages-head", byName.Items[0].Name, "the order turns")
}

func (s *ListsTestSuite) TestAListRefusesWhatItDoesNotKnow() {
	inst, head, _, _ := s.organisation("refuses")
	id := inst.Id.String()

	_, err := head.ListMembersPage(id, testutil.ListQuery{SortBy: "shoeSize"})
	s.Invalid(err, "an unknown order")
	_, err = head.ListMembersPage(id, testutil.ListQuery{Filter: []string{"tall"}})
	s.Invalid(err, "a filter the list does not offer")
	_, err = head.ListMembersPage(id, testutil.ListQuery{SortDir: "sideways"})
	s.Invalid(err, "an unknown direction")
}

func (s *ListsTestSuite) TestMembersComeWithWhatTheCallerMayDoWithThem() {
	inst, head, anna, bert := s.organisation("flags")
	id := inst.Id.String()

	of := func(caller *testutil.UserClient, member *testutil.UserClient) testutil.User {
		for _, m := range Must(caller.ListMembersPage(id, testutil.ListQuery{})).Items {
			if m.Id.String() == member.ID {
				return m
			}
		}
		s.FailNow("member not listed", member.Name)
		return testutil.User{}
	}

	// a head changes their members' roles and removes them
	s.True(of(head, anna).MayChangeRole)
	s.True(of(head, anna).MayRemoveMember)
	s.False(of(head, anna).MayChangeAnyRole, "only the admin gives any role")

	// staff neither change roles nor remove other staff
	s.False(of(anna, bert).MayChangeRole)
	s.False(of(anna, bert).MayRemoveMember)
	// but leave themselves
	s.True(of(anna, anna).MayRemoveMember, "a member leaves")

	// the admin gives any role
	s.True(of(s.DevUser(), anna).MayChangeAnyRole)

	// and what the flags say is what the server does
	_, err := anna.SetUserRoleIn(bert.ID, "head", id)
	s.Forbidden(err, "staff make no one head")
	s.NoError(head.RemoveMember(id, bert.ID), "the head removes staff")
}

func (s *ListsTestSuite) TestTheOrganisationSaysWhatTheCallerMayDoInIt() {
	inst, head, anna, _ := s.organisation("invites")
	id := inst.Id.String()

	asHead := Must(head.GetInstitution(id))
	s.True(asHead.MayInviteHead)
	s.True(asHead.MayInviteStaff)
	s.True(asHead.MayCreateWorkshop)
	s.True(asHead.MayListApiKeys)

	asStaff := Must(anna.GetInstitution(id))
	s.False(asStaff.MayInviteHead, "staff invite no heads")
	s.True(asStaff.MayInviteStaff)
	s.True(asStaff.MayCreateWorkshop)
	s.True(asStaff.MayListApiKeys)

}

func (s *ListsTestSuite) TestParticipantsAreAListOfTheirOwn() {
	inst, head, _, _ := s.organisation("participants")
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Pages"))
	invite := Must(head.CreateWorkshopInvite(workshop.Id.String()))
	for range 3 {
		Must(s.JoinWorkshop(*invite.InviteToken))
	}

	page := Must(head.ListParticipantsPage(workshop.Id.String(), testutil.ListQuery{Limit: 2}))
	s.Equal(4, page.Total, "three guests and the workshop's creator")
	s.Len(page.Items, 2)

	_, err := head.ListParticipantsPage(workshop.Id.String(), testutil.ListQuery{SortBy: "age"})
	s.Invalid(err)
}

// An individual is in no organisation, also while they work in one of its
// workshops: no head makes them head or staff without an invite, and they
// are no member to remove.
func (s *ListsTestSuite) TestAnIndividualInAWorkshopIsNoMember() {
	inst, head, _, _ := s.organisation("visit")
	id := inst.Id.String()
	workshop := Must(head.CreateWorkshop(id, "Visited"))
	invite := Must(head.CreateWorkshopInvite(workshop.Id.String()))

	visitor := s.CreateUser("visit-individual")
	s.Equal("individual", visitor.GetRole())
	s.NoError(visitor.AcceptWorkshopInviteByToken(*invite.InviteToken), "the individual takes part")

	_, err := head.SetUserRoleIn(visitor.ID, "staff", id)
	s.Error(err, "a head makes a visiting individual no staff")
	_, err = head.SetUserRoleIn(visitor.ID, "head", id)
	s.Error(err, "nor head")
	s.Error(head.RemoveMember(id, visitor.ID), "a visiting individual is no member to remove")
	s.Equal("individual", Must(s.DevUser().GetUser(visitor.ID)).Role.Role, "still an individual")

	for _, m := range Must(head.ListMembersPage(id, testutil.ListQuery{})).Items {
		s.NotEqual(visitor.ID, m.Id.String(), "and no member")
	}
}
