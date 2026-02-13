package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// WorkshopIndividualTestSuite tests that individual users (not part of the org)
// can join workshops via invite link and use workshop features like seeing games
// and playing with the workshop API key, but cannot modify workshop settings.
type WorkshopIndividualTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopIndividualSuite(t *testing.T) {
	s := &WorkshopIndividualTestSuite{}
	s.SuiteName = "Workshop Individual Tests"
	suite.Run(t, s)
}

// workshopWithKey sets up an institution, head, workshop with an API key, and a game.
// Returns (head, workshopID, gameID, inviteToken).
func (s *WorkshopIndividualTestSuite) workshopWithKey(prefix string) (*testutil.UserClient, string, string, string) {
	admin := s.DevUser()

	// Create institution + head
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds API key, shares with org
	keyShare := Must(head.AddApiKey("mock-ind-key-"+prefix, prefix+" Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))

	// Head creates workshop and sets API key
	workshop := Must(head.CreateWorkshop(inst.ID.String(), prefix+" Workshop"))
	wsIDStr := workshop.ID.String()
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	// Enable showOtherParticipantsGames so individuals can see head's games
	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       prefix + " Workshop",
		"active":                     true,
		"public":                     false,
		"showPublicGames":            false,
		"showOtherParticipantsGames": true,
		"designEditingEnabled":       false,
		"isPaused":                   false,
	}))

	// Head enters workshop mode and uploads a game
	Must(head.SetActiveWorkshop(&wsIDStr))
	game := Must(head.UploadGame("alien-first-contact"))
	gameIDStr := game.ID.String()

	// Create a workshop invite for individuals to join
	invite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	s.Require().NotNil(invite.InviteToken)

	s.T().Logf("Setup: head=%s, ws=%s, game=%s", head.Name, wsIDStr, gameIDStr)
	return head, wsIDStr, gameIDStr, *invite.InviteToken
}

// TestIndividualJoinsWorkshopViaInviteToken verifies that an individual user
// (not part of the org) can join a workshop by accepting the invite token.
func (s *WorkshopIndividualTestSuite) TestIndividualJoinsWorkshopViaInviteToken() {
	_, wsIDStr, _, inviteToken := s.workshopWithKey("ind-join")

	// Create individual user (default role, no institution)
	individual := s.CreateUser("ind-joiner")
	me := Must(individual.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "new user should have individual role")
	s.Nil(me.Role.Institution, "individual should not belong to any institution")

	// Individual accepts the workshop invite token
	err := individual.AcceptWorkshopInviteByToken(inviteToken)
	s.NoError(err, "individual should be able to accept workshop invite")

	// Verify they are now in workshop mode
	me = Must(individual.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "role should still be individual")
	s.Require().NotNil(me.Role.Workshop, "individual should have active workshop")
	s.Equal(wsIDStr, me.Role.Workshop.ID.String(), "active workshop should match")
	s.T().Logf("Individual %s joined workshop %s via invite token", individual.Name, wsIDStr)
}

// TestIndividualCanSeeWorkshopGames verifies that an individual in workshop mode
// can see games created by the head in that workshop.
func (s *WorkshopIndividualTestSuite) TestIndividualCanSeeWorkshopGames() {
	_, _, gameIDStr, inviteToken := s.workshopWithKey("ind-games")

	// Individual joins workshop
	individual := s.CreateUser("ind-viewer")
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))

	// Individual should see the workshop game in their game list
	games := Must(individual.ListGames())
	s.True(containsGameByID(games, gameIDStr),
		"individual should see workshop games after joining")
	s.T().Logf("Individual sees %d games, including workshop game %s", len(games), gameIDStr)
}

// TestIndividualCanPlayGameWithWorkshopKey verifies that an individual in workshop mode
// can resolve the workshop API key for playing games.
func (s *WorkshopIndividualTestSuite) TestIndividualCanPlayGameWithWorkshopKey() {
	_, _, gameIDStr, inviteToken := s.workshopWithKey("ind-play")

	// Individual joins workshop
	individual := s.CreateUser("ind-player")
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))

	// Individual should see API key available (resolved via workshop key)
	available := Must(individual.GetApiKeyStatus(gameIDStr))
	s.True(available, "individual should have API key available via workshop key")
	s.T().Logf("Individual: API key available = %v", available)
}

