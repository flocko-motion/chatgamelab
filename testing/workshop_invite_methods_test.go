package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WorkshopInviteMethodsTestSuite tests the two new workshop invite methods:
// 1. Email invite — head/staff invites a registered user by email
// 2. Add org individual — head/staff directly adds an individual to the workshop
type WorkshopInviteMethodsTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopInviteMethodsSuite(t *testing.T) {
	s := &WorkshopInviteMethodsTestSuite{}
	s.SuiteName = "Workshop Invite Methods Tests"
	suite.Run(t, s)
}

// workshopSetup creates an institution, head, workshop with API key, and returns
// (head, institutionID, workshopID, inviteToken).
func (s *WorkshopInviteMethodsTestSuite) workshopSetup(prefix string) (*testutil.UserClient, string, string) {
	admin := s.DevUser()

	// Create institution + head
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds API key, shares with org
	keyShare := Must(head.AddApiKey("mock-inv-key-"+prefix, prefix+" Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))

	// Head creates workshop and sets API key
	workshop := Must(head.CreateWorkshop(inst.ID.String(), prefix+" Workshop"))
	wsIDStr := workshop.ID.String()
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	// Activate the workshop
	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       prefix + " Workshop",
		"active":                     true,
		"public":                     false,
		"showPublicGames":            false,
		"showOtherParticipantsGames": true,
		"designEditingEnabled":       false,
		"isPaused":                   false,
	}))

	s.T().Logf("Setup: head=%s, inst=%s, ws=%s", head.Name, inst.ID, wsIDStr)
	return head, inst.ID.String(), wsIDStr
}

// --- Email invite tests ---

func (s *WorkshopInviteMethodsTestSuite) TestHeadCanInviteByEmailToWorkshop() {
	head, _, wsID := s.workshopSetup("email-head")

	target := s.CreateUser("email-target")
	invite, err := head.InviteToWorkshopByEmail(wsID, target.Email)
	s.NoError(err, "head should be able to invite by email")
	s.NotEmpty(invite.ID, "invite should have an ID")
	s.T().Logf("Head invited %s by email, invite=%s", target.Email, invite.ID)

	// Target should see the pending invite
	incoming := Must(target.GetInvitesIncoming())
	found := false
	for _, inv := range incoming {
		if inv.ID == invite.ID {
			found = true
			break
		}
	}
	s.True(found, "target should see the pending workshop invite")
}

