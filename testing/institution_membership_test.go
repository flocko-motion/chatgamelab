package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// MultiUserTestSuite contains multi-user collaboration tests: institutions,
// their members, invites and heads.
type MultiUserTestSuite struct {
	testutil.BaseSuite
	clientAdmin *testutil.UserClient
}

// SetupSuite runs once when the suite starts to initialize common resources
func (s *MultiUserTestSuite) SetupSuite() {
	// Call parent SetupSuite to initialize base suite
	s.BaseSuite.SetupSuite()

	// Verify initial state (should have 9 dev users)
	initialUsers := Must(s.DevUser().GetUsers())
	s.Equal(9, len(initialUsers), "should have 9 users (preseeded dev users)")
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
	s.NotEmpty(institution1.Id)
	s.Equal("Test Institution 1", institution1.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution1.Name, institution1.Id)

	institution2 := Must(s.clientAdmin.CreateInstitution("Test Institution 2"))
	s.NotEmpty(institution2.Id)
	s.Equal("Test Institution 2", institution2.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution2.Name, institution2.Id)

	// Admin invites bob to institution 2 - bob accepts (we do this so that institution 2 has >0 members)
	bobInvite := Must(s.clientAdmin.InviteToInstitution(
		institution2.Id.String(),
		"staff",
		clientBob.ID,
	))
	s.NotEmpty(bobInvite.Id)
	s.Equal("pending", bobInvite.Status)
	s.T().Logf("Created invite for bob: %s", bobInvite.Id)
	bobAcceptInvite := Must(clientBob.AcceptInvite(bobInvite.Id.String()))
	s.Equal("accepted", bobAcceptInvite.Status)
	s.T().Logf("Bob accepted invite")

	// Admin invites harry to institution as head
	harryInvite := Must(s.clientAdmin.InviteToInstitution(
		institution1.Id.String(),
		"head",
		clientHarry.ID,
	))
	s.NotEmpty(harryInvite.Id)
	s.Equal("pending", harryInvite.Status)
	s.T().Logf("Created invite for harry: %s", harryInvite.Id)

	// admin can see the invites (bob's accepted + harry's pending)
	adminInvites := Must(s.clientAdmin.GetInvitesOutgoing(institution1.Id.String()))
	s.Equal(1, len(adminInvites), "admin should see 1 invite (harry's pending)")
	s.T().Logf("Admin sees %d invite(s)", len(adminInvites))
	adminInvites = Must(s.clientAdmin.GetInvitesOutgoing(institution2.Id.String()))
	s.Equal(1, len(adminInvites), "admin should see 1 invite (bob's accepted)")
	s.T().Logf("Admin sees %d invite(s)", len(adminInvites))

	// Harry sees invitation
	harryInvites := Must(clientHarry.GetInvitesIncoming())
	s.Equal(1, len(harryInvites), "harry should see 1 pending invite")
	s.Equal(harryInvite.Id.String(), harryInvites[0].Id)
	s.T().Logf("Harry sees %d pending invite(s)", len(harryInvites))

	// Verify harry can't list institutions (not a member yet)
	_, err := clientHarry.GetInstitutions()
	s.Forbidden(err)
	// ..but can view individual institutions by ID (public data, no members shown)
	inst1BeforeAccept := Must(clientHarry.GetInstitution(institution1.Id.String()))
	s.Equal(0, len(testutil.MembersOf(inst1BeforeAccept)), "harry should not see members before joining")
	inst2BeforeAccept := Must(clientHarry.GetInstitution(institution2.Id.String()))
	s.Equal(0, len(testutil.MembersOf(inst2BeforeAccept)), "harry should not see members of institution2 (not authorized)")

	// Harry accepts invitation
	acceptedHarryInvite := Must(clientHarry.AcceptInvite(harryInvite.Id.String()))
	s.Equal("accepted", acceptedHarryInvite.Status)
	s.T().Logf("Harry accepted invite")

	// Verify harry can now see only his institution in the list
	institutions := Must(clientHarry.GetInstitutions())
	s.Equal(1, len(institutions), "harry should see 1 institution")
	s.Equal(institution1.Id, institutions[0].Id)
	s.T().Logf("Harry sees %d institution(s)", len(institutions))
	// ..and can see institution 1 by ID - with members
	inst1AfterAccept := Must(clientHarry.GetInstitution(institution1.Id.String()))
	s.Equal(1, len(testutil.MembersOf(inst1AfterAccept)), "harry should see 1 member (himself) after accepting")
	// ..and can view institution 2 public data but not members (not a member)
	inst2AfterAccept := Must(clientHarry.GetInstitution(institution2.Id.String()))
	s.Equal(0, len(testutil.MembersOf(inst2AfterAccept)), "harry should not see members of institution2 (not a member)")

	// Verify harry's role in the institution
	harryUser := Must(clientHarry.GetMe())
	s.NotNil(harryUser.Role)
	s.Equal("head", harryUser.Role.Role)
	s.NotNil(harryUser.Role.Institution)
	s.Equal(institution1.Id, harryUser.Role.Institution.Id)

	// Verify that admin can see that harry accepted the invite
	adminInvites = Must(s.clientAdmin.GetInvitesOutgoing(institution1.Id.String()))
	s.Equal(1, len(adminInvites), "admin should see 1 invite (harry's)")
	s.T().Logf("Admin sees %d invite(s)", len(adminInvites))
	s.Equal("accepted", adminInvites[0].Status)

	// Harry invites tanja to institution as staff
	tanjaInvite := Must(clientHarry.InviteToInstitution(
		institution1.Id.String(),
		"staff",
		clientTanja.ID,
	))
	s.NotEmpty(tanjaInvite.Id)
	s.Equal("pending", tanjaInvite.Status)
	s.T().Logf("Harry created invite for tanja: %s", tanjaInvite.Id)

	// Tanja sees invitation
	tanjaInvites := Must(clientTanja.GetInvitesIncoming())
	s.Equal(1, len(tanjaInvites), "tanja should see 1 pending invite")
	s.Equal(tanjaInvite.Id.String(), tanjaInvites[0].Id)
	s.T().Logf("Tanja sees %d pending invite(s)", len(tanjaInvites))

	// Verify: Tanja can't list institutions yet (not a member)
	_, err = clientTanja.GetInstitutions()
	s.Forbidden(err)
	// ..but can view institution public data (no members shown)
	tanjaInst1 := Must(clientTanja.GetInstitution(institution1.Id.String()))
	s.Equal(0, len(testutil.MembersOf(tanjaInst1)), "tanja should not see members before joining")
	tanjaInst2 := Must(clientTanja.GetInstitution(institution2.Id.String()))
	s.Equal(0, len(testutil.MembersOf(tanjaInst2)), "tanja should not see members of institution2")

	// Tanja accepts invitation
	acceptedTanjaInvite := Must(clientTanja.AcceptInvite(tanjaInvite.Id.String()))
	s.Equal("accepted", acceptedTanjaInvite.Status)
	s.T().Logf("Tanja accepted invite")

	// Verify tanja can now see her institution in the list
	tanjaInstitutions := Must(clientTanja.GetInstitutions())
	s.Equal(1, len(tanjaInstitutions), "tanja should see 1 institution")
	// ..and can view institution1 with members
	tanjaInst1After := Must(clientTanja.GetInstitution(institution1.Id.String()))
	s.Equal(2, len(testutil.MembersOf(tanjaInst1After)), "tanja should see 2 members (harry and herself)")
	// ..and can view institution2 public data but not members
	tanjaInst2After := Must(clientTanja.GetInstitution(institution2.Id.String()))
	s.Equal(0, len(testutil.MembersOf(tanjaInst2After)), "tanja should not see members of institution2")

	// Verify tanja's role in the institution
	tanjaUser := Must(clientTanja.GetMe())
	s.NotNil(tanjaUser.Role)
	s.Equal("staff", tanjaUser.Role.Role)
	s.NotNil(tanjaUser.Role.Institution)
	s.Equal(institution1.Id, tanjaUser.Role.Institution.Id)

	// Verify institution members
	finalInstitution := Must(s.clientAdmin.GetInstitution(institution1.Id.String()))
	s.Equal(2, len(testutil.MembersOf(finalInstitution))) // harry (head) + tanja (staff)
	memberNames := make([]string, len(testutil.MembersOf(finalInstitution)))
	for i, member := range testutil.MembersOf(finalInstitution) {
		memberNames[i] = member.Name
	}
	s.Contains(memberNames, "harry")
	s.Contains(memberNames, "tanja")
	s.T().Logf("Institution members: %v", memberNames)

	// Verify tanja (staff) can also invite alice to institution
	tanjaInvite = Must(clientTanja.InviteToInstitution(institution1.Id.String(), "staff", clientAlice.ID))
	s.T().Logf("Tanja (staff) invited alice: %s", tanjaInvite.Id)
	// Revoke it so harry can invite alice next
	MustSucceed(clientTanja.RevokeInvite(tanjaInvite.Id.String()))

	// Harry creates invitiation to alice, revokes it (hard delete), invite no longer exists
	invitation := Must(clientHarry.InviteToInstitution(institution1.Id.String(), "staff", clientAlice.ID))
	MustSucceed(clientHarry.RevokeInvite(invitation.Id.String()))
	// Invite is hard-deleted, so it can't be found any more
	_, err = clientHarry.GetInvite(invitation.Id.String())
	s.NotFound(err)
	// Alice also can't see it
	_, err = clientAlice.GetInvite(invitation.Id.String())
	s.NotFound(err)
	// Alice can't accept it (it doesn't exist)
	_, err = clientAlice.AcceptInvite(invitation.Id.String())
	s.NotFound(err)

	// Harry creates invitiation to alice, she declines it, he sees that, she can't accept it after declining
	invitation = Must(clientHarry.InviteToInstitution(institution1.Id.String(), "staff", clientAlice.ID))
	Must(clientAlice.DeclineInvite(invitation.Id.String()))
	invitationDeclined := Must(clientHarry.GetInvite(invitation.Id.String()))
	s.Equal("declined", invitationDeclined.Status)
	// alice can see that it's declined too
	invitationDeclined = Must(clientAlice.GetInvite(invitation.Id.String()))
	s.Equal("declined", invitationDeclined.Status)
	// he can't revoke it any more: only a pending invite can be revoked
	s.Conflict(clientHarry.RevokeInvite(invitation.Id.String()))
	// alice can't accept it any more
	_, err = clientAlice.AcceptInvite(invitation.Id.String())
	s.Conflict(err)

	// Harry invites alice once more, she accepts
	aliceInvitation := Must(clientHarry.InviteToInstitution(institution1.Id.String(), "staff", clientAlice.ID))
	Must(clientAlice.AcceptInvite(aliceInvitation.Id.String()))

	// Now harry, alice, tanja should all be able to list the institution members
	harryMembers := Must(clientHarry.GetInstitution(institution1.Id.String()))
	s.Equal(3, len(testutil.MembersOf(harryMembers))) // harry (head) + tanja (staff) + alice (staff)

	aliceMembers := Must(clientAlice.GetInstitution(institution1.Id.String()))
	s.Equal(3, len(testutil.MembersOf(aliceMembers)))

	tanjaMembers := Must(clientTanja.GetInstitution(institution1.Id.String()))
	s.Equal(3, len(testutil.MembersOf(tanjaMembers)))

	// Alice can't remove tanja from the institution (only heads can remove members)
	s.Forbidden(clientAlice.RemoveMember(institution1.Id.String(), clientTanja.ID))

	// Harry can remove tanja and does
	MustSucceed(clientHarry.RemoveMember(institution1.Id.String(), clientTanja.ID))

	// Verify tanja was removed
	finalMembers := Must(clientHarry.GetInstitution(institution1.Id.String()))
	s.Equal(2, len(testutil.MembersOf(finalMembers))) // harry (head) + alice (staff)
	memberNames = make([]string, len(testutil.MembersOf(finalMembers)))
	for i, member := range testutil.MembersOf(finalMembers) {
		memberNames[i] = member.Name
	}
	s.Contains(memberNames, "harry")
	s.Contains(memberNames, "alice")
	s.NotContains(memberNames, "tanja")
	s.T().Logf("Institution now has 2 members: %v", memberNames)
}