// TestIndividualCannotUpdateWorkshopSettings verifies that an individual in workshop mode
// cannot update workshop settings (only head/staff can).
func (s *WorkshopIndividualTestSuite) TestIndividualCannotUpdateWorkshopSettings() {
	_, wsIDStr, _, inviteToken := s.workshopWithKey("ind-settings")

	// Individual joins workshop
	individual := s.CreateUser("ind-settings-user")
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))

	// Individual should NOT be able to update workshop settings
	_, err := individual.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       "Hacked Workshop",
		"active":                     true,
		"public":                     false,
		"showPublicGames":            true,
		"showOtherParticipantsGames": true,
		"designEditingEnabled":       true,
		"isPaused":                   true,
	})
	s.Error(err, "individual should not be able to update workshop settings")
	s.T().Logf("Correctly denied individual settings update: %v", err)
}

// TestIndividualCannotTogglePause verifies that an individual in workshop mode
// cannot pause or unpause the workshop.
func (s *WorkshopIndividualTestSuite) TestIndividualCannotTogglePause() {
	head, wsIDStr, _, inviteToken := s.workshopWithKey("ind-pause")

	// Individual joins workshop
	individual := s.CreateUser("ind-pause-user")
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))

	// Head pauses the workshop
	ws := Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       "Paused Workshop",
		"active":                     true,
		"public":                     false,
		"showPublicGames":            false,
		"showOtherParticipantsGames": true,
		"designEditingEnabled":       false,
		"isPaused":                   true,
	}))
	s.True(ws.IsPaused, "head should be able to pause")

	// Individual tries to unpause — should fail
	_, err := individual.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       "Paused Workshop",
		"active":                     true,
		"public":                     false,
		"showPublicGames":            false,
		"showOtherParticipantsGames": true,
		"designEditingEnabled":       false,
		"isPaused":                   false,
	})
	s.Error(err, "individual should not be able to unpause the workshop")
	s.T().Logf("Correctly denied individual unpause: %v", err)

	// Verify workshop is still paused
	ws = Must(head.GetWorkshop(wsIDStr))
	s.True(ws.IsPaused, "workshop should still be paused after individual's failed attempt")
}

// TestParticipantCannotUpdateSettings verifies that a participant (anonymous user)
// also cannot update workshop settings.
func (s *WorkshopIndividualTestSuite) TestParticipantCannotUpdateSettings() {
	_, wsIDStr, _, inviteToken := s.workshopWithKey("part-settings")

	// Create participant via anonymous invite
	resp, err := s.AcceptWorkshopInviteAnonymously(inviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*resp.AuthToken)

	// Participant should NOT be able to update workshop settings
	_, err = participant.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       "Hacked Workshop",
		"active":                     true,
		"public":                     false,
		"showPublicGames":            true,
		"showOtherParticipantsGames": true,
		"designEditingEnabled":       true,
		"isPaused":                   true,
	})
	s.Error(err, "participant should not be able to update workshop settings")
	s.T().Logf("Correctly denied participant settings update: %v", err)
}

// TestIndividualRolePreservedAfterJoining verifies that an individual's role is NOT
// overwritten to participant when joining a workshop via invite token.
func (s *WorkshopIndividualTestSuite) TestIndividualRolePreservedAfterJoining() {
	_, _, _, inviteToken := s.workshopWithKey("ind-role-keep")

	individual := s.CreateUser("ind-role-keeper")
	me := Must(individual.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "should start as individual")

	// Join workshop via token
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))

	// Role must still be individual (not participant)
	me = Must(individual.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "role must remain individual after joining workshop")
	s.Require().NotNil(me.Role.Workshop, "should have active workshop set")
	s.T().Logf("Individual role preserved: %s (workshop: %s)", me.Role.Role, me.Role.Workshop.ID)
}

