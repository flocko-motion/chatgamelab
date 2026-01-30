package testing

import "cgl/obj"

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

	// Test permissions: Toto (staff) can view and edit workshops
	totoView := Must(clientToto.GetWorkshop(workshop.ID.String()))
	s.Equal("Updated Workshop Name", totoView.Name)
	s.T().Logf("Toto can view workshop: %s", totoView.Name)

	// Toto (staff) can edit the workshop (staff can modify workshops in their institution)
	totoEdit := Must(clientToto.UpdateWorkshop(workshop.ID.String(), map[string]interface{}{
		"name":   "Toto's Edit",
		"active": true,
		"public": false,
	}))
	s.Equal("Toto's Edit", totoEdit.Name)
	s.True(totoEdit.Active)
	s.False(totoEdit.Public)
	s.T().Logf("Toto (staff) can edit workshop: %s", totoEdit.Name)

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

	// Test invite management: Toto (staff) can revoke workshop invites
	MustSucceed(clientToto.RevokeInvite(workshopInvite.ID.String()))
	s.T().Logf("Toto (staff) revoked workshop invite")

	// Verify invite is deleted (hard delete) - fetching it should fail
	Fail(clientTimo.GetInvite(workshopInvite.ID.String()))
	s.T().Logf("Invite was hard-deleted (expected)")

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

// TestWorkshopInvites tests the complete workshop invite workflow:
// institution with one head and one staff member, creating one workshop,
// creating public invite, anonymous users coming with such a token get a
// ad-hoc created user account and are added to the workshop as participant - the user authenticates with a newly generated token (token can be traded for jwt anytime)
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
		institution.ID.String(),
		string(obj.RoleHead),
		clientHead.ID,
	))
	Must(clientHead.AcceptInvite(headInvite.ID.String()))
	s.T().Logf("Head joined institution")

	// Head invites staff
	staffInvite := Must(clientHead.InviteToInstitution(
		institution.ID.String(),
		string(obj.RoleStaff),
		clientStaff.ID,
	))
	Must(clientStaff.AcceptInvite(staffInvite.ID.String()))
	s.T().Logf("Staff joined institution")

	// Staff creates a workshop
	workshop := Must(clientStaff.CreateWorkshop(institution.ID.String(), "Anonymous Participant Workshop"))
	s.NotEmpty(workshop.ID)
	s.Equal("Anonymous Participant Workshop", workshop.Name)
	s.T().Logf("Created workshop: %s", workshop.Name)

	// Staff creates a workshop invite (open invite with token)
	workshopInvite := Must(clientStaff.CreateWorkshopInvite(workshop.ID.String(), string(obj.RoleParticipant)))
	s.NotEmpty(workshopInvite.ID)
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
	clientParticipant1 := s.CreateUserWithToken(*participant1Response.AuthToken)
	s.Equal(participant1Response.User.Name, clientParticipant1.Name)
	s.T().Logf("Participant 1 authenticated with token only")

	// Participant 1 can see workshop
	_ = Must(clientParticipant1.GetWorkshop(workshop.ID.String()))
	s.T().Logf("Participant 1 can see workshop")

	// Another anonymous user accepts the same invite
	participant2Response, err := s.AcceptWorkshopInviteAnonymously(*workshopInvite.InviteToken)
	s.NoError(err, "should be able to accept workshop invite anonymously")
	s.NotNil(participant2Response.User)
	s.NotEmpty(participant2Response.User.Name)
	s.NotEqual(participant1Response.User.Name, participant2Response.User.Name, "participants should have different names")
	s.T().Logf("Anonymous user 2 created: %s", participant2Response.User.Name)

	// Participant can see workshop and all other participants (they're in the same workshop!)
	workshopFromParticipant := Must(clientParticipant1.GetWorkshop(workshop.ID.String()))
	s.Equal(workshop.ID, workshopFromParticipant.ID)
	// Should see: workshop owner (staff) + 2 anonymous participants = 3 total
	s.Equal(3, len(workshopFromParticipant.Participants), "should see workshop owner + 2 anonymous participants")
	s.T().Logf("Participant 1 can see %d participants (owner + anonymous users)", len(workshopFromParticipant.Participants))

	// Staff can see workshop participants
	_ = Must(clientStaff.GetWorkshop(workshop.ID.String()))
	s.T().Logf("Workshop has participants (checking via workshop view)")

	// Head can also see participants
	headWorkshopView := Must(clientHead.GetWorkshop(workshop.ID.String()))
	s.Equal(workshop.ID, headWorkshopView.ID)
	s.T().Logf("Head can view workshop with participants")

	// Staff revokes the invite to stop new users from joining
	MustSucceed(clientStaff.RevokeInvite(workshopInvite.ID.String()))
	s.T().Logf("Staff revoked workshop invite")

	// Verify invite is revoked
	revokedInvite := Must(clientStaff.GetInvite(workshopInvite.ID.String()))
	s.Equal(obj.InviteStatusRevoked, revokedInvite.Status)
	s.T().Logf("Invite status confirmed: %s", revokedInvite.Status)

	// New anonymous user cannot accept revoked invite
	_, acceptRevokedErr := s.AcceptWorkshopInviteAnonymously(*workshopInvite.InviteToken)
	s.Error(acceptRevokedErr, "should not be able to accept revoked invite")
	s.T().Logf("Revoked invite correctly rejected new participant")

	// Staff can deactivate/remove a participant (deletes user account)
	MustSucceed(clientStaff.DeleteUser(participant1Response.User.ID.String()))
	s.T().Logf("Staff removed participant 1 (user account deleted)")

	// Verify participant 1 is deleted (cannot authenticate anymore)
	_, getDeletedUserErr := clientParticipant1.GetMe()
	s.Error(getDeletedUserErr, "deleted participant should not be able to authenticate")
	s.T().Logf("Deleted participant cannot authenticate (expected)")

	// Participant 2 should still be able to authenticate
	clientParticipant2 := s.CreateUserWithToken(*participant2Response.AuthToken)
	participant2Me := Must(clientParticipant2.GetMe())
	s.Equal(participant2Response.User.Name, participant2Me.Name)
	s.T().Logf("Participant 2 can still authenticate")

	// Head can also remove participants
	MustSucceed(clientHead.DeleteUser(participant2Response.User.ID.String()))
	s.T().Logf("Head removed participant 2 (user account deleted)")
}
