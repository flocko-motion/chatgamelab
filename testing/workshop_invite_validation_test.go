package testing

import (
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WorkshopInviteValidationSuite tests workshop invite validation rules:
// - Head/staff of the same org cannot be invited to workshops (they already have access)
// - Workshop invites include workshop name and institution name in the response
type WorkshopInviteValidationSuite struct {
	testutil.BaseSuite
}

func TestWorkshopInviteValidationSuite(t *testing.T) {
	s := &WorkshopInviteValidationSuite{}
	s.SuiteName = "Workshop Invite Validation Tests"
	suite.Run(t, s)
}

// workshopSetup creates an institution, head, workshop with API key
func (s *WorkshopInviteValidationSuite) workshopSetup(prefix string) (*testutil.UserClient, string, string) {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	keyShare := Must(head.AddApiKey("mock-"+prefix, prefix+" Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))

	workshop := Must(head.CreateWorkshop(inst.ID.String(), prefix+" Workshop"))
	wsIDStr := workshop.ID.String()
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":             prefix + " Workshop",
		"active":           true,
		"public":           false,
		"isPaused":         false,
	}))

	return head, inst.ID.String(), wsIDStr
}

// TestCannotInviteHeadOfSameOrg verifies that head of the same org cannot be invited by email.
func (s *WorkshopInviteValidationSuite) TestCannotInviteHeadOfSameOrg() {
	head, instID, wsID := s.workshopSetup("inv-head")
	admin := s.DevUser()

	// Create another head in the same org
	otherHead := s.CreateUser("inv-head-other")
	otherInvite := Must(admin.InviteToInstitution(instID, "head", otherHead.ID))
	Must(otherHead.AcceptInvite(otherInvite.ID.String()))

	// Try to invite the other head — should fail
	_, err := head.InviteToWorkshopByEmail(wsID, otherHead.Email)
	s.Error(err, "should not be able to invite head of same org to workshop")
}

// TestCannotInviteStaffOfSameOrg verifies that staff of the same org cannot be invited by email.
func (s *WorkshopInviteValidationSuite) TestCannotInviteStaffOfSameOrg() {
	head, instID, wsID := s.workshopSetup("inv-staff")
	admin := s.DevUser()

	// Create staff in the same org
	staff := s.CreateUser("inv-staff-member")
	staffInvite := Must(admin.InviteToInstitution(instID, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	// Try to invite staff — should fail
	_, err := head.InviteToWorkshopByEmail(wsID, staff.Email)
	s.Error(err, "should not be able to invite staff of same org to workshop")
}

// TestCanInviteHeadOfDifferentOrg verifies that head of a different org CAN be invited.
func (s *WorkshopInviteValidationSuite) TestCanInviteHeadOfDifferentOrg() {
	head, _, wsID := s.workshopSetup("inv-difforg")
	admin := s.DevUser()

	// Create a different org with a head
	otherInst := Must(admin.CreateInstitution("inv-difforg Other Org"))
	otherHead := s.CreateUser("inv-difforg-other")
	otherInvite := Must(admin.InviteToInstitution(otherInst.ID.String(), "head", otherHead.ID))
	Must(otherHead.AcceptInvite(otherInvite.ID.String()))

	// Should be able to invite head from different org
	invite, err := head.InviteToWorkshopByEmail(wsID, otherHead.Email)
	s.NoError(err, "should be able to invite head of different org to workshop")
	s.NotEmpty(invite.ID)
}

// TestWorkshopInviteIncludesNames verifies that the invite listing response
// includes workshopName and institutionName for workshop invites.
func (s *WorkshopInviteValidationSuite) TestWorkshopInviteIncludesNames() {
	head, _, wsID := s.workshopSetup("inv-names")

	// Create a user and invite them
	target := s.CreateUser("inv-names-target")
	Must(head.InviteToWorkshopByEmail(wsID, target.Email))

	// Target fetches their invites with full details
	invites := Must(target.GetInvitesIncomingDetailed())
	s.Require().Len(invites, 1, "should have one pending invite")

	invite := invites[0]
	s.NotNil(invite.WorkshopID, "should have workshopId")
	s.NotNil(invite.WorkshopName, "should have workshopName")
	s.Equal("inv-names Workshop", *invite.WorkshopName)
	s.NotNil(invite.InstitutionName, "should have institutionName")
	s.Equal("inv-names Org", *invite.InstitutionName)
}

// TestCanInviteIndividualOfSameOrg verifies that individual of the same org CAN be invited.
func (s *WorkshopInviteValidationSuite) TestCanInviteIndividualOfSameOrg() {
	head, instID, wsID := s.workshopSetup("inv-ind")
	admin := s.DevUser()

	// Create an individual in the same org
	individual := s.CreateUser("inv-ind-user")
	indInvite := Must(admin.InviteToInstitution(instID, "individual", individual.ID))
	Must(individual.AcceptInvite(indInvite.ID.String()))

	// Should be able to invite individual from same org
	invite, err := head.InviteToWorkshopByEmail(wsID, individual.Email)
	s.NoError(err, "should be able to invite individual of same org to workshop")
	s.NotEmpty(invite.ID)
}
