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
	// Step 1: Check initial user count (should be 1: dev user)
	initialUsers := Must(s.DevUser().GetUsers())
	s.Equal(1, len(initialUsers), "should have 1 user (dev user)")
	s.T().Logf("Initial users: %d", len(initialUsers))

	// Crate Users
	clientAdmin := s.CreateUser("admin")
	s.Role(clientAdmin, string(obj.RoleAdmin))
	s.Equal("admin", clientAdmin.Name)
	s.Equal("admin@test.local", clientAdmin.Email)

	clientAlice := s.CreateUser("alice")
	s.Equal("alice", clientAlice.Name)
	s.Equal("alice@test.local", clientAlice.Email)

	clientHarry := s.CreateUser("harry")
	s.Equal("harry", clientHarry.Name)
	s.Equal("harry@test.local", clientHarry.Email)

	clientTanja := s.CreateUser("tanja")
	s.Equal("tanja", clientTanja.Name)
	s.Equal("tanja@test.local", clientTanja.Email)

	// Admin creates institutions
	institution1 := Must(clientAdmin.CreateInstitution("Test Institution 1"))
	s.NotEmpty(institution1.ID)
	s.Equal("Test Institution 1", institution1.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution1.Name, institution1.ID)

	institution2 := Must(clientAdmin.CreateInstitution("Test Institution 2"))
	s.NotEmpty(institution2.ID)
	s.Equal("Test Institution 2", institution2.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution2.Name, institution2.ID)

	// Step 6: Admin invites harry to institution as head
	harryInvite := Must(clientAdmin.InviteToInstitution(
		institution1.ID.String(),
		string(obj.RoleHead),
		clientHarry.ID,
	))
	s.NotEmpty(harryInvite.ID)
	s.Equal(obj.InviteStatusPending, harryInvite.Status)
	s.T().Logf("Created invite for harry: %s", harryInvite.ID)

	// admin can see the invite
	adminInvites := Must(clientAdmin.GetInvites())
	s.Equal(1, len(adminInvites), "admin should see 1 pending invite")
	s.Equal(harryInvite.ID, adminInvites[0].ID)
	s.T().Logf("Admin sees %d pending invite(s)", len(adminInvites))

	// Harry sees invitation
	harryInvites := Must(clientHarry.GetInvites())
	s.Equal(1, len(harryInvites), "harry should see 1 pending invite")
	s.Equal(harryInvite.ID, harryInvites[0].ID)
	s.T().Logf("Harry sees %d pending invite(s)", len(harryInvites))

	// Verify harry can't see the institutions
	Fail(clientHarry.GetInstitutions())
	Fail(clientHarry.GetInstitution(institution1.ID.String()))
	Fail(clientHarry.GetInstitution(institution2.ID.String()))

	// Harry accepts invitation
	acceptedHarryInvite := Must(clientHarry.AcceptInvite(harryInvite.ID.String()))
	s.Equal(obj.InviteStatusAccepted, acceptedHarryInvite.Status)
	s.T().Logf("Harry accepted invite")

	// Verify harry can now see only his institution
	Must(clientHarry.GetInstitutions())
	Must(clientHarry.GetInstitution(institution1.ID.String()))
	Fail(clientHarry.GetInstitution(institution2.ID.String()))

	// Verify harry's role in the institution
	harryUser := Must(clientHarry.GetMe())
	s.NotNil(harryUser.Role)
	s.Equal(obj.RoleHead, harryUser.Role.Role)
	s.NotNil(harryUser.Role.Institution)
	s.Equal(institution1.ID, harryUser.Role.Institution.ID)

	// Verify that admin can see that harry accepted the invite
	adminInvites = Must(clientAdmin.GetInvites())
	s.Equal(1, len(adminInvites), "admin should still see 1 invite")
	s.Equal(harryInvite.ID, adminInvites[0].ID)
	s.Equal(obj.InviteStatusAccepted, adminInvites[0].Status, "invite should be accepted")
	s.T().Logf("Admin sees invite with status: %s", adminInvites[0].Status)

	// Harry invites tanja to institution as staff
	tanjaInvite := Must(clientHarry.InviteToInstitution(
		institution1.ID.String(),
		string(obj.RoleStaff),
		clientTanja.ID,
	))
	s.NotEmpty(tanjaInvite.ID)
	s.Equal(obj.InviteStatusPending, tanjaInvite.Status)
	s.T().Logf("Harry created invite for tanja: %s", tanjaInvite.ID)

	// Tanja sees invitation
	tanjaInvites := Must(clientTanja.GetInvites())
	s.Equal(1, len(tanjaInvites), "tanja should see 1 pending invite")
	s.Equal(tanjaInvite.ID, tanjaInvites[0].ID)
	s.T().Logf("Tanja sees %d pending invite(s)", len(tanjaInvites))

	// Verify: Tanja can't see institution yet
	Fail(clientTanja.GetInstitutions())
	Fail(clientTanja.GetInstitution(institution1.ID.String()))
	Fail(clientTanja.GetInstitution(institution2.ID.String()))

	// Tanja accepts invitation
	acceptedTanjaInvite := Must(clientTanja.AcceptInvite(tanjaInvite.ID.String()))
	s.Equal(obj.InviteStatusAccepted, acceptedTanjaInvite.Status)
	s.T().Logf("Tanja accepted invite")

	// Verify tanja can now see institution
	Must(clientTanja.GetInstitutions())
	Must(clientTanja.GetInstitution(institution1.ID.String()))
	Fail(clientTanja.GetInstitution(institution2.ID.String()))

	// Verify tanja's role in the institution
	tanjaUser := Must(clientTanja.GetMe())
	s.NotNil(tanjaUser.Role)
	s.Equal(obj.RoleStaff, tanjaUser.Role.Role)
	s.NotNil(tanjaUser.Role.Institution)
	s.Equal(institution1.ID, tanjaUser.Role.Institution.ID)

	// Verify institution members
	finalInstitution := Must(clientAdmin.GetInstitution(institution1.ID.String()))
	s.Equal(2, len(finalInstitution.Members)) // harry (head) + tanja (staff)
	memberNames := make([]string, len(finalInstitution.Members))
	for i, member := range finalInstitution.Members {
		memberNames[i] = member.Name
	}
	s.Contains(memberNames, "harry")
	s.Contains(memberNames, "tanja")
	s.T().Logf("Institution members: %v", memberNames)

	// Verify tanja can't invite alice to institution
	Fail(clientTanja.InviteToInstitution(institution1.ID.String(), string(obj.RoleStaff), clientAlice.ID))

	// Harray creates invitiation to alice, revokes it, alice can't accept it
	invitation := Must(clientHarry.InviteToInstitution(institution1.ID.String(), string(obj.RoleStaff), clientAlice.ID))
	MustSucceed(clientHarry.RevokeInvite(invitation.ID.String()))
	invitationRevoked := Must(clientHarry.GetInvite(invitation.ID.String()))
	s.Equal(obj.InviteStatusRevoked, invitationRevoked.Status)
	// alice can see that it's revoked..
	invitationRevoked = Must(clientAlice.GetInvite(invitation.ID.String()))
	s.Equal(obj.InviteStatusRevoked, invitationRevoked.Status)
	// ..and can't accept it any more
	Fail(clientAlice.AcceptInvite(invitation.ID.String()))

	// Harray creates invitiation to alice, she declines it, he sees that, she can't accept it after declining
	invitation = Must(clientHarry.InviteToInstitution(institution1.ID.String(), string(obj.RoleStaff), clientAlice.ID))
	Must(clientAlice.DeclineInvite(invitation.ID.String()))
	invitationDeclined := Must(clientHarry.GetInvite(invitation.ID.String()))
	s.Equal(obj.InviteStatusDeclined, invitationDeclined.Status)
	// harry can see that it's declined
	invitationDeclined = Must(clientAlice.GetInvite(invitation.ID.String()))
	s.Equal(obj.InviteStatusDeclined, invitationDeclined.Status)
	// he can't revoke it any more
	MustFail(clientHarry.RevokeInvite(invitation.ID.String()))
	// alice can't accept it any more
	Fail(clientAlice.AcceptInvite(invitation.ID.String()))

}
