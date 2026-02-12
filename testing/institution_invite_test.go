package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// InstitutionInviteTestSuite tests institution invite permissions.
type InstitutionInviteTestSuite struct {
	testutil.BaseSuite
}

func TestInstitutionInviteSuite(t *testing.T) {
	s := &InstitutionInviteTestSuite{}
	s.SuiteName = "Institution Invite Tests"
	suite.Run(t, s)
}

// TestHeadCanInviteToInstitution tests that a head can invite users to their institution.
func (s *InstitutionInviteTestSuite) TestHeadCanInviteToInstitution() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Head Invite Org"))
	instIDStr := inst.ID.String()

	head := s.CreateUser("hi-head")
	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	s.T().Logf("Head joined institution")

	// Head invites a new user as staff
	newUser := s.CreateUser("hi-new-staff")
	invite, err := head.InviteToInstitution(instIDStr, "staff", newUser.ID)
	s.NoError(err, "head should be able to invite to institution")
	s.NotEmpty(invite.ID)
	s.T().Logf("Head invited new user as staff: invite %s", invite.ID)

	// New user accepts and becomes staff
	Must(newUser.AcceptInvite(invite.ID.String()))
	me := Must(newUser.GetMe())
	s.Equal("staff", string(me.Role.Role), "invited user should be staff")
	s.T().Logf("New user accepted invite and is now staff")
}

// TestSecondHeadCanInvite tests that a second head (not the original) can also invite.
func (s *InstitutionInviteTestSuite) TestSecondHeadCanInvite() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Second Head Org"))
	instIDStr := inst.ID.String()

	head1 := s.CreateUser("sh-head1")
	head2 := s.CreateUser("sh-head2")

	h1Invite := Must(admin.InviteToInstitution(instIDStr, "head", head1.ID))
	Must(head1.AcceptInvite(h1Invite.ID.String()))
	h2Invite := Must(admin.InviteToInstitution(instIDStr, "head", head2.ID))
	Must(head2.AcceptInvite(h2Invite.ID.String()))
	s.T().Logf("Two heads in institution")

	// Second head invites a new user as staff
	newUser := s.CreateUser("sh-new-staff")
	invite, err := head2.InviteToInstitution(instIDStr, "staff", newUser.ID)
	s.NoError(err, "second head should be able to invite")
	s.NotEmpty(invite.ID)
	s.T().Logf("Second head invited new user as staff")

	Must(newUser.AcceptInvite(invite.ID.String()))
	me := Must(newUser.GetMe())
	s.Equal("staff", string(me.Role.Role))
	s.T().Logf("New user is staff via second head's invite")
}

// TestStaffCanInvite tests that a staff member can also invite users to the institution.
func (s *InstitutionInviteTestSuite) TestStaffCanInvite() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Staff Invite Org"))
	instIDStr := inst.ID.String()

	head := s.CreateUser("si-head")
	staff := s.CreateUser("si-staff")

	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(instIDStr, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.T().Logf("Head and staff in institution")

	// Staff invites a new user as staff
	newUser := s.CreateUser("si-new-staff")
	invite, err := staff.InviteToInstitution(instIDStr, "staff", newUser.ID)
	s.NoError(err, "staff should be able to invite to institution")
	s.NotEmpty(invite.ID)
	s.T().Logf("Staff invited new user as staff")

	Must(newUser.AcceptInvite(invite.ID.String()))
	me := Must(newUser.GetMe())
	s.Equal("staff", string(me.Role.Role))
	s.T().Logf("New user is staff via staff's invite")
}

// TestParticipantCannotInviteToInstitution tests that a participant cannot create institution invites.
func (s *InstitutionInviteTestSuite) TestParticipantCannotInviteToInstitution() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Part No Invite Org"))
	instIDStr := inst.ID.String()

	head := s.CreateUser("pni-head")
	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Create workshop + participant
	workshop := Must(head.CreateWorkshop(instIDStr, "PNI Workshop"))
	wsInvite := Must(head.CreateWorkshopInvite(workshop.ID.String(), string(obj.RoleParticipant)))
	resp, err := s.AcceptWorkshopInviteAnonymously(*wsInvite.InviteToken)
	s.NoError(err)
	participant := s.CreateUserWithToken(*resp.AuthToken)
	s.T().Logf("Participant joined: %s", participant.Name)

	// Participant tries to invite — should fail
	newUser := s.CreateUser("pni-target")
	_, err = participant.InviteToInstitution(instIDStr, "staff", newUser.ID)
	s.Error(err, "participant should not be able to invite to institution")
	s.T().Logf("Participant correctly rejected: %v", err)
}

