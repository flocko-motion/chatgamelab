package testing

import (
	"fmt"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// OrganisationAccessTestSuite checks who sees and changes an organisation
// and its members, as the secure core decides it (core/organisation*.go,
// core/membership.go).
type OrganisationAccessTestSuite struct {
	testutil.BaseSuite
	n int
}

func TestOrganisationAccessSuite(t *testing.T) {
	s := &OrganisationAccessTestSuite{}
	s.SuiteName = "Organisation access"
	suite.Run(t, s)
}

// organisation is a new organisation at work, with a head.
func (s *OrganisationAccessTestSuite) organisation() (id string, head *testutil.UserClient) {
	s.n++
	inst := Must(s.DevUser().CreateInstitution(fmt.Sprintf("Zugriff %d", s.n)))
	return inst.Id.String(), s.member(inst.Id.String(), "head")
}

// member is a new account holding role in organisation id.
func (s *OrganisationAccessTestSuite) member(id, role string) *testutil.UserClient {
	s.n++
	user := s.CreateUser(fmt.Sprintf("%s%d", role, s.n))
	invite := Must(s.DevUser().InviteToInstitution(id, role, user.ID))
	Must(user.AcceptInviteByToken(invite.Id.String()))
	return user
}

// guest is a guest in a new workshop of organisation id.
func (s *OrganisationAccessTestSuite) guest(id string, head *testutil.UserClient) *testutil.UserClient {
	workshop := Must(head.CreateWorkshop(id, fmt.Sprintf("Werkstatt %d", s.n)))
	return Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshop.Id.String())).InviteToken))
}

// An organisation exists only for its members, the guests in its workshops
// and the admin; for anyone else it is not found, and it is not listed.
func (s *OrganisationAccessTestSuite) TestOnlyItsPeopleSeeAnOrganisation() {
	id, head := s.organisation()
	_, otherHead := s.organisation()

	for name, outsider := range map[string]*testutil.UserClient{
		"an individual":             s.CreateUser(fmt.Sprintf("aussen%d", s.n)),
		"another organisation head": otherHead,
	} {
		_, err := outsider.GetInstitution(id)
		s.NotFound(err, name)
		_, err = outsider.GetInstitutionMembers(id)
		s.NotFound(err, name)
	}
	for _, org := range Must(otherHead.ListInstitutions()) {
		s.NotEqual(id, org.Id.String(), "a head lists only their own organisation")
	}

	s.Equal(head.ID, Must(head.GetInstitution(id)).Members[0].Id.String())
	s.True(Must(s.DevUser().GetInstitution(id)).HasRecord, "the admin sees its record")
}

// A guest sees the card of their workshop's organisation, nothing more, and
// not its members.
func (s *OrganisationAccessTestSuite) TestAGuestSeesOnlyTheCard() {
	id, head := s.organisation()
	guest := s.guest(id, head)

	org := Must(guest.GetInstitution(id))
	s.NotEmpty(org.Name)
	s.False(org.HasSettings, "no settings")
	s.False(org.HasDetails, "no details")
	s.False(org.HasRecord, "no record")
	s.Empty(org.Members, "no members")
	s.Empty(Must(guest.GetInstitutionMembers(id)))
}

// Head and staff are the members (spec, Roles) and list each other with
// their emails. Guests take part in its workshops and are not members.
func (s *OrganisationAccessTestSuite) TestMembersListTheMembers() {
	id, head := s.organisation()
	staff := s.member(id, "staff")
	s.guest(id, head)

	for name, member := range map[string]*testutil.UserClient{"head": head, "staff": staff} {
		members := Must(member.GetInstitutionMembers(id))
		s.Len(members, 2, name)
		for _, m := range members {
			s.True(m.HasContact, "%s sees how to reach %s", name, m.Name)
		}
	}
}

// Head and staff rename the organisation and set its restriction; a guest
// of its workshops does not, and only a head consents.
func (s *OrganisationAccessTestSuite) TestHeadAndStaffChangeTheOrganisation() {
	id, head := s.organisation()
	staff := s.member(id, "staff")
	guest := s.guest(id, head)

	renamed := Must(staff.RenameInstitution(id, fmt.Sprintf("Umbenannt %d", s.n)))
	s.True(renamed.MayEditSettings)
	s.False(renamed.MayEditDetails, "staff do not consent")
	set := Must(head.SetInstitutionPromptConstraints(id, "Keine Gewalt."))
	s.Equal("Keine Gewalt.", *set.PromptConstraints)
	cleared := Must(staff.SetInstitutionPromptConstraints(id, ""))
	s.Nil(cleared.PromptConstraints, "an empty restriction removes it")

	_, err := guest.RenameInstitution(id, "Nein")
	s.Forbidden(err, "a guest of its workshops")
	_, err = guest.SetInstitutionPromptConstraints(id, "Nein")
	s.Forbidden(err, "a guest of its workshops")

	_, otherHead := s.organisation()
	_, err = otherHead.RenameInstitution(id, "Nein")
	s.NotFound(err, "another head")
}

// A head removes members and guests, staff remove guests only, a member
// leaves, becoming an individual again; the last head stays. Nobody removes
// someone of another organisation: for them, that user is not found there.
func (s *OrganisationAccessTestSuite) TestRemovingMembers() {
	id, head := s.organisation()
	staff := s.member(id, "staff")
	staff2 := s.member(id, "staff")
	leaver := s.member(id, "staff")
	guest := s.guest(id, head)

	s.Forbidden(staff.RemoveMember(id, staff2.ID), "staff remove guests only")
	s.NoError(staff.RemoveMember(id, guest.ID))
	s.NoError(leaver.RemoveMember(id, leaver.ID))
	s.Equal("individual", testutil.RoleOf(Must(leaver.GetMe())), "who leaves is an individual again")
	s.NoError(head.RemoveMember(id, staff2.ID))
	s.Refused(head.RemoveMember(id, head.ID), testutil.CodeLastHead, "the last head stays")

	otherID, otherHead := s.organisation()
	s.NotFound(head.RemoveMember(id, otherHead.ID), "not a member here")
	s.NotFound(head.RemoveMember(otherID, otherHead.ID), "another organisation")

	members := Must(head.GetInstitutionMembers(id))
	s.Len(members, 2, "head and staff are left")
}

// Only an admin creates and deletes organisations.
func (s *OrganisationAccessTestSuite) TestOnlyAnAdminCreatesAndDeletes() {
	id, head := s.organisation()

	_, err := head.CreateInstitution("Eigene")
	s.Forbidden(err, "a head creates none")
	s.Forbidden(head.DeleteInstitution(id), "a head deletes none")

	s.NoError(s.DevUser().DeleteInstitution(id))
	_, err = head.GetInstitution(id)
	s.NotFound(err, "gone")
}
