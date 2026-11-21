package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// UserAccessTestSuite tests who may list accounts and read an account's
// profile.
type UserAccessTestSuite struct {
	testutil.BaseSuite
}

func TestUserAccessSuite(t *testing.T) {
	s := &UserAccessTestSuite{}
	s.SuiteName = "User Access Tests"
	suite.Run(t, s)
}

// institutionWithHeadAndStaff creates an institution with a head and a staff
// member. Returns (head, staff).
func (s *UserAccessTestSuite) institutionWithHeadAndStaff(name, headName, staffName string) (*testutil.UserClient, *testutil.UserClient) {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution(name))
	head := s.CreateUser(headName)
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	staff := s.CreateUser(staffName)
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))
	return head, staff
}

// An admin sees every account.
func (s *UserAccessTestSuite) TestAdminCanListAllUsers() {
	admin := s.DevUser()
	s.CreateUser("ua-alice")
	s.CreateUser("ua-bob")

	users := Must(admin.GetUsers())
	s.GreaterOrEqual(len(users), 3, "admin should see at least 3 users (dev + alice + bob)")
}

// A head sees their institution's members only.
func (s *UserAccessTestSuite) TestHeadCanListOrgMembers() {
	head, _ := s.institutionWithHeadAndStaff("Head List Org", "ua-head", "ua-staff")
	s.CreateUser("ua-outsider")

	users := Must(head.GetUsers())
	s.GreaterOrEqual(len(users), 2, "head should see at least 2 org members")
	for _, u := range users {
		s.NotEqual("ua-outsider", u.Name, "head should not see users outside their institution")
	}
}

// A staff member sees their institution's members.
func (s *UserAccessTestSuite) TestStaffCanListOrgMembers() {
	_, staff := s.institutionWithHeadAndStaff("Staff List Org", "ua-head2", "ua-staff2")

	users := Must(staff.GetUsers())
	s.GreaterOrEqual(len(users), 2, "staff should see at least 2 org members")
}

// An individual, without an institution, lists no accounts.
func (s *UserAccessTestSuite) TestIndividualCannotListUsers() {
	individual := s.CreateUser("ua-individual")

	users := Must(individual.GetUsers())
	s.Empty(users, "a list shows only what the caller may list: nothing for an individual")
}

func (s *UserAccessTestSuite) TestHeadCanReadOrgMemberProfile() {
	head, staff := s.institutionWithHeadAndStaff("Head Read Org", "ua-head3", "ua-staff3")

	profile := Must(head.GetUser(staff.ID))
	s.Equal(staff.Name, profile.Name)
}

func (s *UserAccessTestSuite) TestStaffCanReadOrgMemberProfile() {
	head, staff := s.institutionWithHeadAndStaff("Staff Read Org", "ua-head4", "ua-staff4")

	profile := Must(staff.GetUser(head.ID))
	s.Equal(head.Name, profile.Name)
}

func (s *UserAccessTestSuite) TestIndividualCannotReadOtherProfile() {
	individual := s.CreateUser("ua-ind-read")
	other := s.CreateUser("ua-other-read")

	_, err := individual.GetUser(other.ID)
	s.NotFound(err, "a user the caller cannot see does not exist for them")
}

// Anyone reads their own profile by its id.
func (s *UserAccessTestSuite) TestUserCanReadOwnProfile() {
	user := s.CreateUser("ua-self-read")

	profile := Must(user.GetUser(user.ID))
	s.Equal(user.Name, profile.Name)
}

func (s *UserAccessTestSuite) TestHeadCannotReadUserOutsideOrg() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Head No Read Org"))
	head := s.CreateUser("ua-head5")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	outsider := s.CreateUser("ua-outsider2")

	_, err := head.GetUser(outsider.ID)
	s.NotFound(err, "a user outside the head's organisation does not exist for them")
}
