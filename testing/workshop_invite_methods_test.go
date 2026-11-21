package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// WorkshopInviteMethodsTestSuite tests the two ways to bring someone into a workshop besides the invite link:
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
// (head, institutionID, workshopID).
func (s *WorkshopInviteMethodsTestSuite) workshopSetup(prefix string) (*testutil.UserClient, string, string) {
	admin := s.DevUser()

	// Create institution + head
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Head adds API key, shares with org
	keyShare := Must(head.AddApiKey("mock-inv-key-"+prefix, prefix+" Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))

	// Head creates workshop and sets API key
	workshop := Must(head.CreateWorkshop(inst.Id.String(), prefix+" Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	// Activate the workshop
	Must(head.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:                       prefix + " Workshop",
		Active:                     true,
		ShowOtherParticipantsGames: true,
	}))

	s.T().Logf("Setup: head=%s, inst=%s, ws=%s", head.Name, inst.Id, wsIDStr)
	return head, inst.Id.String(), wsIDStr
}

// --- Email invite tests ---

func (s *WorkshopInviteMethodsTestSuite) TestHeadCanInviteByEmailToWorkshop() {
	head, _, wsID := s.workshopSetup("email-head")

	target := s.CreateUser("email-target")
	invite, err := head.InviteToWorkshopByEmail(wsID, target.Email)
	s.NoError(err, "head should be able to invite by email")
	s.NotEmpty(invite.Id, "invite should have an ID")
	s.T().Logf("Head invited %s by email, invite=%s", target.Email, invite.Id)

	// Target should see the pending invite
	incoming := Must(target.GetInvitesIncoming())
	found := false
	for _, inv := range incoming {
		if inv.Id == invite.Id {
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
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	// Staff invites someone by email
	target := s.CreateUser("email-staff-target")
	invite, err := staff.InviteToWorkshopByEmail(wsID, target.Email)
	s.NoError(err, "staff should be able to invite by email")
	s.NotEmpty(invite.Id)
	s.T().Logf("Staff invited %s by email, invite=%s", target.Email, invite.Id)

	_ = head // used in setup
}

func (s *WorkshopInviteMethodsTestSuite) TestEmailInviteIndividualEntersWorkshopMode() {
	head, instID, wsID := s.workshopSetup("email-ind")
	admin := s.DevUser()

	// Create an individual in the org
	individual := s.CreateUser("email-ind-user")
	indInvite := Must(admin.InviteToInstitution(instID, "individual", individual.ID))
	Must(individual.AcceptInvite(indInvite.Id.String()))

	me := Must(individual.GetMe())
	s.Equal("individual", me.Role.Role, "should be individual")

	// Head invites individual by email
	invite := Must(head.InviteToWorkshopByEmail(wsID, individual.Email))

	// Individual accepts the invite
	Must(individual.AcceptInvite(invite.Id.String()))

	// Should enter workshop mode without role change
	me = Must(individual.GetMe())
	s.Equal("individual", me.Role.Role, "role should stay individual")
	s.Require().NotNil(me.Role.Workshop, "should have active workshop")
	s.Equal(wsID, me.Role.Workshop.Id.String(), "should be in the right workshop")
	s.T().Logf("Individual entered workshop mode via email invite")
}

func (s *WorkshopInviteMethodsTestSuite) TestEmailInviteNewUserKeepsIndividualRole() {
	head, _, wsID := s.workshopSetup("email-new")

	// A fresh user
	newUser := s.CreateUser("email-new-user")
	me := Must(newUser.GetMe())
	s.Equal("individual", me.Role.Role, "new user should have individual role by default")

	// Head invites new user by email
	invite := Must(head.InviteToWorkshopByEmail(wsID, newUser.Email))

	// New user accepts → should enter workshop mode (individual stays individual)
	Must(newUser.AcceptInvite(invite.Id.String()))
	me = Must(newUser.GetMe())
	s.Equal("individual", me.Role.Role, "role should stay individual")
	s.Require().NotNil(me.Role.Workshop, "should have active workshop after accepting")
	s.Equal(wsID, me.Role.Workshop.Id.String(), "should be in the right workshop")
	s.T().Logf("New user accepted workshop email invite, role=%s", me.Role.Role)
}

// TestEmailInviteStaffOfOtherOrgKeepsRole: staff of another organisation may
// be invited, but enters only its own organisation's workshops. Accepting is
// refused and the role stays.
func (s *WorkshopInviteMethodsTestSuite) TestEmailInviteStaffOfOtherOrgKeepsRole() {
	head, _, wsID := s.workshopSetup("email-otherstaff")
	admin := s.DevUser()

	otherInst := Must(admin.CreateInstitution("email-otherstaff Other Org"))
	staff := s.CreateUser("email-otherstaff-user")
	staffInvite := Must(admin.InviteToInstitution(otherInst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	invite := Must(head.InviteToWorkshopByEmail(wsID, staff.Email))
	_, err := staff.AcceptInvite(invite.Id.String())
	s.Forbidden(err, "staff of another organisation cannot enter the workshop")

	me := Must(staff.GetMe())
	s.Equal("staff", me.Role.Role, "role should stay staff")
	s.Require().NotNil(me.Role.Institution, "should stay in its organisation")
	s.Equal(otherInst.Id.String(), me.Role.Institution.Id.String())
	s.Nil(me.Role.Workshop, "should not be in the workshop")
}

// TestEmailInviteAdminKeepsRole: an admin never becomes a Guest by accepting
// a workshop invite; admins have no workshop mode, so accepting is refused.
func (s *WorkshopInviteMethodsTestSuite) TestEmailInviteAdminKeepsRole() {
	head, _, wsID := s.workshopSetup("email-admin")

	target := s.CreateUser("email-admin-user")
	s.Role(target, "admin")

	invite := Must(head.InviteToWorkshopByEmail(wsID, target.Email))
	_, err := target.AcceptInvite(invite.Id.String())
	s.Forbidden(err, "an admin cannot enter a workshop")

	me := Must(target.GetMe())
	s.Equal("admin", me.Role.Role, "role should stay admin")
	s.Nil(me.Role.Workshop, "should not be in the workshop")
}

func (s *WorkshopInviteMethodsTestSuite) TestEmailInviteDuplicateRejected() {
	head, _, wsID := s.workshopSetup("email-dup")

	target := s.CreateUser("email-dup-target")

	// First invite should succeed
	_, err := head.InviteToWorkshopByEmail(wsID, target.Email)
	s.NoError(err)

	// Duplicate invite should fail
	_, err = head.InviteToWorkshopByEmail(wsID, target.Email)
	s.Conflict(err, "duplicate workshop email invite should be rejected")
}

func (s *WorkshopInviteMethodsTestSuite) TestGuestCannotInviteByEmail() {
	head, _, wsID := s.workshopSetup("email-perm")

	// Create a workshop invite link and have someone join as guest
	invite := Must(head.CreateWorkshopInvite(wsID))
	s.Require().NotNil(invite.InviteToken)

	guest := Must(s.JoinWorkshop(*invite.InviteToken))

	// Guest tries to invite by email — should fail
	target := s.CreateUser("email-perm-target")
	_, err := guest.InviteToWorkshopByEmail(wsID, target.Email)
	s.Forbidden(err, "guest should not be able to invite by email")
}

// --- Add org individual tests ---

func (s *WorkshopInviteMethodsTestSuite) TestHeadCanAddIndividualToWorkshop() {
	head, instID, wsID := s.workshopSetup("add-head")
	admin := s.DevUser()

	// Create an individual in the org
	individual := s.CreateUser("add-ind-user")
	indInvite := Must(admin.InviteToInstitution(instID, "individual", individual.ID))
	Must(individual.AcceptInvite(indInvite.Id.String()))

	// Head adds individual directly
	err := head.AddMemberToWorkshop(wsID, individual.ID)
	s.NoError(err, "head should be able to add individual to workshop")

	// Individual should now be in workshop mode
	me := Must(individual.GetMe())
	s.Equal("individual", me.Role.Role, "role should stay individual")
	s.Require().NotNil(me.Role.Workshop, "should have active workshop")
	s.Equal(wsID, me.Role.Workshop.Id.String(), "should be in the right workshop")
	s.T().Logf("Head added individual to workshop directly")
}

func (s *WorkshopInviteMethodsTestSuite) TestStaffCanAddIndividualToWorkshop() {
	_, instID, wsID := s.workshopSetup("add-staff")
	admin := s.DevUser()

	// Create a staff user
	staff := s.CreateUser("add-staff-user")
	staffInvite := Must(admin.InviteToInstitution(instID, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	// Create an individual in the org
	individual := s.CreateUser("add-staff-ind")
	indInvite := Must(admin.InviteToInstitution(instID, "individual", individual.ID))
	Must(individual.AcceptInvite(indInvite.Id.String()))

	// Staff adds individual
	err := staff.AddMemberToWorkshop(wsID, individual.ID)
	s.NoError(err, "staff should be able to add individual to workshop")

	me := Must(individual.GetMe())
	s.Require().NotNil(me.Role.Workshop, "should have active workshop")
	s.Equal(wsID, me.Role.Workshop.Id.String())
	s.T().Logf("Staff added individual to workshop")
}

func (s *WorkshopInviteMethodsTestSuite) TestCannotAddNonIndividual() {
	head, instID, wsID := s.workshopSetup("add-nonind")
	admin := s.DevUser()

	// Create a staff user in the org
	staff := s.CreateUser("add-nonind-staff")
	staffInvite := Must(admin.InviteToInstitution(instID, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	// Head tries to add staff via AddMemberToWorkshop — should fail
	err := head.AddMemberToWorkshop(wsID, staff.ID)
	s.Invalid(err, "should not be able to add non-individual to workshop")
}

func (s *WorkshopInviteMethodsTestSuite) TestCannotAddUserFromDifferentOrg() {
	head, _, wsID := s.workshopSetup("add-difforg")
	admin := s.DevUser()

	// Create a different org with an individual
	otherInst := Must(admin.CreateInstitution("Other Org"))
	otherInd := s.CreateUser("add-other-ind")
	otherInvite := Must(admin.InviteToInstitution(otherInst.Id.String(), "individual", otherInd.ID))
	Must(otherInd.AcceptInvite(otherInvite.Id.String()))

	// Head tries to add individual from different org — should fail
	err := head.AddMemberToWorkshop(wsID, otherInd.ID)
	s.NotFound(err, "should not be able to add user from different org: for the head, they do not exist")
}

func (s *WorkshopInviteMethodsTestSuite) TestGuestCannotAddMember() {
	head, _, wsID := s.workshopSetup("add-perm")

	// Create a workshop invite link and have someone join as guest
	invite := Must(head.CreateWorkshopInvite(wsID))
	s.Require().NotNil(invite.InviteToken)

	guest := Must(s.JoinWorkshop(*invite.InviteToken))

	// Guest tries to add a member — should fail
	target := s.CreateUser("add-perm-target")
	err := guest.AddMemberToWorkshop(wsID, target.ID)
	s.Forbidden(err, "guest should not be able to add members")
}

func (s *WorkshopInviteMethodsTestSuite) TestAddedIndividualCanLeaveWorkshop() {
	head, instID, wsID := s.workshopSetup("add-leave")
	admin := s.DevUser()

	// Create an individual in the org
	individual := s.CreateUser("add-leave-ind")
	indInvite := Must(admin.InviteToInstitution(instID, "individual", individual.ID))
	Must(individual.AcceptInvite(indInvite.Id.String()))

	// Head adds individual
	MustSucceed(head.AddMemberToWorkshop(wsID, individual.ID))

	me := Must(individual.GetMe())
	s.Require().NotNil(me.Role.Workshop, "should be in workshop after being added")

	// Individual leaves workshop
	Must(individual.SetActiveWorkshop(nil))
	me = Must(individual.GetMe())
	s.Nil(me.Role.Workshop, "should have no workshop after leaving")
	s.Equal("individual", me.Role.Role, "role should still be individual")
	s.T().Logf("Individual left workshop after being directly added")
}
