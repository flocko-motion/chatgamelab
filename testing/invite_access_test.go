package testing

import (
	"fmt"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// InviteAccessTestSuite checks who invites, sees and revokes invites, and
// who changes roles, as the secure core decides it (core/invite.go,
// core/role.go).
type InviteAccessTestSuite struct {
	testutil.BaseSuite
	n int
}

func TestInviteAccessSuite(t *testing.T) {
	s := &InviteAccessTestSuite{}
	s.SuiteName = "Invite access"
	suite.Run(t, s)
}

// org is an organisation at work with a head and staff.
func (s *InviteAccessTestSuite) org() (id string, head, staff *testutil.UserClient) {
	s.n++
	id = Must(s.DevUser().CreateInstitution(fmt.Sprintf("Einladung %d", s.n))).Id.String()
	return id, s.member(id, "head"), s.member(id, "staff")
}

func (s *InviteAccessTestSuite) member(org, role string) *testutil.UserClient {
	s.n++
	user := s.CreateUser(fmt.Sprintf("%s%d", role, s.n))
	invite := Must(s.DevUser().InviteToInstitution(org, role, user.ID))
	Must(user.AcceptInviteByToken(invite.Id.String()))
	return user
}

// Staff invite staff and individuals, never heads, and do not see the
// invites to become head; heads do.
func (s *InviteAccessTestSuite) TestOnlyHeadsInviteHeads() {
	org, head, staff := s.org()
	s.n++
	newcomer := s.CreateUser(fmt.Sprintf("neu%d", s.n))

	_, err := staff.InviteToInstitution(org, "head", newcomer.ID)
	s.Forbidden(err, "staff invite no heads")
	Must(staff.InviteToInstitution(org, "staff", newcomer.ID))

	s.n++
	future := s.CreateUser(fmt.Sprintf("leitung%d", s.n))
	headInvite := Must(head.InviteToInstitution(org, "head", future.ID))

	for _, inv := range Must(staff.GetInvitesOutgoing(org)) {
		s.NotEqual(headInvite.Id, inv.Id, "staff do not see invites to become head")
	}
	seen := false
	for _, inv := range Must(head.GetInvitesOutgoing(org)) {
		seen = seen || inv.Id == headInvite.Id
	}
	s.True(seen, "a head sees them")
	_, err = staff.GetInvite(headInvite.Id.String())
	s.NotFound(err)
}

// An organisation is cooperative: its head revokes a pending invite staff
// made; nobody outside it does, and an answered one stays.
func (s *InviteAccessTestSuite) TestTheOrganisationRevokes() {
	org, head, staff := s.org()
	s.n++
	newcomer := s.CreateUser(fmt.Sprintf("neu%d", s.n))
	invite := Must(staff.InviteToInstitution(org, "staff", newcomer.ID))

	_, otherHead, _ := s.org()
	s.NotFound(otherHead.RevokeInvite(invite.Id.String()), "another organisation's head")
	s.NoError(head.RevokeInvite(invite.Id.String()))

	answered := Must(staff.InviteToInstitution(org, "staff", newcomer.ID))
	Must(newcomer.DeclineInvite(answered.Id.String()))
	s.Conflict(head.RevokeInvite(answered.Id.String()), "a declined invite stays")
}

// Only the invited person answers a targeted invite.
func (s *InviteAccessTestSuite) TestOnlyTheInvitedAnswer() {
	org, head, staff := s.org()
	s.n++
	newcomer := s.CreateUser(fmt.Sprintf("neu%d", s.n))
	invite := Must(head.InviteToInstitution(org, "staff", newcomer.ID))

	_, err := staff.AcceptInvite(invite.Id.String())
	s.Error(err, "someone else accepts nothing")
	Must(newcomer.AcceptInvite(invite.Id.String()))
	s.Equal("staff", testutil.RoleOf(Must(newcomer.GetMe())))
}

// A head makes members head or staff in their organisation, never the last
// head away; only the admin takes a role away.
func (s *InviteAccessTestSuite) TestRoleChanges() {
	org, head, staff := s.org()

	Must(head.SetUserRoleIn(staff.ID, "head", org))
	Must(head.SetUserRoleIn(staff.ID, "staff", org))
	_, err := head.SetUserRoleIn(staff.ID, "admin", org)
	s.Forbidden(err, "a head makes no admin")
	_, err = head.SetUserRoleIn(head.ID, "staff", org)
	s.Refused(err, testutil.CodeLastHead, "the last head stays")

	s.Forbidden(head.RemoveUserRole(staff.ID), "only the admin takes a role away")
	s.NoError(s.DevUser().RemoveUserRole(staff.ID))
}
