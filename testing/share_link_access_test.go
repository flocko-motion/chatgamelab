package testing

import (
	"fmt"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// ShareLinkAccessTestSuite checks who sees, changes and stops share links,
// and who pays, as the secure core decides it (core/share_link.go,
// core/payer.go; #451).
type ShareLinkAccessTestSuite struct {
	testutil.BaseSuite
	n int
}

func TestShareLinkAccessSuite(t *testing.T) {
	s := &ShareLinkAccessTestSuite{}
	s.SuiteName = "Share link access"
	suite.Run(t, s)
}

// org is an organisation whose staff member owner gave their key to it,
// with a colleague on its staff and a head.
func (s *ShareLinkAccessTestSuite) org() (id string, head, owner, colleague *testutil.UserClient, orgShare string) {
	s.n++
	id = Must(s.DevUser().CreateInstitution(fmt.Sprintf("Gutschein %d", s.n))).Id.String()
	member := func(role string) *testutil.UserClient {
		s.n++
		user := s.CreateUser(fmt.Sprintf("%s%d", role, s.n))
		invite := Must(s.DevUser().InviteToInstitution(id, role, user.ID))
		Must(user.AcceptInviteByToken(invite.Id.String()))
		return user
	}
	head, owner, colleague = member("head"), member("staff"), member("staff")
	key := Must(owner.AddApiKey(fmt.Sprintf("mock-link-%d", s.n), "Link key", "mock"))
	orgShare = Must(owner.ShareApiKeyWithInstitution(key.Id.String(), id)).Id.String()
	return id, head, owner, colleague, orgShare
}

// publicGame is a public game of user's.
func (s *ShareLinkAccessTestSuite) publicGame(user *testutil.UserClient) string {
	game := Must(user.UploadGame("alien-first-contact"))
	Must(user.SetGamePublic(game.Id.String(), true))
	return game.Id.String()
}

// A link a colleague paid with the organisation's key is the payer's to see
// and stop, and the organisation's: whoever made it. Nobody outside sees it.
func (s *ShareLinkAccessTestSuite) TestThePayerAndTheOrganisationStopALink() {
	_, head, owner, colleague, orgShare := s.org()
	game := s.publicGame(colleague)

	link := Must(colleague.CreateGameShare(game, orgShare, nil))
	s.True(link.MayManage)

	s.n++
	stranger := s.CreateUser(fmt.Sprintf("fremd%d", s.n))
	s.NotFound(stranger.DeleteGameShare(game, link.Id.String()), "for a stranger it does not exist")

	seen := Must(owner.GetPrivateShareStatus(game)).Shares
	s.Require().Len(seen, 1, "the payer sees it")
	s.True(seen[0].MayManage, "the payer may stop it")
	s.NoError(owner.DeleteGameShare(game, link.Id.String()))

	again := Must(colleague.CreateGameShare(game, orgShare, nil))
	s.NoError(head.DeleteGameShare(game, again.Id.String()), "the head stops it too")
}

// Taking the organisation's share back stops the links drawn from it.
func (s *ShareLinkAccessTestSuite) TestTakingTheSourceBackStopsItsLinks() {
	_, _, owner, colleague, orgShare := s.org()
	game := s.publicGame(colleague)
	Must(colleague.CreateGameShare(game, orgShare, nil))
	s.Len(Must(colleague.GetPrivateShareStatus(game)).Shares, 1)

	s.NoError(owner.DeleteApiKey(orgShare, false))
	s.Empty(Must(colleague.GetPrivateShareStatus(game)).Shares, "the link stopped with its source")
}

// Those in a workshop see its link, so they hand it on, but do not change
// it.
func (s *ShareLinkAccessTestSuite) TestGuestsDoNotChangeWorkshopLinks() {
	org, head, _, _, orgShare := s.org()
	workshop := Must(head.CreateWorkshop(org, fmt.Sprintf("Werkstatt %d", s.n)))
	wsID := workshop.Id.String()
	Must(head.SetWorkshopApiKey(wsID, &orgShare))
	Must(head.SetActiveWorkshop(&wsID))
	game := Must(head.UploadGame("alien-first-contact"))
	link := Must(head.CreateWorkshopGameShare(game.Id.String(), wsID, nil))

	guest := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(wsID)).InviteToken))
	seen := Must(guest.GetPrivateShareStatus(game.Id.String())).Shares
	s.Require().Len(seen, 1, "the guest sees the workshop's link")
	s.False(seen[0].MayManage)
	limit := 1
	_, err := guest.SetGameShareLimit(game.Id.String(), link.Id.String(), &limit)
	s.Forbidden(err, "a guest changes no link")
}

// The organisation's free-use key pays for its members, not for the guests
// in its workshops.
func (s *ShareLinkAccessTestSuite) TestFreeUseIsForMembers() {
	org, head, _, colleague, orgShare := s.org()
	Must(head.SetInstitutionFreeUseApiKey(org, &orgShare))
	game := s.publicGame(head)

	s.True(Must(colleague.GetApiKeyStatus(game)), "a member plays with it")

	workshop := Must(head.CreateWorkshop(org, fmt.Sprintf("Ohne Schlüssel %d", s.n)))
	guest := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshop.Id.String())).InviteToken))
	s.False(Must(guest.GetApiKeyStatus(game)), "a guest does not")
}
