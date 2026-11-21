package testing

import (
	"testing"

	"cgl/testing/testutil"

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
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds API key, shares with org
	keyShare := Must(head.AddApiKey("mock-ind-key-"+prefix, prefix+" Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))

	// Head creates workshop and sets API key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), prefix+" Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	// Enable showOtherParticipantsGames so individuals can see head's games
	Must(head.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:                       prefix + " Workshop",
		Active:                     true,
		ShowOtherParticipantsGames: true,
	}))

	// Head enters workshop mode and uploads a game
	Must(head.SetActiveWorkshop(&wsIDStr))
	game := Must(head.UploadGame("alien-first-contact"))
	gameIDStr := game.Id.String()

	// Create a workshop invite for individuals to join
	invite := Must(head.CreateWorkshopInvite(wsIDStr))
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
	s.Equal("individual", me.Role.Role, "new user should have individual role")
	s.Nil(me.Role.Institution, "individual should not belong to any institution")

	// Individual accepts the workshop invite token
	err := individual.AcceptWorkshopInviteByToken(inviteToken)
	s.NoError(err, "individual should be able to accept workshop invite")

	// Verify they are now in workshop mode
	me = Must(individual.GetMe())
	s.Equal("individual", me.Role.Role, "role should still be individual")
	s.Require().NotNil(me.Role.Workshop, "individual should have active workshop")
	s.Equal(wsIDStr, me.Role.Workshop.Id.String(), "active workshop should match")
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
	_, err := individual.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:                       "Hacked Workshop",
		Active:                     true,
		ShowPublicGames:            true,
		ShowOtherParticipantsGames: true,
		IsPaused:                   true,
	})
	s.Forbidden(err, "individual should not be able to update workshop settings")
}

// TestIndividualCannotTogglePause verifies that an individual in workshop mode
// cannot pause or unpause the workshop.
func (s *WorkshopIndividualTestSuite) TestIndividualCannotTogglePause() {
	head, wsIDStr, _, inviteToken := s.workshopWithKey("ind-pause")

	// Individual joins workshop
	individual := s.CreateUser("ind-pause-user")
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))

	// Head pauses the workshop
	ws := Must(head.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:                       "Paused Workshop",
		Active:                     true,
		ShowOtherParticipantsGames: true,
		IsPaused:                   true,
	}))
	s.True(ws.IsPaused, "head should be able to pause")

	// Individual tries to unpause — should fail
	_, err := individual.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:                       "Paused Workshop",
		Active:                     true,
		ShowOtherParticipantsGames: true,
		IsPaused:                   false,
	})
	s.Forbidden(err, "individual should not be able to unpause the workshop")

	// Verify workshop is still paused
	ws = Must(head.GetWorkshop(wsIDStr))
	s.True(ws.IsPaused, "workshop should still be paused after individual's failed attempt")
}

// TestGuestCannotUpdateSettings verifies that a Guest in a workshop
// also cannot update workshop settings.
func (s *WorkshopIndividualTestSuite) TestGuestCannotUpdateSettings() {
	_, wsIDStr, _, inviteToken := s.workshopWithKey("part-settings")

	// Create guest via anonymous invite
	guest := Must(s.JoinWorkshop(inviteToken))

	// Guest should NOT be able to update workshop settings
	_, err := guest.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:                       "Hacked Workshop",
		Active:                     true,
		ShowPublicGames:            true,
		ShowOtherParticipantsGames: true,
		IsPaused:                   true,
	})
	s.Forbidden(err, "guest should not be able to update workshop settings")
}

// TestIndividualRolePreservedAfterJoining verifies that an individual's role is NOT
// overwritten to guest when joining a workshop via invite token.
func (s *WorkshopIndividualTestSuite) TestIndividualRolePreservedAfterJoining() {
	_, _, _, inviteToken := s.workshopWithKey("ind-role-keep")

	individual := s.CreateUser("ind-role-keeper")
	me := Must(individual.GetMe())
	s.Equal("individual", me.Role.Role, "should start as individual")

	// Join workshop via token
	MustSucceed(individual.AcceptWorkshopInviteByToken(inviteToken))

	// Role must still be individual (not guest)
	me = Must(individual.GetMe())
	s.Equal("individual", me.Role.Role, "role must remain individual after joining workshop")
	s.Require().NotNil(me.Role.Workshop, "should have active workshop set")
	s.T().Logf("Individual role preserved: %s (workshop: %s)", me.Role.Role, me.Role.Workshop.Id)
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
	s.Equal("individual", me.Role.Role, "role should still be individual after leaving")
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
	s.Equal("individual", me.Role.Role, "role should still be individual after rejoin")
	s.Require().NotNil(me.Role.Workshop, "should be in workshop after rejoin")
	s.Equal(wsIDStr, me.Role.Workshop.Id.String(), "should be in the same workshop")
	s.T().Logf("Individual successfully rejoined workshop")
}

// TestNoInviteAsIndividual: an organisation invites as head or staff only.
// An individual is in no organisation (spec, Roles); one an organisation
// invites becomes head or staff by accepting.
func (s *WorkshopIndividualTestSuite) TestNoInviteAsIndividual() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Individual Invite Org"))
	target := s.CreateUser("ind-target")

	_, err := admin.InviteToInstitution(inst.Id.String(), "individual", target.ID)
	s.Error(err, "nobody is invited into an organisation as individual")
	me := Must(target.GetMe())
	s.Equal("individual", me.Role.Role)
	s.Nil(me.Role.Institution, "an individual is in no organisation")
}

// TestTargetedInviteWithHeadStaffStillWorks verifies that targeted institution
// invites with head/staff roles still work correctly.
func (s *WorkshopIndividualTestSuite) TestTargetedInviteWithHeadStaffStillWorks() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Head Staff Invite Org"))

	// Head invite should work
	headTarget := s.CreateUser("head-target")
	invite, err := admin.InviteToInstitution(inst.Id.String(), "head", headTarget.ID)
	s.NoError(err, "targeted invite with 'head' role should work")
	s.NotEmpty(invite.Id)
	s.T().Logf("Head invite created: %s", invite.Id)

	// Staff invite should work
	staffTarget := s.CreateUser("staff-target")
	invite, err = admin.InviteToInstitution(inst.Id.String(), "staff", staffTarget.ID)
	s.NoError(err, "targeted invite with 'staff' role should work")
	s.NotEmpty(invite.Id)
	s.T().Logf("Staff invite created: %s", invite.Id)
}

// containsGameByID checks if a game list contains a game with the given ID string.
func containsGameByID(games []testutil.Game, gameID string) bool {
	for _, g := range games {
		if g.Id.String() == gameID {
			return true
		}
	}
	return false
}
