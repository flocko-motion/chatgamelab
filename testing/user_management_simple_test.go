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

	// Verify initial state (should have 1 dev user)
	initialUsers := Must(s.DevUser().GetUsers())
	s.Equal(1, len(initialUsers), "should have 1 user (dev user)")
	s.T().Logf("Initial users: %d", len(initialUsers))

	// Create admin user for all tests
	s.clientAdmin = s.CreateUser("admin")
	s.Role(s.clientAdmin, string(obj.RoleAdmin))
	s.Equal("admin", s.clientAdmin.Name)
	s.Equal("admin@test.local", s.clientAdmin.Email)
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
	adminInvites := Must(s.clientAdmin.GetInvites())
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
	adminInvites = Must(s.clientAdmin.GetInvites())
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

// TestWorkshopManagement creating a single institution with tony as head and timo as staff
// timo will create a workshop and generate invite links
// anonymous users can join the workshop - they will be assigned a random name
func (s *MultiUserTestSuite) TestWorkshopManagement() {
	// Create users
	clientTony := s.CreateUser("tony")
	s.Equal("tony", clientTony.Name)
	s.Equal("tony@test.local", clientTony.Email)

	clientTimo := s.CreateUser("timo")
	s.Equal("timo", clientTimo.Name)
	s.Equal("timo@test.local", clientTimo.Email)

	clientToto := s.CreateUser("toto")
	s.Equal("toto", clientToto.Name)
	s.Equal("toto@test.local", clientToto.Email)

	clientSteve := s.CreateUser("steve")
	s.Equal("steve", clientSteve.Name)
	s.Equal("steve@test.local", clientSteve.Email)

	// Admin creates institution
	institution := Must(s.clientAdmin.CreateInstitution("Workshop Institution"))
	s.NotEmpty(institution.ID)
	s.Equal("Workshop Institution", institution.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution.Name, institution.ID)

	// Admin invites Tony as head
	tonyInvite := Must(s.clientAdmin.InviteToInstitution(
		institution.ID.String(),
		string(obj.RoleHead),
		clientTony.ID,
	))
	s.Equal(obj.InviteStatusPending, tonyInvite.Status)
	s.T().Logf("Created invite for Tony as head")

	// Tony accepts
	Must(clientTony.AcceptInvite(tonyInvite.ID.String()))
	s.T().Logf("Tony accepted invite and became head")

	// Tony invites Timo as staff
	timoInvite := Must(clientTony.InviteToInstitution(
		institution.ID.String(),
		string(obj.RoleStaff),
		clientTimo.ID,
	))
	s.Equal(obj.InviteStatusPending, timoInvite.Status)
	s.T().Logf("Tony invited Timo as staff")

	// Timo accepts
	Must(clientTimo.AcceptInvite(timoInvite.ID.String()))
	s.T().Logf("Timo accepted invite and became staff")

	// Tony invites Toto as staff
	totoInvite := Must(clientTony.InviteToInstitution(
		institution.ID.String(),
		string(obj.RoleStaff),
		clientToto.ID,
	))
	s.Equal(obj.InviteStatusPending, totoInvite.Status)
	s.T().Logf("Tony invited Toto as staff")

	// Toto accepts
	Must(clientToto.AcceptInvite(totoInvite.ID.String()))
	s.T().Logf("Toto accepted invite and became staff")

	// Verify institution has 3 members
	instWithMembers := Must(clientTony.GetInstitution(institution.ID.String()))
	s.Equal(3, len(instWithMembers.Members), "institution should have 3 members")
	s.T().Logf("Institution has %d members", len(instWithMembers.Members))

	// Timo creates a workshop
	workshop := Must(clientTimo.CreateWorkshop(institution.ID.String(), "Test Workshop"))
	s.NotEmpty(workshop.ID)
	s.Equal("Test Workshop", workshop.Name)
	s.NotNil(workshop.Institution)
	s.Equal(institution.ID, workshop.Institution.ID)
	s.True(workshop.Active, "workshop should be active by default")
	s.False(workshop.Public, "workshop should be private by default")
	s.T().Logf("Timo created workshop: %s (ID: %s)", workshop.Name, workshop.ID)

	// Timo updates workshop name
	updatedName := Must(clientTimo.UpdateWorkshop(workshop.ID.String(), map[string]interface{}{
		"name":   "Updated Workshop Name",
		"active": workshop.Active,
		"public": workshop.Public,
	}))
	s.Equal("Updated Workshop Name", updatedName.Name)
	s.True(updatedName.Active)
	s.False(updatedName.Public)
	s.T().Logf("Timo updated workshop name to: %s", updatedName.Name)

	// Timo sets workshop to inactive
	updatedActive := Must(clientTimo.UpdateWorkshop(workshop.ID.String(), map[string]interface{}{
		"name":   updatedName.Name,
		"active": false,
		"public": updatedName.Public,
	}))
	s.Equal("Updated Workshop Name", updatedActive.Name)
	s.False(updatedActive.Active, "workshop should now be inactive")
	s.False(updatedActive.Public)
	s.T().Logf("Timo set workshop to inactive")

	// Timo makes workshop public
	updatedPublic := Must(clientTimo.UpdateWorkshop(workshop.ID.String(), map[string]interface{}{
		"name":   updatedActive.Name,
		"active": updatedActive.Active,
		"public": true,
	}))
	s.Equal("Updated Workshop Name", updatedPublic.Name)
	s.False(updatedPublic.Active)
	s.True(updatedPublic.Public, "workshop should now be public")
	s.T().Logf("Timo made workshop public")

	// Verify final state
	finalWorkshop := Must(clientTimo.GetWorkshop(workshop.ID.String()))
	s.Equal("Updated Workshop Name", finalWorkshop.Name)
	s.False(finalWorkshop.Active)
	s.True(finalWorkshop.Public)
	s.T().Logf("Final workshop state verified: name=%s, active=%v, public=%v",
		finalWorkshop.Name, finalWorkshop.Active, finalWorkshop.Public)

	// Test permissions: Toto (staff, not owner) can view but not edit
	totoView := Must(clientToto.GetWorkshop(workshop.ID.String()))
	s.Equal("Updated Workshop Name", totoView.Name)
	s.T().Logf("Toto can view workshop: %s", totoView.Name)

	// Toto cannot edit the workshop (not owner)
	_, totoEditErr := clientToto.UpdateWorkshop(workshop.ID.String(), map[string]interface{}{
		"name":   "Toto's Edit",
		"active": true,
		"public": false,
	})
	s.Error(totoEditErr, "Toto should not be able to edit workshop (not owner)")
	s.T().Logf("Toto cannot edit workshop (expected)")

	// Tony (head) can edit the workshop even though he's not the owner
	tonyEdit := Must(clientTony.UpdateWorkshop(workshop.ID.String(), map[string]interface{}{
		"name":   "Tony's Edit",
		"active": true,
		"public": false,
	}))
	s.Equal("Tony's Edit", tonyEdit.Name)
	s.True(tonyEdit.Active)
	s.False(tonyEdit.Public)
	s.T().Logf("Tony (head) can edit workshop: %s", tonyEdit.Name)

	// Timo (owner) can still edit
	timoEdit := Must(clientTimo.UpdateWorkshop(workshop.ID.String(), map[string]interface{}{
		"name":   "Timo's Final Edit",
		"active": false,
		"public": true,
	}))
	s.Equal("Timo's Final Edit", timoEdit.Name)
	s.False(timoEdit.Active)
	s.True(timoEdit.Public)
	s.T().Logf("Timo (owner) can edit workshop: %s", timoEdit.Name)

	// Verify all members can view the final state
	tonyFinalView := Must(clientTony.GetWorkshop(workshop.ID.String()))
	s.Equal("Timo's Final Edit", tonyFinalView.Name)
	totoFinalView := Must(clientToto.GetWorkshop(workshop.ID.String()))
	s.Equal("Timo's Final Edit", totoFinalView.Name)
	s.T().Logf("All members can view final workshop state")

	// Test workshop listing permissions
	// Institution members can list workshops
	tonyWorkshops := Must(clientTony.ListWorkshops(institution.ID.String()))
	s.Equal(1, len(tonyWorkshops), "Tony should see 1 workshop")
	s.Equal("Timo's Final Edit", tonyWorkshops[0].Name)
	s.T().Logf("Tony (head) can list workshops: %d found", len(tonyWorkshops))

	timoWorkshops := Must(clientTimo.ListWorkshops(institution.ID.String()))
	s.Equal(1, len(timoWorkshops), "Timo should see 1 workshop")
	s.T().Logf("Timo (owner) can list workshops: %d found", len(timoWorkshops))

	totoWorkshops := Must(clientToto.ListWorkshops(institution.ID.String()))
	s.Equal(1, len(totoWorkshops), "Toto should see 1 workshop")
	s.T().Logf("Toto (staff) can list workshops: %d found", len(totoWorkshops))

	// Steve (not a member) cannot list workshops
	_, steveListErr := clientSteve.ListWorkshops(institution.ID.String())
	s.Error(steveListErr, "Steve should not be able to list workshops (not a member)")
	s.T().Logf("Steve cannot list workshops (expected)")

	// Create workshop invites to test invite visibility
	workshopInvite := Must(clientTimo.CreateWorkshopInvite(workshop.ID.String(), string(obj.RoleParticipant)))
	s.NotEmpty(workshopInvite.ID)
	s.T().Logf("Timo created workshop invite: %s", workshopInvite.ID)

	// Staff members can see workshop invites
	timoViewWithInvites := Must(clientTimo.GetWorkshop(workshop.ID.String()))
	s.Equal(1, len(timoViewWithInvites.Invites), "Timo (staff) should see 1 invite")
	s.T().Logf("Timo can see %d workshop invite(s)", len(timoViewWithInvites.Invites))

	tonyViewWithInvites := Must(clientTony.GetWorkshop(workshop.ID.String()))
	s.Equal(1, len(tonyViewWithInvites.Invites), "Tony (head) should see 1 invite")
	s.T().Logf("Tony can see %d workshop invite(s)", len(tonyViewWithInvites.Invites))

	totoViewWithInvites := Must(clientToto.GetWorkshop(workshop.ID.String()))
	s.Equal(1, len(totoViewWithInvites.Invites), "Toto (staff) should see 1 invite")
	s.T().Logf("Toto can see %d workshop invite(s)", len(totoViewWithInvites.Invites))

	// Steve CAN view the public workshop by ID (to check if it's active)
	// but CANNOT see invites (not a member)
	steveViewPublic := Must(clientSteve.GetWorkshop(workshop.ID.String()))
	s.Equal("Timo's Final Edit", steveViewPublic.Name)
	s.False(steveViewPublic.Active, "steve can see workshop is inactive")
	s.True(steveViewPublic.Public, "steve can see workshop is public")
	s.Equal(0, len(steveViewPublic.Invites), "Steve should not see invites (not a member)")
	s.T().Logf("Steve can view public workshop by ID: %s (active=%v, invites=%d)",
		steveViewPublic.Name, steveViewPublic.Active, len(steveViewPublic.Invites))

	// Make workshop private - now Steve cannot view it
	privateWorkshop := Must(clientTimo.UpdateWorkshop(workshop.ID.String(), map[string]interface{}{
		"name":   "Timo's Final Edit",
		"active": false,
		"public": false,
	}))
	s.False(privateWorkshop.Public, "workshop should now be private")
	s.T().Logf("Workshop is now private")

	// Steve cannot view private workshop
	_, steveViewPrivateErr := clientSteve.GetWorkshop(workshop.ID.String())
	s.Error(steveViewPrivateErr, "Steve should not be able to view private workshop")
	s.T().Logf("Steve cannot view private workshop (expected)")
}
