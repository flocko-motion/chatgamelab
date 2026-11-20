package testing

import "cgl/testing/testutil"

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
	s.NotEmpty(institution.Id)
	s.Equal("Workshop Institution", institution.Name)
	s.T().Logf("Created institution: %s (ID: %s)", institution.Name, institution.Id)

	// Admin invites Tony as head
	tonyInvite := Must(s.clientAdmin.InviteToInstitution(
		institution.Id.String(),
		"head",
		clientTony.ID,
	))
	s.Equal("pending", tonyInvite.Status)
	s.T().Logf("Created invite for Tony as head")

	// Tony accepts
	Must(clientTony.AcceptInvite(tonyInvite.Id.String()))
	s.T().Logf("Tony accepted invite and became head")

	// Tony invites Timo as staff
	timoInvite := Must(clientTony.InviteToInstitution(
		institution.Id.String(),
		"staff",
		clientTimo.ID,
	))
	s.Equal("pending", timoInvite.Status)
	s.T().Logf("Tony invited Timo as staff")

	// Timo accepts
	Must(clientTimo.AcceptInvite(timoInvite.Id.String()))
	s.T().Logf("Timo accepted invite and became staff")

	// Tony invites Toto as staff
	totoInvite := Must(clientTony.InviteToInstitution(
		institution.Id.String(),
		"staff",
		clientToto.ID,
	))
	s.Equal("pending", totoInvite.Status)
	s.T().Logf("Tony invited Toto as staff")

	// Toto accepts
	Must(clientToto.AcceptInvite(totoInvite.Id.String()))
	s.T().Logf("Toto accepted invite and became staff")

	// Verify institution has 3 members
	instWithMembers := Must(clientTony.GetInstitution(institution.Id.String()))
	s.Len(testutil.MembersOf(instWithMembers), 3, "institution should have 3 members")

	// Timo creates a workshop
	workshop := Must(clientTimo.CreateWorkshop(institution.Id.String(), "Test Workshop"))
	s.NotEmpty(workshop.Id)
	s.Equal("Test Workshop", workshop.Name)
	s.NotNil(workshop.Institution)
	s.Equal(institution.Id, workshop.Institution.Id)
	s.True(workshop.Active, "workshop should be active by default")
	s.False(workshop.Public, "workshop should be private by default")
	s.T().Logf("Timo created workshop: %s (ID: %s)", workshop.Name, workshop.Id)

	// Timo updates workshop name
	updatedName := Must(clientTimo.UpdateWorkshop(workshop.Id.String(), testutil.WorkshopUpdate{Name: "Updated Workshop Name", Active: workshop.Active, Public: workshop.Public}))
	s.Equal("Updated Workshop Name", updatedName.Name)
	s.True(updatedName.Active)
	s.False(updatedName.Public)
	s.T().Logf("Timo updated workshop name to: %s", updatedName.Name)

	// Timo sets workshop to inactive
	updatedActive := Must(clientTimo.UpdateWorkshop(workshop.Id.String(), testutil.WorkshopUpdate{Name: updatedName.Name, Active: false, Public: updatedName.Public}))
	s.Equal("Updated Workshop Name", updatedActive.Name)
	s.False(updatedActive.Active, "workshop should now be inactive")
	s.False(updatedActive.Public)
	s.T().Logf("Timo set workshop to inactive")

	// Timo makes workshop public
	updatedPublic := Must(clientTimo.UpdateWorkshop(workshop.Id.String(), testutil.WorkshopUpdate{Name: updatedActive.Name, Active: updatedActive.Active, Public: true}))
	s.Equal("Updated Workshop Name", updatedPublic.Name)
	s.False(updatedPublic.Active)
	s.True(updatedPublic.Public, "workshop should now be public")
	s.T().Logf("Timo made workshop public")

	// Verify final state
	finalWorkshop := Must(clientTimo.GetWorkshop(workshop.Id.String()))
	s.Equal("Updated Workshop Name", finalWorkshop.Name)
	s.False(finalWorkshop.Active)
	s.True(finalWorkshop.Public)
	s.T().Logf("Final workshop state verified: name=%s, active=%v, public=%v",
		finalWorkshop.Name, finalWorkshop.Active, finalWorkshop.Public)

	// Test permissions: Toto (staff) can view and edit workshops
	totoView := Must(clientToto.GetWorkshop(workshop.Id.String()))
	s.Equal("Updated Workshop Name", totoView.Name)
	s.T().Logf("Toto can view workshop: %s", totoView.Name)

	// Toto (staff) can edit the workshop (staff can modify workshops in their institution)
	totoEdit := Must(clientToto.UpdateWorkshop(workshop.Id.String(), testutil.WorkshopUpdate{Name: "Toto's Edit", Active: true}))
	s.Equal("Toto's Edit", totoEdit.Name)
	s.True(totoEdit.Active)
	s.False(totoEdit.Public)
	s.T().Logf("Toto (staff) can edit workshop: %s", totoEdit.Name)

	// Tony (head) can edit the workshop even though he's not the owner
	tonyEdit := Must(clientTony.UpdateWorkshop(workshop.Id.String(), testutil.WorkshopUpdate{Name: "Tony's Edit", Active: true}))
	s.Equal("Tony's Edit", tonyEdit.Name)
	s.True(tonyEdit.Active)
	s.False(tonyEdit.Public)
	s.T().Logf("Tony (head) can edit workshop: %s", tonyEdit.Name)

	// Timo (owner) can still edit
	timoEdit := Must(clientTimo.UpdateWorkshop(workshop.Id.String(), testutil.WorkshopUpdate{Name: "Timo's Final Edit", Public: true}))
	s.Equal("Timo's Final Edit", timoEdit.Name)
	s.False(timoEdit.Active)
	s.True(timoEdit.Public)
	s.T().Logf("Timo (owner) can edit workshop: %s", timoEdit.Name)

	// Verify all members can view the final state
	tonyFinalView := Must(clientTony.GetWorkshop(workshop.Id.String()))
	s.Equal("Timo's Final Edit", tonyFinalView.Name)
	totoFinalView := Must(clientToto.GetWorkshop(workshop.Id.String()))
	s.Equal("Timo's Final Edit", totoFinalView.Name)
	s.T().Logf("All members can view final workshop state")

	// Test workshop listing permissions
	// Institution members can list workshops
	tonyWorkshops := Must(clientTony.ListWorkshops(institution.Id.String()))
	s.Equal(1, len(tonyWorkshops), "Tony should see 1 workshop")
	s.Equal("Timo's Final Edit", tonyWorkshops[0].Name)
	s.T().Logf("Tony (head) can list workshops: %d found", len(tonyWorkshops))

	timoWorkshops := Must(clientTimo.ListWorkshops(institution.Id.String()))
	s.Equal(1, len(timoWorkshops), "Timo should see 1 workshop")
	s.T().Logf("Timo (owner) can list workshops: %d found", len(timoWorkshops))

	totoWorkshops := Must(clientToto.ListWorkshops(institution.Id.String()))
	s.Equal(1, len(totoWorkshops), "Toto should see 1 workshop")
	s.T().Logf("Toto (staff) can list workshops: %d found", len(totoWorkshops))

	// Steve (not a member) cannot list workshops
	_, steveListErr := clientSteve.ListWorkshops(institution.Id.String())
	s.Forbidden(steveListErr, "Steve should not be able to list workshops (not a member)")
	s.T().Logf("Steve cannot list workshops (expected)")

	// Create workshop invites to test invite visibility
	workshopInvite := Must(clientTimo.CreateWorkshopInvite(workshop.Id.String()))
	s.NotEmpty(workshopInvite.Id)
	s.T().Logf("Timo created workshop invite: %s", workshopInvite.Id)

	// Staff members can see workshop invites
	timoViewWithInvites := Must(clientTimo.GetWorkshop(workshop.Id.String()))
	s.Equal(1, len(testutil.WorkshopInvitesOf(timoViewWithInvites)), "Timo (staff) should see 1 invite")
	s.T().Logf("Timo can see %d workshop invite(s)", len(testutil.WorkshopInvitesOf(timoViewWithInvites)))

	tonyViewWithInvites := Must(clientTony.GetWorkshop(workshop.Id.String()))
	s.Equal(1, len(testutil.WorkshopInvitesOf(tonyViewWithInvites)), "Tony (head) should see 1 invite")
	s.T().Logf("Tony can see %d workshop invite(s)", len(testutil.WorkshopInvitesOf(tonyViewWithInvites)))

	totoViewWithInvites := Must(clientToto.GetWorkshop(workshop.Id.String()))
	s.Equal(1, len(testutil.WorkshopInvitesOf(totoViewWithInvites)), "Toto (staff) should see 1 invite")
	s.T().Logf("Toto can see %d workshop invite(s)", len(testutil.WorkshopInvitesOf(totoViewWithInvites)))

	// Steve CAN view the public workshop by ID (to check if it's active)
	// but CANNOT see invites (not a member)
	steveViewPublic := Must(clientSteve.GetWorkshop(workshop.Id.String()))
	s.Equal("Timo's Final Edit", steveViewPublic.Name)
	s.False(steveViewPublic.Active, "steve can see workshop is inactive")
	s.True(steveViewPublic.Public, "steve can see workshop is public")
	s.Equal(0, len(testutil.WorkshopInvitesOf(steveViewPublic)), "Steve should not see invites (not a member)")
	s.T().Logf("Steve can view public workshop by ID: %s (active=%v, invites=%d)",
		steveViewPublic.Name, steveViewPublic.Active, len(testutil.WorkshopInvitesOf(steveViewPublic)))

	// Test invite management: Toto (staff) can revoke workshop invites
	MustSucceed(clientToto.RevokeInvite(workshopInvite.Id.String()))
	s.T().Logf("Toto (staff) revoked workshop invite")

	// The invite is deleted (hard delete): it can't be found any more
	_, err := clientTimo.GetInvite(workshopInvite.Id.String())
	s.NotFound(err)
	s.T().Logf("Invite was hard-deleted (expected)")

	// Make workshop private - now Steve cannot view it
	privateWorkshop := Must(clientTimo.UpdateWorkshop(workshop.Id.String(), testutil.WorkshopUpdate{Name: "Timo's Final Edit", Public: false}))
	s.False(privateWorkshop.Public, "workshop should now be private")
	s.T().Logf("Workshop is now private")

	// Steve cannot view private workshop
	_, steveViewPrivateErr := clientSteve.GetWorkshop(workshop.Id.String())
	s.Forbidden(steveViewPrivateErr, "Steve should not be able to view private workshop")
	s.T().Logf("Steve cannot view private workshop (expected)")
}

