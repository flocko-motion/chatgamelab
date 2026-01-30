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
	clientAdmin *testutil.UserClient
}

// SetupSuite runs once when the suite starts to initialize common resources
func (s *MultiUserTestSuite) SetupSuite() {
	// Call parent SetupSuite to initialize base suite
	s.BaseSuite.SetupSuite()

	// Verify initial state (should have 5 dev users)
	initialUsers := Must(s.DevUser().GetUsers())
	s.Equal(5, len(initialUsers), "should have 5 users (preseeded dev users)")
	s.T().Logf("Initial users: %d", len(initialUsers))

	// Use preseeded admin user
	s.clientAdmin = s.DevUser()
}

// TestMultiUserSuite runs the multi-user test suite
func TestMultiUserSuite(t *testing.T) {
	s := &MultiUserTestSuite{}
	s.SuiteName = "Multi-User Tests"
	suite.Run(t, s)
}

// TestInstitutionManagement tests the complete user and institution management workflow:
// creating users with different roles, inviting members to institutions,
// accepting/declining invites, and managing institution membership with
// proper permission enforcement
func (s *MultiUserTestSuite) TestInstitutionManagement() {
	// Create test users
	clientAlice := s.CreateUser("alice")
	s.Equal("alice", clientAlice.Name)
	s.Equal("alice@test.local", clientAlice.Email)
	s.Equal("individual", clientAlice.GetRole())

	clientBob := s.CreateUser("bob")
	s.Equal("bob", clientBob.Name)
	s.Equal("bob@test.local", clientBob.Email)
	s.Equal("individual", clientBob.GetRole())

	clientHarry := s.CreateUser("harry")
	s.Equal("harry", clientHarry.Name)
	s.Equal("harry@test.local", clientHarry.Email)
	s.Equal("individual", clientHarry.GetRole())

	clientTanja := s.CreateUser("tanja")
	s.Equal("tanja", clientTanja.Name)
	s.Equal("tanja@test.local", clientTanja.Email)
	s.Equal("individual", clientTanja.GetRole())

	// Admin creates institutions
	institution1 := Must(s.clientAdmin.CreateInstitution("Test Institution 1"))
	s.NotEmpty(institution1.ID)
	s.Equal("Test Institution 1", institution1.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution1.Name, institution1.ID)

	institution2 := Must(s.clientAdmin.CreateInstitution("Test Institution 2"))
	s.NotEmpty(institution2.ID)
	s.Equal("Test Institution 2", institution2.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution2.Name, institution2.ID)

	// Admin invites bob to institution 2 - bob accepts (we do this so that institution 2 has >0 members)
	bobInvite := Must(s.clientAdmin.InviteToInstitution(
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
	harryInvite := Must(s.clientAdmin.InviteToInstitution(
		institution1.ID.String(),
		string(obj.RoleHead),
		clientHarry.ID,
	))
	s.NotEmpty(harryInvite.ID)
	s.Equal(obj.InviteStatusPending, harryInvite.Status)
	s.T().Logf("Created invite for harry: %s", harryInvite.ID)

	// admin can see the invites (bob's accepted + harry's pending)
	adminInvites := Must(s.clientAdmin.GetInvitesOutgoing(institution1.ID.String()))
	s.Equal(1, len(adminInvites), "admin should see 1 invite (harry's pending)")
	s.T().Logf("Admin sees %d invite(s)", len(adminInvites))
	adminInvites = Must(s.clientAdmin.GetInvitesOutgoing(institution2.ID.String()))
	s.Equal(1, len(adminInvites), "admin should see 1 invite (bob's accepted)")
	s.T().Logf("Admin sees %d invite(s)", len(adminInvites))

	// Harry sees invitation
	harryInvites := Must(clientHarry.GetInvitesIncoming())
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
	adminInvites = Must(s.clientAdmin.GetInvitesOutgoing(institution1.ID.String()))
	s.Equal(1, len(adminInvites), "admin should see 1 invite (harry's)")
	s.T().Logf("Admin sees %d invite(s)", len(adminInvites))
	s.Equal(obj.InviteStatusAccepted, adminInvites[0].Status)

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
	tanjaInvites := Must(clientTanja.GetInvitesIncoming())
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
	finalInstitution := Must(s.clientAdmin.GetInstitution(institution1.ID.String()))
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

	// Harry creates invitiation to alice, revokes it (hard delete), invite no longer exists
	invitation := Must(clientHarry.InviteToInstitution(institution1.ID.String(), string(obj.RoleStaff), clientAlice.ID))
	MustSucceed(clientHarry.RevokeInvite(invitation.ID.String()))
	// Invite is hard-deleted, so fetching it should fail
	Fail(clientHarry.GetInvite(invitation.ID.String()))
	// Alice also can't see it
	Fail(clientAlice.GetInvite(invitation.ID.String()))
	// Alice can't accept it (it doesn't exist)
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

// TestInstitutionManagementLeadership a head can not remove himself from the institution
// he can though invite an existing staff to become head - once a second head exists,
// the head can remove the other head... then reinvite the removed head to become head again
// users can't remove themselves - they must ask another head to remove them
// this ensures that an institution is never without head
func (s *MultiUserTestSuite) TestInstitutionManagementLeadership() {
	// Create users with unique names
	clientCharlie := s.CreateUser("charlie")
	clientDiana := s.CreateUser("diana")

	// Admin creates institution
	institution := Must(s.clientAdmin.CreateInstitution("Leadership Test Institution"))
	s.T().Logf("Created institution: %s", institution.Name)

	// Admin invites Charlie as head
	charlieInvite := Must(s.clientAdmin.InviteToInstitution(
		institution.ID.String(),
		string(obj.RoleHead),
		clientCharlie.ID,
	))
	Must(clientCharlie.AcceptInvite(charlieInvite.ID.String()))
	s.T().Logf("Charlie became head")

	// Admin invites Diana as staff
	dianaInvite := Must(s.clientAdmin.InviteToInstitution(
		institution.ID.String(),
		string(obj.RoleStaff),
		clientDiana.ID,
	))
	Must(clientDiana.AcceptInvite(dianaInvite.ID.String()))
	s.T().Logf("Diana became staff")

	// Charlie (head) cannot remove himself - institution must have at least one head
	MustFail(clientCharlie.RemoveMember(institution.ID.String(), clientCharlie.ID))
	s.T().Logf("Charlie cannot remove himself (expected - must have at least one head)")

	// Charlie invites Diana to become head (promoting staff to head)
	dianaHeadInvite := Must(clientCharlie.InviteToInstitution(
		institution.ID.String(),
		string(obj.RoleHead),
		clientDiana.ID,
	))
	Must(clientDiana.AcceptInvite(dianaHeadInvite.ID.String()))
	s.T().Logf("Diana promoted to head (lost staff role due to single-role enforcement)")

	// Verify both are heads now (2 members: charlie head, diana head)
	// Note: Diana's old staff role was deleted when she accepted the head role
	instWithHeads := Must(clientCharlie.GetInstitution(institution.ID.String()))
	s.Equal(2, len(instWithHeads.Members), "should have 2 members")
	headCount := 0
	for _, member := range instWithHeads.Members {
		if member.Role == obj.RoleHead {
			headCount++
		}
	}
	s.Equal(2, headCount, "should have 2 heads (charlie and diana)")
	s.T().Logf("Institution now has 2 heads: charlie and diana")

	// Now Charlie can remove Diana (another head)
	MustSucceed(clientCharlie.RemoveMember(institution.ID.String(), clientDiana.ID))
	s.T().Logf("Charlie removed Diana (head)")

	// Verify Diana was removed (1 member remains: charlie)
	instAfterRemoval := Must(clientCharlie.GetInstitution(institution.ID.String()))
	s.Equal(1, len(instAfterRemoval.Members), "should have 1 member (charlie)")
	s.Equal("charlie", instAfterRemoval.Members[0].Name)
	s.Equal(obj.RoleHead, instAfterRemoval.Members[0].Role)
	s.T().Logf("Institution now has 1 member: charlie (head)")

	// Charlie re-invites Diana as head
	dianaHeadInvite2 := Must(clientCharlie.InviteToInstitution(
		institution.ID.String(),
		string(obj.RoleHead),
		clientDiana.ID,
	))
	Must(clientDiana.AcceptInvite(dianaHeadInvite2.ID.String()))
	s.T().Logf("Diana re-invited and became head again")

	// Verify both are heads again (2 members: charlie, diana)
	instWithBothHeads := Must(clientCharlie.GetInstitution(institution.ID.String()))
	s.Equal(2, len(instWithBothHeads.Members), "should have 2 members")
	s.T().Logf("Institution has 2 heads again")

	// Charlie cannot remove himself - the validation prevents self-removal
	MustFail(clientCharlie.RemoveMember(institution.ID.String(), clientCharlie.ID))
	s.T().Logf("Charlie cannot remove himself (expected - validation prevents self-removal)")

	// Diana also cannot remove herself
	MustFail(clientDiana.RemoveMember(institution.ID.String(), clientDiana.ID))
	s.T().Logf("Diana cannot remove herself (expected - validation prevents self-removal)")
}

// TestInstitutionManagementLeadershipSteal creates two institutions with one head each
// the head of institution 1 invites the head of institution 2 to become head of institution 1
// when accepting, the user loses their old role and institution 2 becomes headless
// users can only give up their had position, if this doesn't leave their old institution headless
func (s *MultiUserTestSuite) TestInstitutionManagementLeadershipSteal() {
	// Create users with unique names
	clientEve := s.CreateUser("eve")
	clientFrank := s.CreateUser("frank")

	// Admin creates two institutions
	institution1 := Must(s.clientAdmin.CreateInstitution("Institution Alpha"))
	s.T().Logf("Created institution1: %s", institution1.Name)

	institution2 := Must(s.clientAdmin.CreateInstitution("Institution Beta"))
	s.T().Logf("Created institution2: %s", institution2.Name)

	// Admin invites Eve as head of institution1
	eveInvite1 := Must(s.clientAdmin.InviteToInstitution(
		institution1.ID.String(),
		string(obj.RoleHead),
		clientEve.ID,
	))
	Must(clientEve.AcceptInvite(eveInvite1.ID.String()))
	s.T().Logf("Eve became head of institution1")

	// Admin invites Frank as head of institution2
	frankInvite2 := Must(s.clientAdmin.InviteToInstitution(
		institution2.ID.String(),
		string(obj.RoleHead),
		clientFrank.ID,
	))
	Must(clientFrank.AcceptInvite(frankInvite2.ID.String()))
	s.T().Logf("Frank became head of institution2")

	// Verify each institution has one head
	inst1Members := Must(clientEve.GetInstitution(institution1.ID.String()))
	s.Equal(1, len(inst1Members.Members), "institution1 should have 1 member")
	s.Equal("eve", inst1Members.Members[0].Name)

	inst2Members := Must(clientFrank.GetInstitution(institution2.ID.String()))
	s.Equal(1, len(inst2Members.Members), "institution2 should have 1 member")
	s.Equal("frank", inst2Members.Members[0].Name)

	// Frank cannot leave institution2 because he's the last head
	MustFail(clientFrank.RemoveMember(institution2.ID.String(), clientFrank.ID))
	s.T().Logf("Frank cannot leave institution2 (last head)")

	// Solution: First introduce Grace to become head of institution2
	// This way institution2 won't be headless when Frank leaves
	clientGrace := s.CreateUser("grace")
	graceInvite2 := Must(clientFrank.InviteToInstitution(
		institution2.ID.String(),
		string(obj.RoleHead),
		clientGrace.ID,
	))
	Must(clientGrace.AcceptInvite(graceInvite2.ID.String()))
	s.T().Logf("Grace became head of institution2")

	// Verify institution2 now has 2 heads
	inst2With2Heads := Must(clientFrank.GetInstitution(institution2.ID.String()))
	s.Equal(2, len(inst2With2Heads.Members), "institution2 should have 2 members")
	s.T().Logf("Institution2 now has 2 heads: frank and grace")

	// Now Eve invites Frank to become head of institution1
	frankInvite1 := Must(clientEve.InviteToInstitution(
		institution1.ID.String(),
		string(obj.RoleHead),
		clientFrank.ID,
	))
	s.T().Logf("Eve invited Frank to become head of institution1")

	// Frank can now accept the invite (institution2 won't be headless)
	Must(clientFrank.AcceptInvite(frankInvite1.ID.String()))
	s.T().Logf("Frank accepted head role in institution1 (lost role in institution2)")

	// Verify Frank is now head of institution1
	inst1WithFrank := Must(clientEve.GetInstitution(institution1.ID.String()))
	s.Equal(2, len(inst1WithFrank.Members), "institution1 should have 2 members")
	frankFound := false
	for _, member := range inst1WithFrank.Members {
		if member.Name == "frank" {
			frankFound = true
			s.Equal(obj.RoleHead, member.Role)
		}
	}
	s.True(frankFound, "frank should be a head in institution1")
	s.T().Logf("Institution1 now has 2 heads: eve and frank")

	// Institution2 now has only Grace (Frank left when accepting new role)
	inst2AfterFrankLeft := Must(clientGrace.GetInstitution(institution2.ID.String()))
	s.Equal(1, len(inst2AfterFrankLeft.Members), "institution2 should have 1 member")
	s.Equal("grace", inst2AfterFrankLeft.Members[0].Name)
	s.Equal(obj.RoleHead, inst2AfterFrankLeft.Members[0].Role)
	s.T().Logf("Institution2 now has 1 head: grace (Frank left)")

	// Frank cannot remove himself from institution1 (self-removal validation)
	MustFail(clientFrank.RemoveMember(institution1.ID.String(), clientFrank.ID))
	s.T().Logf("Frank cannot remove himself from institution1 (validation prevents self-removal)")

	// Eve can remove Frank from institution1 (since there are 2 heads)
	MustSucceed(clientEve.RemoveMember(institution1.ID.String(), clientFrank.ID))
	s.T().Logf("Eve removed Frank from institution1")

	// Verify Frank was removed from institution1
	inst1AfterRemoval := Must(clientEve.GetInstitution(institution1.ID.String()))
	s.Equal(1, len(inst1AfterRemoval.Members), "institution1 should have 1 member")
	s.Equal("eve", inst1AfterRemoval.Members[0].Name)
	s.T().Logf("Institution1 back to 1 head: eve")

	// Frank now has no role in any institution
	s.T().Logf("Frank has no role in any institution")
}
