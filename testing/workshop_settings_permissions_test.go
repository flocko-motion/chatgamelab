package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// WorkshopSettingsPermissionsTestSuite tests that only head/staff can
// update workshop settings. Guests and individuals must be denied.
type WorkshopSettingsPermissionsTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopSettingsPermissionsSuite(t *testing.T) {
	s := &WorkshopSettingsPermissionsTestSuite{}
	s.SuiteName = "Workshop Settings Permissions Tests"
	suite.Run(t, s)
}

// setupWorkshop creates an institution, head, workshop, and a guest.
// Returns (head, guest, workshopID).
func (s *WorkshopSettingsPermissionsTestSuite) setupWorkshop(prefix string) (*testutil.UserClient, *testutil.UserClient, string) {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), prefix+" Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))

	s.T().Logf("Setup: head=%s, guest=%s, ws=%s", head.Name, guest.Name, wsIDStr)
	return head, guest, wsIDStr
}

// workshopUpdatePayload is a valid set of settings with the given isPaused value.
func workshopUpdatePayload(isPaused bool) testutil.WorkshopUpdate {
	return testutil.WorkshopUpdate{
		Name:                       "Updated Workshop",
		Active:                     true,
		ShowOtherParticipantsGames: true,
		IsPaused:                   isPaused,
	}
}

// TestHeadCanUpdateWorkshopSettings verifies that head can update settings including isPaused.
func (s *WorkshopSettingsPermissionsTestSuite) TestHeadCanUpdateWorkshopSettings() {
	head, _, wsIDStr := s.setupWorkshop("perm-head")

	// Head should be able to pause
	ws := Must(head.UpdateWorkshop(wsIDStr, workshopUpdatePayload(true)))
	s.True(ws.IsPaused, "head should be able to pause the workshop")

	// Head should be able to unpause
	ws = Must(head.UpdateWorkshop(wsIDStr, workshopUpdatePayload(false)))
	s.False(ws.IsPaused, "head should be able to unpause the workshop")
}

// TestGuestCannotUpdateWorkshopSettings verifies that a guest
// cannot update any workshop settings.
func (s *WorkshopSettingsPermissionsTestSuite) TestGuestCannotUpdateWorkshopSettings() {
	_, guest, wsIDStr := s.setupWorkshop("perm-part")

	// Guest should NOT be able to update workshop settings
	_, err := guest.UpdateWorkshop(wsIDStr, workshopUpdatePayload(true))
	s.Forbidden(err, "guest should not be able to update workshop settings")
}

// TestIndividualCannotUpdateWorkshopSettings verifies that an individual user
// who has entered the workshop via workshop mode still cannot update settings.
func (s *WorkshopSettingsPermissionsTestSuite) TestIndividualCannotUpdateWorkshopSettings() {
	_, _, wsIDStr := s.setupWorkshop("perm-ind")

	// Create an individual user (default role for new users)
	individual := s.CreateUser("perm-individual")

	// Individual enters the workshop via workshop mode
	_, err := individual.SetActiveWorkshop(&wsIDStr)
	s.NoError(err, "individual should be able to enter workshop mode")

	// Verify they are in workshop mode
	me := Must(individual.GetMe())
	s.Require().NotNil(me.Role, "individual should have a role")
	s.Equal("individual", me.Role.Role, "user should have individual role")
	s.Require().NotNil(me.Role.Workshop, "individual should have active workshop")
	s.T().Logf("Individual %s entered workshop %s", individual.Name, wsIDStr)

	// Individual should NOT be able to update workshop settings
	_, err = individual.UpdateWorkshop(wsIDStr, workshopUpdatePayload(true))
	s.Forbidden(err, "individual in workshop mode should not be able to update workshop settings")
}

// TestGuestCannotTogglePause verifies specifically that a guest
// cannot toggle the isPaused setting even if they know the workshop ID.
func (s *WorkshopSettingsPermissionsTestSuite) TestGuestCannotTogglePause() {
	head, guest, wsIDStr := s.setupWorkshop("perm-pause")

	// Head pauses the workshop
	ws := Must(head.UpdateWorkshop(wsIDStr, workshopUpdatePayload(true)))
	s.True(ws.IsPaused, "head should be able to pause")

	// Guest tries to unpause — should fail
	_, err := guest.UpdateWorkshop(wsIDStr, workshopUpdatePayload(false))
	s.Forbidden(err, "guest should not be able to unpause the workshop")

	// Verify workshop is still paused
	ws = Must(head.GetWorkshop(wsIDStr))
	s.True(ws.IsPaused, "workshop should still be paused after guest's failed attempt")
}