// TestUpdateParticipantName tests that Head/Staff can update participant names in their workshops
// but cannot update names of participants in other institutions or update regular users
func (s *MultiUserTestSuite) TestUpdateParticipantName() {
	// Create users for institution A
	clientHeadA := s.CreateUser("head-inst-a")
	clientStaffA := s.CreateUser("staff-inst-a")

	// Create users for institution B
	clientHeadB := s.CreateUser("head-inst-b")

	// Create a regular user (not a participant)
	clientRegularUser := s.CreateUser("regular-user")

	// Admin creates institution A
	institutionA := Must(s.clientAdmin.CreateInstitution("Institution A"))
	s.T().Logf("Created institution A: %s", institutionA.Name)

	// Admin creates institution B
	institutionB := Must(s.clientAdmin.CreateInstitution("Institution B"))
	s.T().Logf("Created institution B: %s", institutionB.Name)

	// Setup institution A: Head and Staff
	headAInvite := Must(s.clientAdmin.InviteToInstitution(institutionA.Id.String(), "head", clientHeadA.ID))
	Must(clientHeadA.AcceptInvite(headAInvite.Id.String()))
	s.T().Logf("Head A joined institution A")

	staffAInvite := Must(clientHeadA.InviteToInstitution(institutionA.Id.String(), "staff", clientStaffA.ID))
	Must(clientStaffA.AcceptInvite(staffAInvite.Id.String()))
	s.T().Logf("Staff A joined institution A")

	// Setup institution B: Head only
	headBInvite := Must(s.clientAdmin.InviteToInstitution(institutionB.Id.String(), "head", clientHeadB.ID))
	Must(clientHeadB.AcceptInvite(headBInvite.Id.String()))
	s.T().Logf("Head B joined institution B")

	// Staff A creates a workshop in institution A
	workshopA := Must(clientStaffA.CreateWorkshop(institutionA.Id.String(), "Workshop A"))
	s.T().Logf("Created workshop A: %s", workshopA.Name)

	// Create workshop invite and have an anonymous user join
	workshopInvite := Must(clientStaffA.CreateWorkshopInvite(workshopA.Id.String()))
	s.NotNil(workshopInvite.InviteToken)
	s.T().Logf("Created workshop invite with token")

	// Anonymous participant joins the workshop
	participantResponse, err := s.AcceptWorkshopInviteAnonymously(*workshopInvite.InviteToken)
	s.NoError(err)
	s.NotNil(participantResponse.User)
	participantID := participantResponse.User.Id.String()
	originalName := participantResponse.User.Name
	s.T().Logf("Participant joined workshop A: %s (ID: %s)", originalName, participantID)

	// === Test 1: Staff A can update participant name in their workshop ===
	newName := "Updated Participant Name"
	updatedUser, err := clientStaffA.UpdateUserName(participantID, newName)
	s.NoError(err, "Staff A should be able to update participant name in their workshop")
	s.Equal(newName, updatedUser.Name)
	s.T().Logf("✓ Staff A updated participant name to: %s", newName)

	// === Test 2: Head A can update participant name in their institution's workshop ===
	newName2 := "Head Updated Name"
	updatedUser2, err := clientHeadA.UpdateUserName(participantID, newName2)
	s.NoError(err, "Head A should be able to update participant name in their institution's workshop")
	s.Equal(newName2, updatedUser2.Name)
	s.T().Logf("✓ Head A updated participant name to: %s", newName2)

	// === Test 3: Head B (different institution) CANNOT update participant name ===
	_, err = clientHeadB.UpdateUserName(participantID, "Forbidden Name")
	s.Forbidden(err, "Head B should NOT be able to update participant in another institution")
	s.T().Logf("✓ Head B correctly denied - cannot update participant in another institution")

	// === Test 4: Staff A CANNOT update a regular user's name ===
	_, err = clientStaffA.UpdateUserName(clientRegularUser.ID, "Hacked Name")
	s.Forbidden(err, "Staff A should NOT be able to update a regular user's name")
	s.T().Logf("✓ Staff A correctly denied - cannot update regular user's name")

	// === Test 5: Head A CANNOT update a regular user's name ===
	_, err = clientHeadA.UpdateUserName(clientRegularUser.ID, "Hacked Name")
	s.Forbidden(err, "Head A should NOT be able to update a regular user's name")
	s.T().Logf("✓ Head A correctly denied - cannot update regular user's name")

	// === Test 6: Regular user CAN update their own name ===
	_, err = clientRegularUser.UpdateUserName(clientRegularUser.ID, "My New Name")
	s.NoError(err, "Users should be able to update their own name")
	s.T().Logf("✓ Regular user can update their own name")

	// Verify final state
	workshop := Must(clientHeadA.GetWorkshop(workshopA.Id.String()))
	found := false
	for _, p := range testutil.WorkshopParticipantsOf(workshop) {
		if p.Id.String() == participantID {
			s.Equal(newName2, p.Name, "Participant name should be the last updated value")
			found = true
			break
		}
	}
	s.True(found, "Participant should still be in workshop")
	s.T().Logf("✓ Final verification: participant name is correct in workshop")
}