// TestInstitutionManagementLeadership: the last head can't leave the institution.
// He can invite an existing staff to become head; once a second head exists, he
// can remove the other head, reinvite them, and then leave himself. This ensures
// that an institution is never without a head.
func (s *MultiUserTestSuite) TestInstitutionManagementLeadership() {
	// Create users with unique names
	clientCharlie := s.CreateUser("charlie")
	clientDiana := s.CreateUser("diana")

	// Admin creates institution
	institution := Must(s.clientAdmin.CreateInstitution("Leadership Test Institution"))
	s.T().Logf("Created institution: %s", institution.Name)

	// Admin invites Charlie as head
	charlieInvite := Must(s.clientAdmin.InviteToInstitution(
		institution.Id.String(),
		"head",
		clientCharlie.ID,
	))
	Must(clientCharlie.AcceptInvite(charlieInvite.Id.String()))
	s.T().Logf("Charlie became head")

	// Admin invites Diana as staff
	dianaInvite := Must(s.clientAdmin.InviteToInstitution(
		institution.Id.String(),
		"staff",
		clientDiana.ID,
	))
	Must(clientDiana.AcceptInvite(dianaInvite.Id.String()))
	s.T().Logf("Diana became staff")

	// Charlie (head) cannot remove himself - institution must have at least one head
	s.Refused(clientCharlie.RemoveMember(institution.Id.String(), clientCharlie.ID), testutil.CodeLastHead)
	s.T().Logf("Charlie cannot remove himself (expected - must have at least one head)")

	// Charlie invites Diana to become head (promoting staff to head)
	dianaHeadInvite := Must(clientCharlie.InviteToInstitution(
		institution.Id.String(),
		"head",
		clientDiana.ID,
	))
	Must(clientDiana.AcceptInvite(dianaHeadInvite.Id.String()))
	s.T().Logf("Diana promoted to head (lost staff role due to single-role enforcement)")

	// Verify both are heads now (2 members: charlie head, diana head)
	// Note: Diana's old staff role was deleted when she accepted the head role
	instWithHeads := Must(clientCharlie.GetInstitution(institution.Id.String()))
	s.Equal(2, len(testutil.MembersOf(instWithHeads)), "should have 2 members")
	headCount := 0
	for _, member := range testutil.MembersOf(instWithHeads) {
		if member.Role == "head" {
			headCount++
		}
	}
	s.Equal(2, headCount, "should have 2 heads (charlie and diana)")
	s.T().Logf("Institution now has 2 heads: charlie and diana")

	// Now Charlie can remove Diana (another head)
	MustSucceed(clientCharlie.RemoveMember(institution.Id.String(), clientDiana.ID))
	s.T().Logf("Charlie removed Diana (head)")

	// Verify Diana was removed (1 member remains: charlie)
	instAfterRemoval := Must(clientCharlie.GetInstitution(institution.Id.String()))
	s.Equal(1, len(testutil.MembersOf(instAfterRemoval)), "should have 1 member (charlie)")
	s.Equal("charlie", testutil.MembersOf(instAfterRemoval)[0].Name)
	s.Equal("head", testutil.MembersOf(instAfterRemoval)[0].Role)
	s.T().Logf("Institution now has 1 member: charlie (head)")

	// Charlie re-invites Diana as head
	dianaHeadInvite2 := Must(clientCharlie.InviteToInstitution(
		institution.Id.String(),
		"head",
		clientDiana.ID,
	))
	Must(clientDiana.AcceptInvite(dianaHeadInvite2.Id.String()))
	s.T().Logf("Diana re-invited and became head again")

	// Verify both are heads again (2 members: charlie, diana)
	instWithBothHeads := Must(clientCharlie.GetInstitution(institution.Id.String()))
	s.Equal(2, len(testutil.MembersOf(instWithBothHeads)), "should have 2 members")
	s.T().Logf("Institution has 2 heads again")

	// Charlie CAN remove himself (leave) since Diana is also a head
	MustSucceed(clientCharlie.RemoveMember(institution.Id.String(), clientCharlie.ID))
	s.T().Logf("Charlie left the institution (Diana remains as head)")

	// Diana is now the last head — she cannot leave
	s.Refused(clientDiana.RemoveMember(institution.Id.String(), clientDiana.ID), testutil.CodeLastHead)
	s.T().Logf("Diana cannot leave (expected - she is the last head)")
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
		institution1.Id.String(),
		"head",
		clientEve.ID,
	))
	Must(clientEve.AcceptInvite(eveInvite1.Id.String()))
	s.T().Logf("Eve became head of institution1")

	// Admin invites Frank as head of institution2
	frankInvite2 := Must(s.clientAdmin.InviteToInstitution(
		institution2.Id.String(),
		"head",
		clientFrank.ID,
	))
	Must(clientFrank.AcceptInvite(frankInvite2.Id.String()))
	s.T().Logf("Frank became head of institution2")

	// Verify each institution has one head
	inst1Members := Must(clientEve.GetInstitution(institution1.Id.String()))
	s.Equal(1, len(testutil.MembersOf(inst1Members)), "institution1 should have 1 member")
	s.Equal("eve", testutil.MembersOf(inst1Members)[0].Name)

	inst2Members := Must(clientFrank.GetInstitution(institution2.Id.String()))
	s.Equal(1, len(testutil.MembersOf(inst2Members)), "institution2 should have 1 member")
	s.Equal("frank", testutil.MembersOf(inst2Members)[0].Name)

	// Frank cannot leave institution2 because he's the last head
	s.Refused(clientFrank.RemoveMember(institution2.Id.String(), clientFrank.ID), testutil.CodeLastHead)
	s.T().Logf("Frank cannot leave institution2 (last head)")

	// Solution: First introduce Grace to become head of institution2
	// This way institution2 won't be headless when Frank leaves
	clientGrace := s.CreateUser("grace")
	graceInvite2 := Must(clientFrank.InviteToInstitution(
		institution2.Id.String(),
		"head",
		clientGrace.ID,
	))
	Must(clientGrace.AcceptInvite(graceInvite2.Id.String()))
	s.T().Logf("Grace became head of institution2")

	// Verify institution2 now has 2 heads
	inst2With2Heads := Must(clientFrank.GetInstitution(institution2.Id.String()))
	s.Equal(2, len(testutil.MembersOf(inst2With2Heads)), "institution2 should have 2 members")
	s.T().Logf("Institution2 now has 2 heads: frank and grace")

	// Now Eve invites Frank to become head of institution1
	frankInvite1 := Must(clientEve.InviteToInstitution(
		institution1.Id.String(),
		"head",
		clientFrank.ID,
	))
	s.T().Logf("Eve invited Frank to become head of institution1")

	// Frank can now accept the invite (institution2 won't be headless)
	Must(clientFrank.AcceptInvite(frankInvite1.Id.String()))
	s.T().Logf("Frank accepted head role in institution1 (lost role in institution2)")

	// Verify Frank is now head of institution1
	inst1WithFrank := Must(clientEve.GetInstitution(institution1.Id.String()))
	s.Equal(2, len(testutil.MembersOf(inst1WithFrank)), "institution1 should have 2 members")
	frankFound := false
	for _, member := range testutil.MembersOf(inst1WithFrank) {
		if member.Name == "frank" {
			frankFound = true
			s.Equal("head", member.Role)
		}
	}
	s.True(frankFound, "frank should be a head in institution1")
	s.T().Logf("Institution1 now has 2 heads: eve and frank")

	// Institution2 now has only Grace (Frank left when accepting new role)
	inst2AfterFrankLeft := Must(clientGrace.GetInstitution(institution2.Id.String()))
	s.Equal(1, len(testutil.MembersOf(inst2AfterFrankLeft)), "institution2 should have 1 member")
	s.Equal("grace", testutil.MembersOf(inst2AfterFrankLeft)[0].Name)
	s.Equal("head", testutil.MembersOf(inst2AfterFrankLeft)[0].Role)
	s.T().Logf("Institution2 now has 1 head: grace (Frank left)")

	// Frank CAN remove himself from institution1 (there are 2 heads: eve and frank)
	MustSucceed(clientFrank.RemoveMember(institution1.Id.String(), clientFrank.ID))
	s.T().Logf("Frank left institution1 (Eve remains as head)")

	// Verify Frank was removed from institution1
	inst1AfterRemoval := Must(clientEve.GetInstitution(institution1.Id.String()))
	s.Equal(1, len(testutil.MembersOf(inst1AfterRemoval)), "institution1 should have 1 member")
	s.Equal("eve", testutil.MembersOf(inst1AfterRemoval)[0].Name)
	s.T().Logf("Institution1 back to 1 head: eve")

	// Eve is now the last head — she cannot leave
	s.Refused(clientEve.RemoveMember(institution1.Id.String(), clientEve.ID), testutil.CodeLastHead)
	s.T().Logf("Eve cannot leave institution1 (she is the last head)")
}