// TestIndividualCannotInviteToInstitution tests that an individual user cannot invite to any institution.
func (s *InstitutionInviteTestSuite) TestIndividualCannotInviteToInstitution() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Ind No Invite Org"))
	individual := s.CreateUser("ini-individual")

	// Individual tries to invite — should fail
	newUser := s.CreateUser("ini-target")
	_, err := individual.InviteToInstitution(inst.ID.String(), "staff", newUser.ID)
	s.Error(err, "individual should not be able to invite to institution")
	s.T().Logf("Individual correctly rejected: %v", err)
}

// TestHeadCanInviteHead tests that a head can invite another user as head.
func (s *InstitutionInviteTestSuite) TestHeadCanInviteHead() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Head Invite Head Org"))
	instIDStr := inst.ID.String()

	head := s.CreateUser("hih-head")
	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head invites a new user as head
	newHead := s.CreateUser("hih-new-head")
	invite, err := head.InviteToInstitution(instIDStr, "head", newHead.ID)
	s.NoError(err, "head should be able to invite another head")
	s.NotEmpty(invite.ID)

	Must(newHead.AcceptInvite(invite.ID.String()))
	me := Must(newHead.GetMe())
	s.Equal("head", string(me.Role.Role), "invited user should be head")
	s.T().Logf("Head successfully invited another head")
}

// TestStaffCannotInviteHead tests that a staff member cannot invite someone as head.
func (s *InstitutionInviteTestSuite) TestStaffCannotInviteHead() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Staff No Head Org"))
	instIDStr := inst.ID.String()

	head := s.CreateUser("snh-head")
	staff := s.CreateUser("snh-staff")

	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(instIDStr, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Staff tries to invite someone as head — should fail
	target := s.CreateUser("snh-target")
	_, err := staff.InviteToInstitution(instIDStr, "head", target.ID)
	s.Error(err, "staff should not be able to invite as head")
	s.T().Logf("Staff correctly rejected from inviting head: %v", err)

	// But staff can still invite as staff
	invite, err := staff.InviteToInstitution(instIDStr, "staff", target.ID)
	s.NoError(err, "staff should be able to invite as staff")
	s.NotEmpty(invite.ID)
	s.T().Logf("Staff can invite as staff: %s", invite.ID)
}

// TestHeadAndStaffCanBothInviteNormalUsers tests that both head and staff
// can invite users as staff or individual.
func (s *InstitutionInviteTestSuite) TestHeadAndStaffCanBothInviteNormalUsers() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Both Invite Org"))
	instIDStr := inst.ID.String()

	head := s.CreateUser("bi-head")
	staff := s.CreateUser("bi-staff")

	headInvite := Must(admin.InviteToInstitution(instIDStr, "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(head.InviteToInstitution(instIDStr, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Head invites as staff
	user1 := s.CreateUser("bi-user1")
	invite1, err := head.InviteToInstitution(instIDStr, "staff", user1.ID)
	s.NoError(err, "head should invite as staff")
	Must(user1.AcceptInvite(invite1.ID.String()))
	me1 := Must(user1.GetMe())
	s.Equal("staff", string(me1.Role.Role))
	s.T().Logf("Head invited user1 as staff")

	// Staff invites as staff
	user2 := s.CreateUser("bi-user2")
	invite2, err := staff.InviteToInstitution(instIDStr, "staff", user2.ID)
	s.NoError(err, "staff should invite as staff")
	Must(user2.AcceptInvite(invite2.ID.String()))
	me2 := Must(user2.GetMe())
	s.Equal("staff", string(me2.Role.Role))
	s.T().Logf("Staff invited user2 as staff")

	// Head invites as individual
	user3 := s.CreateUser("bi-user3")
	invite3, err := head.InviteToInstitution(instIDStr, "individual", user3.ID)
	s.NoError(err, "head should invite as individual")
	Must(user3.AcceptInvite(invite3.ID.String()))
	me3 := Must(user3.GetMe())
	s.Equal("individual", string(me3.Role.Role))
	s.T().Logf("Head invited user3 as individual")

	// Staff invites as individual
	user4 := s.CreateUser("bi-user4")
	invite4, err := staff.InviteToInstitution(instIDStr, "individual", user4.ID)
	s.NoError(err, "staff should invite as individual")
	Must(user4.AcceptInvite(invite4.ID.String()))
	me4 := Must(user4.GetMe())
	s.Equal("individual", string(me4.Role.Role))
	s.T().Logf("Staff invited user4 as individual")
}
