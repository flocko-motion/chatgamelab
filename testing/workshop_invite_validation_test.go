package testing

import (
	"testing"

	"cgl/testing/testutil"

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
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	keyShare := Must(head.AddApiKey("mock-"+prefix, prefix+" Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), prefix+" Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	Must(head.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:   prefix + " Workshop",
		Active: true,
	}))

	return head, inst.Id.String(), wsIDStr
}

// TestCannotInviteHeadOfSameOrg verifies that head of the same org cannot be invited by email.
func (s *WorkshopInviteValidationSuite) TestCannotInviteHeadOfSameOrg() {
	head, instID, wsID := s.workshopSetup("inv-head")
	admin := s.DevUser()

	// Create another head in the same org
	otherHead := s.CreateUser("inv-head-other")
	otherInvite := Must(admin.InviteToInstitution(instID, "head", otherHead.ID))
	Must(otherHead.AcceptInvite(otherInvite.Id.String()))

	// Try to invite the other head — should fail
	_, err := head.InviteToWorkshopByEmail(wsID, otherHead.Email)
	s.Invalid(err, "should not be able to invite head of same org to workshop")
}

// TestCannotInviteStaffOfSameOrg verifies that staff of the same org cannot be invited by email.
func (s *WorkshopInviteValidationSuite) TestCannotInviteStaffOfSameOrg() {
	head, instID, wsID := s.workshopSetup("inv-staff")
	admin := s.DevUser()

	// Create staff in the same org
	staff := s.CreateUser("inv-staff-member")
	staffInvite := Must(admin.InviteToInstitution(instID, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	// Try to invite staff — should fail
	_, err := head.InviteToWorkshopByEmail(wsID, staff.Email)
	s.Invalid(err, "should not be able to invite staff of same org to workshop")
}

// TestCanInviteHeadOfDifferentOrg verifies that head of a different org CAN be invited.
func (s *WorkshopInviteValidationSuite) TestCanInviteHeadOfDifferentOrg() {
	head, _, wsID := s.workshopSetup("inv-difforg")
	admin := s.DevUser()

	// Create a different org with a head
	otherInst := Must(admin.CreateInstitution("inv-difforg Other Org"))
	otherHead := s.CreateUser("inv-difforg-other")
	otherInvite := Must(admin.InviteToInstitution(otherInst.Id.String(), "head", otherHead.ID))
	Must(otherHead.AcceptInvite(otherInvite.Id.String()))

	// Should be able to invite head from different org
	invite, err := head.InviteToWorkshopByEmail(wsID, otherHead.Email)
	s.NoError(err, "should be able to invite head of different org to workshop")
	s.NotEmpty(invite.Id)
}

// TestWorkshopInviteIncludesNames verifies that the invite listing response
// includes workshopName and institutionName for workshop invites.
func (s *WorkshopInviteValidationSuite) TestWorkshopInviteIncludesNames() {
	head, _, wsID := s.workshopSetup("inv-names")

	// Create a user and invite them
	target := s.CreateUser("inv-names-target")
	Must(head.InviteToWorkshopByEmail(wsID, target.Email))

	// Target fetches their invites with full details
	invites := Must(target.GetInvitesIncoming())
	s.Require().Len(invites, 1, "should have one pending invite")

	invite := invites[0]
	s.Require().NotNil(invite.WorkshopId, "should have workshopId")
	s.Equal(wsID, invite.WorkshopId.String())
	s.Require().NotNil(invite.WorkshopName, "should have workshopName")
	s.Equal("inv-names Workshop", *invite.WorkshopName)
	s.Equal("inv-names Org", invite.InstitutionName)
}

// TestCanInviteAnIndividual: an individual, in no organisation, is invited
// to take part in a workshop.
func (s *WorkshopInviteValidationSuite) TestCanInviteAnIndividual() {
	head, _, wsID := s.workshopSetup("inv-ind")

	individual := s.CreateUser("inv-ind-user")
	invite, err := head.InviteToWorkshopByEmail(wsID, individual.Email)
	s.NoError(err, "an individual is invited to take part in a workshop")
	s.NotEmpty(invite.Id)
}