func (s *WorkshopInviteMethodsTestSuite) TestStaffCanInviteByEmailToWorkshop() {
	head, instID, wsID := s.workshopSetup("email-staff")
	admin := s.DevUser()

	// Create a staff user
	staff := s.CreateUser("email-staff-user")
	staffInvite := Must(admin.InviteToInstitution(instID, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Staff invites someone by email
	target := s.CreateUser("email-staff-target")
	invite, err := staff.InviteToWorkshopByEmail(wsID, target.Email)
	s.NoError(err, "staff should be able to invite by email")
	s.NotEmpty(invite.ID)
	s.T().Logf("Staff invited %s by email, invite=%s", target.Email, invite.ID)

	_ = head // used in setup
}

func (s *WorkshopInviteMethodsTestSuite) TestEmailInviteIndividualEntersWorkshopMode() {
	head, instID, wsID := s.workshopSetup("email-ind")
	admin := s.DevUser()

	// Create an individual in the org
	individual := s.CreateUser("email-ind-user")
	indInvite := Must(admin.InviteToInstitution(instID, "individual", individual.ID))
	Must(individual.AcceptInvite(indInvite.ID.String()))

	me := Must(individual.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "should be individual")

	// Head invites individual by email
	invite := Must(head.InviteToWorkshopByEmail(wsID, individual.Email))

	// Individual accepts the invite
	Must(individual.AcceptInvite(invite.ID.String()))

	// Should enter workshop mode without role change
	me = Must(individual.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "role should stay individual")
	s.Require().NotNil(me.Role.Workshop, "should have active workshop")
	s.Equal(wsID, me.Role.Workshop.ID.String(), "should be in the right workshop")
	s.T().Logf("Individual entered workshop mode via email invite")
}

func (s *WorkshopInviteMethodsTestSuite) TestEmailInviteNewUserBecomesParticipant() {
	head, _, wsID := s.workshopSetup("email-new")

	// Create a user with no role (fresh user)
	newUser := s.CreateUser("email-new-user")
	me := Must(newUser.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "new user should have individual role by default")

	// Head invites new user by email
	invite := Must(head.InviteToWorkshopByEmail(wsID, newUser.Email))

	// New user accepts → should enter workshop mode (individual stays individual)
	Must(newUser.AcceptInvite(invite.ID.String()))
	me = Must(newUser.GetMe())
	s.Require().NotNil(me.Role.Workshop, "should have active workshop after accepting")
	s.Equal(wsID, me.Role.Workshop.ID.String(), "should be in the right workshop")
	s.T().Logf("New user accepted workshop email invite, role=%s", me.Role.Role)
}

func (s *WorkshopInviteMethodsTestSuite) TestEmailInviteDuplicateRejected() {
	head, _, wsID := s.workshopSetup("email-dup")

	target := s.CreateUser("email-dup-target")

	// First invite should succeed
	_, err := head.InviteToWorkshopByEmail(wsID, target.Email)
	s.NoError(err)

	// Duplicate invite should fail
	_, err = head.InviteToWorkshopByEmail(wsID, target.Email)
	s.Error(err, "duplicate workshop email invite should be rejected")
	s.T().Logf("Correctly rejected duplicate invite: %v", err)
}

func (s *WorkshopInviteMethodsTestSuite) TestParticipantCannotInviteByEmail() {
	head, _, wsID := s.workshopSetup("email-perm")

	// Create a workshop invite link and have someone join as participant
	invite := Must(head.CreateWorkshopInvite(wsID, string(obj.RoleParticipant)))
	s.Require().NotNil(invite.InviteToken)

	resp, err := s.AcceptWorkshopInviteAnonymously(*invite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*resp.AuthToken)

	// Participant tries to invite by email — should fail
	target := s.CreateUser("email-perm-target")
	_, err = participant.InviteToWorkshopByEmail(wsID, target.Email)
	s.Error(err, "participant should not be able to invite by email")
	s.T().Logf("Correctly denied participant email invite: %v", err)
}

// --- Add org individual tests ---

func (s *WorkshopInviteMethodsTestSuite) TestHeadCanAddIndividualToWorkshop() {
	head, instID, wsID := s.workshopSetup("add-head")
	admin := s.DevUser()

	// Create an individual in the org
	individual := s.CreateUser("add-ind-user")
	indInvite := Must(admin.InviteToInstitution(instID, "individual", individual.ID))
	Must(individual.AcceptInvite(indInvite.ID.String()))

	// Head adds individual directly
	err := head.AddMemberToWorkshop(wsID, individual.ID)
	s.NoError(err, "head should be able to add individual to workshop")

	// Individual should now be in workshop mode
	me := Must(individual.GetMe())
	s.Equal(obj.RoleIndividual, me.Role.Role, "role should stay individual")
	s.Require().NotNil(me.Role.Workshop, "should have active workshop")
	s.Equal(wsID, me.Role.Workshop.ID.String(), "should be in the right workshop")
	s.T().Logf("Head added individual to workshop directly")
}

func (s *WorkshopInviteMethodsTestSuite) TestStaffCanAddIndividualToWorkshop() {
	_, instID, wsID := s.workshopSetup("add-staff")
	admin := s.DevUser()

	// Create a staff user
	staff := s.CreateUser("add-staff-user")
	staffInvite := Must(admin.InviteToInstitution(instID, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Create an individual in the org
	individual := s.CreateUser("add-staff-ind")
	indInvite := Must(admin.InviteToInstitution(instID, "individual", individual.ID))
	Must(individual.AcceptInvite(indInvite.ID.String()))

	// Staff adds individual
	err := staff.AddMemberToWorkshop(wsID, individual.ID)
	s.NoError(err, "staff should be able to add individual to workshop")

	me := Must(individual.GetMe())
	s.Require().NotNil(me.Role.Workshop, "should have active workshop")
	s.Equal(wsID, me.Role.Workshop.ID.String())
	s.T().Logf("Staff added individual to workshop")
}

func (s *WorkshopInviteMethodsTestSuite) TestCannotAddNonIndividual() {
	head, instID, wsID := s.workshopSetup("add-nonind")
	admin := s.DevUser()

	// Create a staff user in the org
	staff := s.CreateUser("add-nonind-staff")
	staffInvite := Must(admin.InviteToInstitution(instID, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Head tries to add staff via AddMemberToWorkshop — should fail
	err := head.AddMemberToWorkshop(wsID, staff.ID)
	s.Error(err, "should not be able to add non-individual to workshop")
	s.T().Logf("Correctly denied adding non-individual: %v", err)
}

func (s *WorkshopInviteMethodsTestSuite) TestCannotAddUserFromDifferentOrg() {
	head, _, wsID := s.workshopSetup("add-difforg")
	admin := s.DevUser()

	// Create a different org with an individual
	otherInst := Must(admin.CreateInstitution("Other Org"))
	otherInd := s.CreateUser("add-other-ind")
	otherInvite := Must(admin.InviteToInstitution(otherInst.ID.String(), "individual", otherInd.ID))
	Must(otherInd.AcceptInvite(otherInvite.ID.String()))

	// Head tries to add individual from different org — should fail
	err := head.AddMemberToWorkshop(wsID, otherInd.ID)
	s.Error(err, "should not be able to add user from different org")
	s.T().Logf("Correctly denied adding user from different org: %v", err)
}

func (s *WorkshopInviteMethodsTestSuite) TestParticipantCannotAddMember() {
	head, _, wsID := s.workshopSetup("add-perm")

	// Create a workshop invite link and have someone join as participant
	invite := Must(head.CreateWorkshopInvite(wsID, string(obj.RoleParticipant)))
	s.Require().NotNil(invite.InviteToken)

	resp, err := s.AcceptWorkshopInviteAnonymously(*invite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*resp.AuthToken)

	// Participant tries to add a member — should fail
	target := s.CreateUser("add-perm-target")
	err = participant.AddMemberToWorkshop(wsID, target.ID)
	s.Error(err, "participant should not be able to add members")
	s.T().Logf("Correctly denied participant add member: %v", err)
}

func (s *WorkshopInviteMethodsTestSuite) TestAddedIndividualCanLeaveWorkshop() {
	head, instID, wsID := s.workshopSetup("add-leave")
	admin := s.DevUser()

	// Create an individual in the org
	individual := s.CreateUser("add-leave-ind")
	indInvite := Must(admin.InviteToInstitution(instID, "individual", individual.ID))
	Must(individual.AcceptInvite(indInvite.ID.String()))

	// Head adds individual
	MustSucceed(head.AddMemberToWorkshop(wsID, individual.ID))

	me := Must(individual.GetMe())
	s.Require().NotNil(me.Role.Workshop, "should be in workshop after being added")

	// Individual leaves workshop
	Must(individual.SetActiveWorkshop(nil))
	me = Must(individual.GetMe())
	s.Nil(me.Role.Workshop, "should have no workshop after leaving")
	s.Equal(obj.RoleIndividual, me.Role.Role, "role should still be individual")
	s.T().Logf("Individual left workshop after being directly added")
}
