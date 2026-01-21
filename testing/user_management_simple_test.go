package testing

import (
	"cgl/api/routes"
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
	clientAdmin.MustPost("institutions", routes.CreateInstitutionRequest{
		Name: "Test Institution",
	}, &institution)
	s.NotEmpty(institution.ID)
	s.Equal("Test Institution", institution.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution.Name, institution.ID)

	// Step 6: Admin invites harry to institution as head
	harryUserID := clientHarry.ID
	var harryInvite obj.UserRoleInvite
	clientAdmin.MustPost("invites/institution", routes.CreateInstitutionInviteRequest{
		InstitutionID: institution.ID.String(),
		Role:          string(obj.RoleHead),
		InvitedUserID: &harryUserID,
	}, &harryInvite)
	s.NotEmpty(harryInvite.ID)
	s.Equal(obj.InviteStatusPending, harryInvite.Status)
	s.T().Logf("Created invite for harry: %s", harryInvite.ID)

	// admin can see the invite
	var adminInvites []obj.UserRoleInvite
	clientAdmin.MustGet("invites", &adminInvites)
	s.Equal(1, len(adminInvites), "admin should see 1 pending invite")
	s.Equal(harryInvite.ID, adminInvites[0].ID)
	s.T().Logf("Admin sees %d pending invite(s)", len(adminInvites))

	// Step 7: Harry sees invitation
	var harryInvites []obj.UserRoleInvite
	clientHarry.MustGet("invites", &harryInvites)
	s.Equal(1, len(harryInvites), "harry should see 1 pending invite")
	s.Equal(harryInvite.ID, harryInvites[0].ID)
	s.T().Logf("Harry sees %d pending invite(s)", len(harryInvites))

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

	// Verify that admin can see that harry accepted the invite
	clientAdmin.MustGet("invites", &adminInvites)
	s.Equal(1, len(adminInvites), "admin should still see 1 invite")
	s.Equal(harryInvite.ID, adminInvites[0].ID)
	s.Equal(obj.InviteStatusAccepted, adminInvites[0].Status, "invite should be accepted")
	s.T().Logf("Admin sees invite with status: %s", adminInvites[0].Status)

	// Step 10: Harry invites tanja to institution as staff
	tanjaUserID := clientTanja.ID
	var tanjaInvite obj.UserRoleInvite
	clientHarry.MustPost("invites/institution", routes.CreateInstitutionInviteRequest{
		InstitutionID: institution.ID.String(),
		Role:          string(obj.RoleStaff),
		InvitedUserID: &tanjaUserID,
	}, &tanjaInvite)
	s.NotEmpty(tanjaInvite.ID)
	s.Equal(obj.InviteStatusPending, tanjaInvite.Status)
	s.T().Logf("Harry created invite for tanja: %s", tanjaInvite.ID)

	// Step 11: Tanja sees invitation
	var tanjaInvites []obj.UserRoleInvite
	clientTanja.MustGet("invites", &tanjaInvites)
	s.Equal(1, len(tanjaInvites), "tanja should see 1 pending invite")
	s.Equal(tanjaInvite.ID, tanjaInvites[0].ID)
	s.T().Logf("Tanja sees %d pending invite(s)", len(tanjaInvites))

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
