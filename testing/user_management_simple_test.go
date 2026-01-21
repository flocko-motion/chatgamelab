package testing

import (
	"cgl/obj"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// MultiUserTestSuite contains multi-user collaboration tests
// Each suite gets its own fresh Docker environment
type MultiUserTestSuite struct {
	testutil.BaseSuite
}

// TestMultiUserSuite runs the multi-user test suite
func TestMultiUserSuite(t *testing.T) {
	s := &MultiUserTestSuite{}
	s.SuiteName = "Multi-User Tests"
	suite.Run(t, s)
}

// TestUserManagement tests institution and invite workflow
// - create admin
// - create user alice
// - create user harry
// - create user tanja
// - admin creates institution
// - admin invites harry to institution as head
// - harry sees invitation and accepts it
// - harry can now see institution
// - harry invites tanja to institution as staff
// - tanja sees invitation and accepts it
// - tanja can now see institution
func (s *MultiUserTestSuite) TestUserManagement() {
	// List users - should have 1 (default dev user)
	var initialUsers []obj.User
	s.DevUser().MustGet("users", &initialUsers)
	s.Equal(1, len(initialUsers), "should have 1 user (dev user)")
	s.T().Logf("Initial users: %d", len(initialUsers))

	// Step 1: Create admin user
	clientAdmin := s.CreateUser("admin")
	s.Role(clientAdmin, string(obj.RoleAdmin))
	s.Equal("admin", clientAdmin.Name)
	s.Equal("admin@test.local", clientAdmin.Email)

	// Step 2: Create user alice
	clientAlice := s.CreateUser("alice")
	s.Equal("alice", clientAlice.Name)
	s.Equal("alice@test.local", clientAlice.Email)

	// Step 3: Create user harry
	clientHarry := s.CreateUser("harry")
	s.Equal("harry", clientHarry.Name)
	s.Equal("harry@test.local", clientHarry.Email)

	// Step 4: Create user tanja
	clientTanja := s.CreateUser("tanja")
	s.Equal("tanja", clientTanja.Name)
	s.Equal("tanja@test.local", clientTanja.Email)

	// Step 5: Admin creates institution
	var institution obj.Institution
	clientAdmin.MustPost("institutions", map[string]string{
		"name": "Test Institution",
	}, &institution)
	s.NotEmpty(institution.ID)
	s.Equal("Test Institution", institution.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution.Name, institution.ID)

	// Step 6: Admin invites harry to institution as head
	var harryInvite obj.UserRoleInvite
	clientAdmin.MustPost("invites/institution", map[string]interface{}{
		"institutionId": institution.ID.String(),
		"role":          string(obj.RoleHead),
		"invitedUserId": clientHarry.ID,
	}, &harryInvite)
	s.NotEmpty(harryInvite.ID)
	s.Equal(obj.InviteStatusPending, harryInvite.Status)
	s.T().Logf("Created invite for harry: %s", harryInvite.ID)

	// Step 7: Harry sees invitation
	// TODO: Add endpoint GET /users/me/invites to list pending invites for current user
	// var harryInvites []obj.UserRoleInvite
	// clientHarry.MustGet("users/me/invites", &harryInvites)
	// s.Contains(harryInvites, harryInvite)

	// Step 8: Harry accepts invitation
	var acceptedInvite obj.UserRoleInvite
	clientHarry.MustPost("invites/institution/"+harryInvite.ID.String()+"/accept", nil, &acceptedInvite)
	s.Equal(obj.InviteStatusAccepted, acceptedInvite.Status)
	s.T().Logf("Harry accepted invite")

	// Step 9: Harry can now see institution
	var harryInstitutions []obj.Institution
	clientHarry.MustGet("institutions", &harryInstitutions)
	s.Equal(1, len(harryInstitutions))
	s.Equal(institution.ID, harryInstitutions[0].ID)
	s.T().Logf("Harry can see institution: %s", harryInstitutions[0].Name)

	// Verify harry's role in the institution
	var harryUser obj.User
	clientHarry.MustGet("users/me", &harryUser)
	s.NotNil(harryUser.Role)
	s.Equal(obj.RoleHead, harryUser.Role.Role)
	s.NotNil(harryUser.Role.Institution)
	s.Equal(institution.ID, harryUser.Role.Institution.ID)

	// Step 10: Harry invites tanja to institution as staff
	var tanjaInvite obj.UserRoleInvite
	clientHarry.MustPost("invites/institution", map[string]interface{}{
		"institutionId": institution.ID.String(),
		"role":          string(obj.RoleStaff),
		"invitedUserId": clientTanja.ID,
	}, &tanjaInvite)
	s.NotEmpty(tanjaInvite.ID)
	s.Equal(obj.InviteStatusPending, tanjaInvite.Status)
	s.T().Logf("Harry created invite for tanja: %s", tanjaInvite.ID)

	// Step 11: Tanja sees invitation
	// TODO: Add endpoint GET /users/me/invites to list pending invites for current user
	// var tanjaInvites []obj.UserRoleInvite
	// clientTanja.MustGet("users/me/invites", &tanjaInvites)
	// s.Contains(tanjaInvites, tanjaInvite)

	// Step 12: Tanja accepts invitation
	var tanjaAcceptedInvite obj.UserRoleInvite
	clientTanja.MustPost("invites/institution/"+tanjaInvite.ID.String()+"/accept", nil, &tanjaAcceptedInvite)
	s.Equal(obj.InviteStatusAccepted, tanjaAcceptedInvite.Status)
	s.T().Logf("Tanja accepted invite")

	// Step 13: Tanja can now see institution
	var tanjaInstitutions []obj.Institution
	clientTanja.MustGet("institutions", &tanjaInstitutions)
	s.Equal(1, len(tanjaInstitutions))
	s.Equal(institution.ID, tanjaInstitutions[0].ID)
	s.T().Logf("Tanja can see institution: %s", tanjaInstitutions[0].Name)

	// Verify tanja's role in the institution
	var tanjaUser obj.User
	clientTanja.MustGet("users/me", &tanjaUser)
	s.NotNil(tanjaUser.Role)
	s.Equal(obj.RoleStaff, tanjaUser.Role.Role)
	s.NotNil(tanjaUser.Role.Institution)
	s.Equal(institution.ID, tanjaUser.Role.Institution.ID)

	// Final verification: Check institution members
	var finalInstitution obj.Institution
	clientAdmin.MustGet("institutions/"+institution.ID.String(), &finalInstitution)
	s.Equal(2, len(finalInstitution.Members)) // harry (head) + tanja (staff)

	memberNames := make([]string, len(finalInstitution.Members))
	for i, member := range finalInstitution.Members {
		memberNames[i] = member.Name
	}
	s.Contains(memberNames, "harry")
	s.Contains(memberNames, "tanja")
	s.T().Logf("Institution members: %v", memberNames)
}
