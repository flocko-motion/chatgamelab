package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WorkshopSettingsPermissionsTestSuite tests that only head/staff can
// update workshop settings. Participants and individuals must be denied.
type WorkshopSettingsPermissionsTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopSettingsPermissionsSuite(t *testing.T) {
	s := &WorkshopSettingsPermissionsTestSuite{}
	s.SuiteName = "Workshop Settings Permissions Tests"
	suite.Run(t, s)
}

// setupWorkshop creates an institution, head, workshop, and a participant.
// Returns (head, participant, workshopID).
func (s *WorkshopSettingsPermissionsTestSuite) setupWorkshop(prefix string) (*testutil.UserClient, *testutil.UserClient, string) {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	workshop := Must(head.CreateWorkshop(inst.ID.String(), prefix+" Workshop"))
	wsIDStr := workshop.ID.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp, err := s.AcceptWorkshopInviteAnonymously(*invite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*resp.AuthToken)

	s.T().Logf("Setup: head=%s, participant=%s, ws=%s", head.Name, participant.Name, wsIDStr)
	return head, participant, wsIDStr
}

// workshopUpdatePayload returns a valid update payload with the given isPaused value.
func workshopUpdatePayload(isPaused bool) map[string]interface{} {
	return map[string]interface{}{
		"name":                       "Updated Workshop",
		"active":                     true,
		"public":                     false,
		"showPublicGames":            false,
		"showOtherParticipantsGames": true,
		"designEditingEnabled":       false,
		"isPaused":                   isPaused,
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

// TestParticipantCannotUpdateWorkshopSettings verifies that a participant
// cannot update any workshop settings.
func (s *WorkshopSettingsPermissionsTestSuite) TestParticipantCannotUpdateWorkshopSettings() {
	_, participant, wsIDStr := s.setupWorkshop("perm-part")

	// Participant should NOT be able to update workshop settings
	_, err := participant.UpdateWorkshop(wsIDStr, workshopUpdatePayload(true))
	s.Error(err, "participant should not be able to update workshop settings")
	s.T().Logf("Correctly denied participant: %v", err)
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
	s.Equal(obj.RoleIndividual, me.Role.Role, "user should have individual role")
	s.Require().NotNil(me.Role.Workshop, "individual should have active workshop")
	s.T().Logf("Individual %s entered workshop %s", individual.Name, wsIDStr)

	// Individual should NOT be able to update workshop settings
	_, err = individual.UpdateWorkshop(wsIDStr, workshopUpdatePayload(true))
	s.Error(err, "individual in workshop mode should not be able to update workshop settings")
	s.T().Logf("Correctly denied individual: %v", err)
}

// TestParticipantCannotTogglePause verifies specifically that a participant
// cannot toggle the isPaused setting even if they know the workshop ID.
func (s *WorkshopSettingsPermissionsTestSuite) TestParticipantCannotTogglePause() {
	head, participant, wsIDStr := s.setupWorkshop("perm-pause")

	// Head pauses the workshop
	ws := Must(head.UpdateWorkshop(wsIDStr, workshopUpdatePayload(true)))
	s.True(ws.IsPaused, "head should be able to pause")

	// Participant tries to unpause — should fail
	_, err := participant.UpdateWorkshop(wsIDStr, workshopUpdatePayload(false))
	s.Error(err, "participant should not be able to unpause the workshop")
	s.T().Logf("Correctly denied participant unpause: %v", err)

	// Verify workshop is still paused
	ws = Must(head.GetWorkshop(wsIDStr))
	s.True(ws.IsPaused, "workshop should still be paused after participant's failed attempt")
}
