package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// UserAccessTestSuite tests role-based access control for user endpoints:
// GET /api/users and GET /api/users/{id}.
type UserAccessTestSuite struct {
	testutil.BaseSuite
}

func TestUserAccessSuite(t *testing.T) {
	s := &UserAccessTestSuite{}
	s.SuiteName = "User Access Tests"
	suite.Run(t, s)
}

// TestAdminCanListAllUsers verifies that an admin can see all users.
func (s *UserAccessTestSuite) TestAdminCanListAllUsers() {
	admin := s.DevUser()

	// Create a few users
	s.CreateUser("ua-alice")
	s.CreateUser("ua-bob")

	users := Must(admin.GetUsers())
	s.GreaterOrEqual(len(users), 3, "admin should see at least 3 users (dev + alice + bob)")
	s.T().Logf("Admin sees %d users", len(users))
}

// TestHeadCanListOrgMembers verifies that a head sees only their
// institution's members, not all users.
func (s *UserAccessTestSuite) TestHeadCanListOrgMembers() {
	admin := s.DevUser()

	// Create institution with head + staff
	inst := Must(admin.CreateInstitution("Head List Org"))
	head := s.CreateUser("ua-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	staff := s.CreateUser("ua-staff")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Create an outsider not in the institution
	s.CreateUser("ua-outsider")

	// Head lists users — should see org members only
	users := Must(head.GetUsers())
	s.T().Logf("Head sees %d users", len(users))

	// Should see at least head + staff (2 members)
	s.GreaterOrEqual(len(users), 2, "head should see at least 2 org members")

	// Should NOT see the outsider
	for _, u := range users {
		s.NotEqual("ua-outsider", u.Name, "head should not see users outside their institution")
	}
}

// TestStaffCanListOrgMembers verifies that a staff member sees only their
// institution's members.
func (s *UserAccessTestSuite) TestStaffCanListOrgMembers() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Staff List Org"))
	head := s.CreateUser("ua-head2")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	staff := s.CreateUser("ua-staff2")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Staff lists users — should see org members
	users := Must(staff.GetUsers())
	s.T().Logf("Staff sees %d users", len(users))
	s.GreaterOrEqual(len(users), 2, "staff should see at least 2 org members")
}

// TestIndividualCannotListUsers verifies that an individual user
// (no institution) cannot list users.
func (s *UserAccessTestSuite) TestIndividualCannotListUsers() {
	individual := s.CreateUser("ua-individual")

	_, err := individual.GetUsers()
	s.Error(err, "individual should not be able to list users")
	s.T().Logf("Correctly denied: %v", err)
}

// TestHeadCanReadOrgMemberProfile verifies that a head can read
// a profile of a member in their institution.
func (s *UserAccessTestSuite) TestHeadCanReadOrgMemberProfile() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Head Read Org"))
	head := s.CreateUser("ua-head3")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	staff := s.CreateUser("ua-staff3")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Head reads staff profile — should succeed
	var profile obj.User
	err := head.Get("users/"+staff.ID, &profile)
	s.NoError(err, "head should be able to read org member's profile")
	s.Equal(staff.Name, profile.Name)
	s.T().Logf("Head read staff profile: %s", profile.Name)
}

// TestStaffCanReadOrgMemberProfile verifies that a staff member can read
// a profile of another member in their institution.
func (s *UserAccessTestSuite) TestStaffCanReadOrgMemberProfile() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Staff Read Org"))
	head := s.CreateUser("ua-head4")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	staff := s.CreateUser("ua-staff4")
	staffInvite := Must(head.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Staff reads head profile — should succeed
	var profile obj.User
	err := staff.Get("users/"+head.ID, &profile)
	s.NoError(err, "staff should be able to read org member's profile")
	s.Equal(head.Name, profile.Name)
	s.T().Logf("Staff read head profile: %s", profile.Name)
}

// TestIndividualCannotReadOtherProfile verifies that an individual user
// cannot read another user's profile.
func (s *UserAccessTestSuite) TestIndividualCannotReadOtherProfile() {
	individual := s.CreateUser("ua-ind-read")
	other := s.CreateUser("ua-other-read")

	var profile obj.User
	err := individual.Get("users/"+other.ID, &profile)
	s.Error(err, "individual should not be able to read another user's profile")
	s.T().Logf("Correctly denied: %v", err)
}

// TestUserCanReadOwnProfile verifies that any user can read their own profile
// via GET /api/users/{id}.
func (s *UserAccessTestSuite) TestUserCanReadOwnProfile() {
	user := s.CreateUser("ua-self-read")

	var profile obj.User
	err := user.Get("users/"+user.ID, &profile)
	s.NoError(err, "user should be able to read own profile via /users/{id}")
	s.Equal(user.Name, profile.Name)
	s.T().Logf("User read own profile: %s", profile.Name)
}

// TestHeadCannotReadUserOutsideOrg verifies that a head cannot read
// a user profile outside their institution.
func (s *UserAccessTestSuite) TestHeadCannotReadUserOutsideOrg() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Head No Read Org"))
	head := s.CreateUser("ua-head5")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	outsider := s.CreateUser("ua-outsider2")

	var profile obj.User
	err := head.Get("users/"+outsider.ID, &profile)
	s.Error(err, "head should not be able to read a user outside their institution")
	s.T().Logf("Correctly denied: %v", err)
}