// TestIndividualLeavesWorkshopReturnsToNormalView verifies that an individual
// can leave a workshop and return to their normal individual view.
func (s *WorkshopIndividualTestSuite) TestIndividualLeavesWorkshopReturnsToNormalView() {
	_, _, gameIDStr, inviteToken := s.workshopWithKey("ind-leave")

	individual := s.CreateUser("ind-leaver")

	// Join workshop
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))
	me := Must(individual.GetMe())
	s.Require().NotNil(me.Role.Workshop, "should be in workshop after joining")

	// Individual can see workshop games
	games := Must(individual.ListGames())
	s.True(containsGameByID(games, gameIDStr), "should see workshop games while in workshop")

	// Leave workshop (set active workshop to nil)
	Must(individual.SetActiveWorkshop(nil))

	// Verify back to normal individual view
	me = Must(individual.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "role should still be individual after leaving")
	s.Nil(me.Role.Workshop, "should have no active workshop after leaving")
	s.T().Logf("Individual left workshop, back to normal view")

	// Workshop games should no longer be visible
	games = Must(individual.ListGames())
	s.False(containsGameByID(games, gameIDStr), "should NOT see workshop games after leaving")
	s.T().Logf("Workshop games no longer visible after leaving")
}

// TestIndividualCanRejoinWorkshopAfterLeaving verifies that an individual can
// leave and rejoin a workshop without issues.
func (s *WorkshopIndividualTestSuite) TestIndividualCanRejoinWorkshopAfterLeaving() {
	_, wsIDStr, _, inviteToken := s.workshopWithKey("ind-rejoin")

	individual := s.CreateUser("ind-rejoiner")

	// Join → leave → rejoin
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))
	me := Must(individual.GetMe())
	s.Require().NotNil(me.Role.Workshop)

	Must(individual.SetActiveWorkshop(nil))
	me = Must(individual.GetMe())
	s.Nil(me.Role.Workshop, "should have no workshop after leaving")

	// Rejoin
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))
	me = Must(individual.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "role should still be individual after rejoin")
	s.Require().NotNil(me.Role.Workshop, "should be in workshop after rejoin")
	s.Equal(wsIDStr, me.Role.Workshop.ID.String(), "should be in the same workshop")
	s.T().Logf("Individual successfully rejoined workshop")
}

// TestTargetedInviteWithIndividualRoleRejected verifies that creating a targeted
// institution invite with role 'individual' is rejected.
func (s *WorkshopIndividualTestSuite) TestTargetedInviteWithIndividualRoleRejected() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("No Individual Invite Org"))
	target := s.CreateUser("ind-target")

	// Attempt to create targeted invite with 'individual' role — should fail
	_, err := admin.InviteToInstitution(inst.ID.String(), "individual", target.ID)
	s.Error(err, "targeted invite with 'individual' role should be rejected")
	s.T().Logf("Correctly rejected individual targeted invite: %v", err)
}

// TestTargetedInviteWithHeadStaffStillWorks verifies that targeted institution
// invites with head/staff roles still work correctly.
func (s *WorkshopIndividualTestSuite) TestTargetedInviteWithHeadStaffStillWorks() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Head Staff Invite Org"))

	// Head invite should work
	headTarget := s.CreateUser("head-target")
	invite, err := admin.InviteToInstitution(inst.ID.String(), "head", headTarget.ID)
	s.NoError(err, "targeted invite with 'head' role should work")
	s.NotEmpty(invite.ID)
	s.T().Logf("Head invite created: %s", invite.ID)

	// Staff invite should work
	staffTarget := s.CreateUser("staff-target")
	invite, err = admin.InviteToInstitution(inst.ID.String(), "staff", staffTarget.ID)
	s.NoError(err, "targeted invite with 'staff' role should work")
	s.NotEmpty(invite.ID)
	s.T().Logf("Staff invite created: %s", invite.ID)
}

// containsGameByID checks if a game list contains a game with the given ID string.
func containsGameByID(games []obj.Game, gameID string) bool {
	parsed, err := uuid.Parse(gameID)
	if err != nil {
		return false
	}
	for _, g := range games {
		if g.ID == parsed {
			return true
		}
	}
	return false
}