// TestWorkshopInvites tests the complete workshop invite workflow:
// institution with one head and one staff member, creating one workshop,
// creating public invite, anonymous users coming with such a token get a
// ad-hoc created user account and are added to the workshop as participant - they are signed in right away and keep a participant token to sign in again
// workshop owner(s) can list such users and deactivate them at will (user account is deleted)
// they can revoke the invite to stop new users coming in
func (s *MultiUserTestSuite) TestWorkshopInvites() {
	// Create users
	clientHead := s.CreateUser("workshop-head")
	clientStaff := s.CreateUser("workshop-staff")

	// Admin creates institution
	institution := Must(s.clientAdmin.CreateInstitution("Workshop Invite Test Institution"))
	s.T().Logf("Created institution: %s", institution.Name)

	// Admin invites head
	headInvite := Must(s.clientAdmin.InviteToInstitution(
		institution.Id.String(),
		"head",
		clientHead.ID,
	))
	Must(clientHead.AcceptInvite(headInvite.Id.String()))
	s.T().Logf("Head joined institution")

	// Head invites staff
	staffInvite := Must(clientHead.InviteToInstitution(
		institution.Id.String(),
		"staff",
		clientStaff.ID,
	))
	Must(clientStaff.AcceptInvite(staffInvite.Id.String()))
	s.T().Logf("Staff joined institution")

	// Staff creates a workshop
	workshop := Must(clientStaff.CreateWorkshop(institution.Id.String(), "Anonymous Participant Workshop"))
	s.NotEmpty(workshop.Id)
	s.Equal("Anonymous Participant Workshop", workshop.Name)
	s.T().Logf("Created workshop: %s", workshop.Name)

	// Staff creates a workshop invite (open invite with token)
	workshopInvite := Must(clientStaff.CreateWorkshopInvite(workshop.Id.String()))
	s.NotEmpty(workshopInvite.Id)
	s.NotNil(workshopInvite.InviteToken)
	s.T().Logf("Created workshop invite with token: %s", *workshopInvite.InviteToken)

	// Anonymous user accepts invite via token (no authentication)
	// This should create an ad-hoc user account with a funny name and return auth token
	participant1Response, err := s.AcceptWorkshopInviteAnonymously(*workshopInvite.InviteToken)
	s.NoError(err, "should be able to accept workshop invite anonymously")
	s.NotNil(participant1Response.User)
	s.NotEmpty(participant1Response.User.Name)
	s.NotNil(participant1Response.AuthToken)
	s.T().Logf("Anonymous user 1 created: %s (token: %s...)", participant1Response.User.Name, (*participant1Response.AuthToken)[:10])

	// Create client for participant 1 using ONLY their auth token (low-barrier login)
	clientParticipant1 := Must(s.ParticipantLogin(*participant1Response.AuthToken))
	s.Equal(participant1Response.User.Name, clientParticipant1.Name)
	s.T().Logf("Participant 1 authenticated with token only")

	// Participant 1 can see workshop
	_ = Must(clientParticipant1.GetWorkshop(workshop.Id.String()))
	s.T().Logf("Participant 1 can see workshop")

	// Another anonymous user accepts the same invite
	participant2Response, err := s.AcceptWorkshopInviteAnonymously(*workshopInvite.InviteToken)
	s.NoError(err, "should be able to accept workshop invite anonymously")
	s.NotNil(participant2Response.User)
	s.NotEmpty(participant2Response.User.Name)
	s.NotEqual(participant1Response.User.Name, participant2Response.User.Name, "participants should have different names")
	s.T().Logf("Anonymous user 2 created: %s", participant2Response.User.Name)

	// Participant can see workshop and all other participants (they're in the same workshop!)
	workshopFromParticipant := Must(clientParticipant1.GetWorkshop(workshop.Id.String()))
	s.Equal(workshop.Id, workshopFromParticipant.Id)
	// Should see: workshop owner (staff) + 2 anonymous participants = 3 total
	s.Equal(3, len(testutil.WorkshopParticipantsOf(workshopFromParticipant)), "should see workshop owner + 2 anonymous participants")
	s.T().Logf("Participant 1 can see %d participants (owner + anonymous users)", len(testutil.WorkshopParticipantsOf(workshopFromParticipant)))

	// Staff can see workshop participants
	workshopFromStaff := Must(clientStaff.GetWorkshop(workshop.Id.String()))
	s.Equal(workshop.Id, workshopFromStaff.Id)
	s.Equal(3, len(testutil.WorkshopParticipantsOf(workshopFromStaff)), "should see workshop owner + 2 anonymous participants")
	s.T().Logf("Staff can see %d participants (owner + anonymous users)", len(testutil.WorkshopParticipantsOf(workshopFromStaff)))

	// Let's pick one of the participants..
	participant := testutil.WorkshopParticipantsOf(workshopFromStaff)[1]
	s.Equal("participant", participant.Role)
	// ..and assume, he lost his access token by closing his browser - the workshop leader can read out his participant token
	participantToken := Must(clientStaff.GetParticipantToken(participant.Id.String()))
	s.NotEmpty(participantToken)
	s.T().Logf("Participant token provided for participant: %s", participantToken)
	// ..and the participant signs in with it
	clientParticipantRecovered := Must(s.ParticipantLogin(participantToken))
	participantRecoveredMe := Must(clientParticipantRecovered.GetMe())
	s.Equal(participant.Name, participantRecoveredMe.Name)
	s.T().Logf("The participant signed in again with their participant token")

	// Head can see workshop participants
	headWorkshopView := Must(clientHead.GetWorkshop(workshop.Id.String()))
	s.Equal(workshop.Id, headWorkshopView.Id)
	s.Equal(3, len(testutil.WorkshopParticipantsOf(headWorkshopView)), "should see workshop owner + 2 anonymous participants")
	s.T().Logf("Head can see %d participants (owner + anonymous users)", len(testutil.WorkshopParticipantsOf(headWorkshopView)))

	// Staff revokes the invite to stop new users from joining
	MustSucceed(clientStaff.RevokeInvite(workshopInvite.Id.String()))
	s.T().Logf("Staff revoked workshop invite")

	// The invite is gone
	_, err = clientStaff.GetInvite(workshopInvite.Id.String())
	s.NotFound(err)
	s.T().Logf("Invite was hard-deleted (expected)")

	// New anonymous user cannot accept revoked invite
	_, acceptRevokedErr := s.AcceptWorkshopInviteAnonymously(*workshopInvite.InviteToken)
	s.NotFound(acceptRevokedErr, "should not be able to accept revoked invite")
	s.T().Logf("Revoked invite correctly rejected new participant")

	// Staff can deactivate/remove a participant (deletes user account)
	MustSucceed(clientStaff.DeleteUser(participant1Response.User.Id.String()))
	s.T().Logf("Staff removed participant 1 (user account deleted)")

	// Verify participant 1 is deleted (cannot authenticate anymore)
	_, getDeletedUserErr := clientParticipant1.GetMe()
	s.Forbidden(getDeletedUserErr, "deleted participant's session no longer names an account")
	s.T().Logf("Deleted participant cannot authenticate (expected)")

	// Participant 2 should still be able to authenticate
	clientParticipant2 := Must(s.ParticipantLogin(*participant2Response.AuthToken))
	participant2Me := Must(clientParticipant2.GetMe())
	s.Equal(participant2Response.User.Name, participant2Me.Name)
	s.T().Logf("Participant 2 can still authenticate")

	// Head can also remove participants
	MustSucceed(clientHead.DeleteUser(participant2Response.User.Id.String()))
	s.T().Logf("Head removed participant 2 (user account deleted)")
}
