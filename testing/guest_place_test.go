package testing

import (
	"fmt"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GuestPlaceTestSuite checks what the database refuses whoever writes, past
// the core (migration 0008): a guest's place is fixed (spec, Roles).
type GuestPlaceTestSuite struct {
	testutil.BaseSuite
}

func TestGuestPlaceSuite(t *testing.T) {
	s := &GuestPlaceTestSuite{}
	s.SuiteName = "Guest place"
	suite.Run(t, s)
}

func (s *GuestPlaceTestSuite) TestTheDatabaseKeepsAGuestsPlace() {
	admin := s.DevUser()
	org := Must(admin.CreateInstitution("Platz Org")).Id.String()
	head := s.CreateUser("platz-head")
	Must(head.AcceptInvite(Must(admin.InviteToInstitution(org, "head", head.ID)).Id.String()))
	ws := Must(head.CreateWorkshop(org, "Platz A")).Id.String()
	other := Must(head.CreateWorkshop(org, "Platz B")).Id.String()
	guest := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(ws)).InviteToken))

	for name, stmt := range map[string]string{
		"moving to another workshop": fmt.Sprintf("UPDATE user_role SET workshop_id = '%s' WHERE user_id = '%s'", other, guest.ID),
		"leaving their workshop":     fmt.Sprintf("UPDATE user_role SET workshop_id = NULL WHERE user_id = '%s'", guest.ID),
		"entering a workshop":        fmt.Sprintf("UPDATE user_role SET active_workshop_id = '%s' WHERE user_id = '%s'", other, guest.ID),
		"turning into staff":         fmt.Sprintf("UPDATE user_role SET role = 'staff' WHERE user_id = '%s'", guest.ID),
	} {
		s.DatabaseRefuses(stmt, "a guest %s", name)
	}
	s.DatabaseRefuses(fmt.Sprintf("UPDATE user_role SET workshop_id = '%s' WHERE user_id = '%s'", ws, head.ID),
		"only a guest is in a workshop by their role; head and staff take part in workshop mode")

	// what may be true still is: head and staff take part in workshop mode
	s.DatabaseAccepts(fmt.Sprintf("UPDATE user_role SET active_workshop_id = '%s' WHERE user_id = '%s'", ws, head.ID))

	me := Must(guest.GetMe())
	s.Require().NotNil(me.Role.Workshop)
	s.Equal(ws, me.Role.Workshop.Id.String(), "the guest is where they were")
}

// The role says where someone is (migration 0007): only head and staff are
// in an organisation, whoever writes.
func (s *GuestPlaceTestSuite) TestTheDatabaseKeepsIndividualsOutOfOrganisations() {
	admin := s.DevUser()
	org := Must(admin.CreateInstitution("Ohne Org")).Id.String()
	individual := s.CreateUser("ohne-org")
	s.DatabaseRefuses(fmt.Sprintf("UPDATE user_role SET institution_id = '%s' WHERE user_id = '%s'", org, individual.ID),
		"an individual is in no organisation")
	head := s.CreateUser("ohne-org-head")
	Must(head.AcceptInvite(Must(admin.InviteToInstitution(org, "head", head.ID)).Id.String()))
	s.DatabaseAccepts(fmt.Sprintf("UPDATE user_role SET institution_id = '%s' WHERE user_id = '%s'", org, head.ID),
		"a head is in one")
}
