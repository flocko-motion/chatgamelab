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

// TestUserManagement tests the complete user and institution management workflow:
// creating users with different roles, inviting members to institutions,
// accepting/declining invites, and managing institution membership with
// proper permission enforcement
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

	clientBob := s.CreateUser("bob")
	s.Equal("bob", clientBob.Name)
	s.Equal("bob@test.local", clientBob.Email)

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

	// Admin invites bob to institution 2 - bob accepts (we do this so that institution 2 has >0 members)
	bobInvite := Must(clientAdmin.InviteToInstitution(
		institution2.ID.String(),
		string(obj.RoleStaff),
		clientBob.ID,
	))
	s.NotEmpty(bobInvite.ID)
	s.Equal(obj.InviteStatusPending, bobInvite.Status)
	s.T().Logf("Created invite for bob: %s", bobInvite.ID)
	bobAcceptInvite := Must(clientBob.AcceptInvite(bobInvite.ID.String()))
	s.Equal(obj.InviteStatusAccepted, bobAcceptInvite.Status)
	s.T().Logf("Bob accepted invite")

	// Admin invites harry to institution as head
	harryInvite := Must(clientAdmin.InviteToInstitution(
		institution1.ID.String(),
		string(obj.RoleHead),
		clientHarry.ID,
	))
	s.NotEmpty(harryInvite.ID)
	s.Equal(obj.InviteStatusPending, harryInvite.Status)
	s.T().Logf("Created invite for harry: %s", harryInvite.ID)

	// admin can see the invites (bob's accepted + harry's pending)
	adminInvites := Must(clientAdmin.GetInvites())
	s.Equal(2, len(adminInvites), "admin should see 2 invites (bob's accepted + harry's pending)")
	s.T().Logf("Admin sees %d invite(s)", len(adminInvites))

	// Harry sees invitation
	harryInvites := Must(clientHarry.GetInvites())
	s.Equal(1, len(harryInvites), "harry should see 1 pending invite")
	s.Equal(harryInvite.ID, harryInvites[0].ID)
	s.T().Logf("Harry sees %d pending invite(s)", len(harryInvites))

	// Verify harry can't list institutions (not a member yet)
	Fail(clientHarry.GetInstitutions())
	// ..but can view individual institutions by ID (public data, no members shown)
	inst1BeforeAccept := Must(clientHarry.GetInstitution(institution1.ID.String()))
	s.Equal(0, len(inst1BeforeAccept.Members), "harry should not see members before joining")
	inst2BeforeAccept := Must(clientHarry.GetInstitution(institution2.ID.String()))
	s.Equal(0, len(inst2BeforeAccept.Members), "harry should not see members of institution2 (not authorized)")

	// Harry accepts invitation
	acceptedHarryInvite := Must(clientHarry.AcceptInvite(harryInvite.ID.String()))
	s.Equal(obj.InviteStatusAccepted, acceptedHarryInvite.Status)
	s.T().Logf("Harry accepted invite")

	// Verify harry can now see only his institution in the list
	institutions := Must(clientHarry.GetInstitutions())
	s.Equal(1, len(institutions), "harry should see 1 institution")
	s.Equal(institution1.ID, institutions[0].ID)
	s.T().Logf("Harry sees %d institution(s)", len(institutions))
	// ..and can see institution 1 by ID - with members
	inst1AfterAccept := Must(clientHarry.GetInstitution(institution1.ID.String()))
	s.Equal(1, len(inst1AfterAccept.Members), "harry should see 1 member (himself) after accepting")
	// ..and can view institution 2 public data but not members (not a member)
	inst2AfterAccept := Must(clientHarry.GetInstitution(institution2.ID.String()))
	s.Equal(0, len(inst2AfterAccept.Members), "harry should not see members of institution2 (not a member)")

	// Verify harry's role in the institution
	harryUser := Must(clientHarry.GetMe())
	s.NotNil(harryUser.Role)
	s.Equal(obj.RoleHead, harryUser.Role.Role)
	s.NotNil(harryUser.Role.Institution)
	s.Equal(institution1.ID, harryUser.Role.Institution.ID)

	// Verify that admin can see that harry accepted the invite
	adminInvites = Must(clientAdmin.GetInvites())
	s.Equal(2, len(adminInvites), "admin should see 2 invites (bob's and harry's)")
	s.T().Logf("Admin sees %d invite(s)", len(adminInvites))

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

	// Verify: Tanja can't list institutions yet (not a member)
	Fail(clientTanja.GetInstitutions())
	// ..but can view institution public data (no members shown)
	tanjaInst1 := Must(clientTanja.GetInstitution(institution1.ID.String()))
	s.Equal(0, len(tanjaInst1.Members), "tanja should not see members before joining")
	tanjaInst2 := Must(clientTanja.GetInstitution(institution2.ID.String()))
	s.Equal(0, len(tanjaInst2.Members), "tanja should not see members of institution2")

	// Tanja accepts invitation
	acceptedTanjaInvite := Must(clientTanja.AcceptInvite(tanjaInvite.ID.String()))
	s.Equal(obj.InviteStatusAccepted, acceptedTanjaInvite.Status)
	s.T().Logf("Tanja accepted invite")

	// Verify tanja can now see her institution in the list
	tanjaInstitutions := Must(clientTanja.GetInstitutions())
	s.Equal(1, len(tanjaInstitutions), "tanja should see 1 institution")
	// ..and can view institution1 with members
	tanjaInst1After := Must(clientTanja.GetInstitution(institution1.ID.String()))
	s.Equal(2, len(tanjaInst1After.Members), "tanja should see 2 members (harry and herself)")
	// ..and can view institution2 public data but not members
	tanjaInst2After := Must(clientTanja.GetInstitution(institution2.ID.String()))
	s.Equal(0, len(tanjaInst2After.Members), "tanja should not see members of institution2")

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

	// Harry creates invitiation to alice, revokes it, alice can't accept it
	invitation := Must(clientHarry.InviteToInstitution(institution1.ID.String(), string(obj.RoleStaff), clientAlice.ID))
	MustSucceed(clientHarry.RevokeInvite(invitation.ID.String()))
	invitationRevoked := Must(clientHarry.GetInvite(invitation.ID.String()))
	s.Equal(obj.InviteStatusRevoked, invitationRevoked.Status)
	// alice can see that it's revoked..
	invitationRevoked = Must(clientAlice.GetInvite(invitation.ID.String()))
	s.Equal(obj.InviteStatusRevoked, invitationRevoked.Status)
	// ..and can't accept it any more
	Fail(clientAlice.AcceptInvite(invitation.ID.String()))

	// Harry creates invitiation to alice, she declines it, he sees that, she can't accept it after declining
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

	// Harry invites alice once more, she accepts
	aliceInvitation := Must(clientHarry.InviteToInstitution(institution1.ID.String(), string(obj.RoleStaff), clientAlice.ID))
	Must(clientAlice.AcceptInvite(aliceInvitation.ID.String()))

	// Now harry, alice, tanja should all be able to list the institution members
	harryMembers := Must(clientHarry.GetInstitution(institution1.ID.String()))
	s.Equal(3, len(harryMembers.Members)) // harry (head) + tanja (staff) + alice (staff)

	aliceMembers := Must(clientAlice.GetInstitution(institution1.ID.String()))
	s.Equal(3, len(aliceMembers.Members))

	tanjaMembers := Must(clientTanja.GetInstitution(institution1.ID.String()))
	s.Equal(3, len(tanjaMembers.Members))

	// Alice can't remove tanja from the institution (only heads can remove members)
	MustFail(clientAlice.RemoveMember(institution1.ID.String(), clientTanja.ID))

	// Harry can remove tanja and does
	MustSucceed(clientHarry.RemoveMember(institution1.ID.String(), clientTanja.ID))

	// Verify tanja was removed
	finalMembers := Must(clientHarry.GetInstitution(institution1.ID.String()))
	s.Equal(2, len(finalMembers.Members)) // harry (head) + alice (staff)
	memberNames = make([]string, len(finalMembers.Members))
	for i, member := range finalMembers.Members {
		memberNames[i] = member.Name
	}
	s.Contains(memberNames, "harry")
	s.Contains(memberNames, "alice")
	s.NotContains(memberNames, "tanja")
	s.T().Logf("Institution now has 2 members: %v", memberNames)

}