// TestInviteByEmail tests email-based invitations
func (s *MultiUserTestSuite) TestInviteByEmail() {
	// Create institution
	institution := Must(s.clientAdmin.CreateInstitution("Email Invite Test Institution"))
	s.T().Logf("Created institution: %s", institution.Name)

	// Create a user with an email
	clientZara := s.CreateUser("zara-email-test", "zara@example.com")
	zaraMe := Must(clientZara.GetMe())
	s.T().Logf("Created user zara-email-test with email: zara@example.com")

	// Admin tries to invite by an email no account has - should fail
	_, err := s.clientAdmin.InviteToInstitutionByEmail(
		institution.Id.String(),
		"staff",
		"zara-new@example.com",
	)
	s.NotFound(err)
	s.T().Logf("Invite to wrong email correctly rejected")

	// Zara changes their email (using admin endpoint for simplicity)
	newEmail := "zara-new@example.com"
	Must(s.clientAdmin.UpdateUser(zaraMe.Id.String(), testutil.UserUpdate{
		Name:  &zaraMe.Name,
		Email: &newEmail,
	}))
	s.T().Logf("Zara changed email to: zara-new@example.com")

	// Admin invites by the new email - should succeed
	invite := Must(s.clientAdmin.InviteToInstitutionByEmail(
		institution.Id.String(),
		"staff",
		"zara-new@example.com",
	))
	s.NotNil(invite.InvitedEmail)
	s.Equal("zara-new@example.com", *invite.InvitedEmail)
	s.T().Logf("Invite by correct email succeeded")

	// Zara accepts the invite
	Must(clientZara.AcceptInvite(invite.Id.String()))
	s.T().Logf("Zara accepted invite and became staff")

	// Verify Zara is now a member
	institutionAfter := Must(s.clientAdmin.GetInstitution(institution.Id.String()))
	s.Equal(1, len(testutil.MembersOf(institutionAfter)), "should have 1 member")
	s.Equal(zaraMe.Id.String(), testutil.MembersOf(institutionAfter)[0].UserId.String())
	s.Equal("staff", testutil.MembersOf(institutionAfter)[0].Role)
}
