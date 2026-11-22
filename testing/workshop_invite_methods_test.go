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

// An individual (in no organisation) takes part in a workshop through an
// email invite and keeps their role: taking part makes no one part of the
// organisation.
func (s *WorkshopInviteMethodsTestSuite) TestEmailInviteIndividualEntersWorkshopMode() {
	head, instID, wsID := s.workshopSetup("email-ind")

	individual := s.CreateUser("email-ind-user")
	me := Must(individual.GetMe())
	s.Equal("individual", me.Role.Role, "should be individual")

	// Head invites individual by email
	invite := Must(head.InviteToWorkshopByEmail(wsID, individual.Email))

	// Individual accepts the invite
	Must(individual.AcceptInvite(invite.Id.String()))

	// Should enter workshop mode without role change
	me = Must(individual.GetMe())
	s.Equal("individual", me.Role.Role, "role should stay individual")
	for _, m := range Must(head.GetInstitutionMembers(instID)) {
		s.NotEqual(individual.ID, m.Id.String(), "taking part makes no one part of the organisation")
	}
	s.Require().NotNil(me.Role.Workshop, "should have active workshop")
	s.Equal(wsID, me.Role.Workshop.Id.String(), "should be in the right workshop")
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
