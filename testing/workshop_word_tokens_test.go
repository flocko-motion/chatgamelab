package testing

import (
	"regexp"
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// WorkshopWordTokensTestSuite tests invite and re-login tokens made of words:
// format, tolerant input, guest login without prefix, access reset, and
// replacement of a used-up invite.
type WorkshopWordTokensTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopWordTokensSuite(t *testing.T) {
	s := &WorkshopWordTokensTestSuite{}
	s.SuiteName = "Workshop Word Tokens Tests"
	suite.Run(t, s)
}

var (
	threeWords = regexp.MustCompile(`^[a-z]{4,10}(-[a-z]{4,10}){2}$`)
	fourWords  = regexp.MustCompile(`^guest-[a-z]{4,10}(-[a-z]{4,10}){3}$`)
)

func (s *WorkshopWordTokensTestSuite) workshopSetup(prefix string) (*testutil.UserClient, string) {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	keyShare := Must(head.AddApiKey("mock-word-key-"+prefix, prefix+" Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), prefix+" Workshop"))
	wsID := workshop.Id.String()
	orgShareID := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsID, &orgShareID))
	Must(head.UpdateWorkshop(wsID, testutil.WorkshopUpdate{
		Name:   prefix + " Workshop",
		Active: true,
	}))
	return head, wsID
}

// codeOf is the guest's own guest token, the code they sign in
// again with.
func (s *WorkshopWordTokensTestSuite) codeOf(guest *testutil.UserClient) string {
	code := Must(guest.GetGuestCode())
	s.Require().NotNil(code.Token, "a guest has a guest token")
	return *code.Token
}

func (s *WorkshopWordTokensTestSuite) TestInviteTokenIsThreeWords() {
	head, wsID := s.workshopSetup("word-invite")
	invite := Must(head.CreateWorkshopInvite(wsID))
	s.Require().NotNil(invite.InviteToken)
	s.Regexp(threeWords, *invite.InviteToken)

	// Typed with capitals and spaces instead of hyphens.
	typed := strings.ToUpper(strings.ReplaceAll(*invite.InviteToken, "-", " "))
	found := Must(s.Public().GetInvite(typed))
	s.Equal(invite.Id, found.Id)

	// Joining with it gives the guest a code of four words.
	guest := Must(s.JoinWorkshop(typed))
	s.Regexp(fourWords, s.codeOf(guest))
}

func (s *WorkshopWordTokensTestSuite) TestGuestLoginWithoutPrefix() {
	head, wsID := s.workshopSetup("word-login")
	invite := Must(head.CreateWorkshopInvite(wsID))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))
	words := strings.TrimPrefix(s.codeOf(guest), "guest-")

	again := Must(s.GuestLogin(strings.ToUpper(words)))
	s.Equal(guest.ID, again.ID)

	_, err := s.GuestLogin("apfel-otter-turm-leise")
	s.Unauthorized(err)
}

func (s *WorkshopWordTokensTestSuite) TestResetGuestToken() {
	head, wsID := s.workshopSetup("word-reset")
	invite := Must(head.CreateWorkshopInvite(wsID))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))
	oldCode := s.codeOf(guest)

	newToken := Must(head.ResetGuestToken(guest.ID))
	s.Regexp(fourWords, newToken)
	s.NotEqual(oldCode, newToken)

	// The reset ends the guest's sessions and their old code.
	_, err := guest.WhoAmI()
	s.Unauthorized(err, "the session started before the reset must end")
	_, err = s.GuestLogin(oldCode)
	s.Unauthorized(err, "the old code must stop working")

	relogged := Must(s.GuestLogin(newToken))
	s.Equal(guest.ID, relogged.ID)

	// A guest cannot reset anyone's token.
	_, err = relogged.ResetGuestToken(guest.ID)
	s.Forbidden(err)
}

func (s *WorkshopWordTokensTestSuite) TestUsedUpInviteIsReplaced() {
	head, wsID := s.workshopSetup("word-usedup")
	one := 1
	first := Must(head.CreateWorkshopInviteWith(testutil.WorkshopInvite{WorkshopId: wsID, MaxUses: &one}))
	s.Require().NotNil(first.InviteToken)
	Must(s.AcceptWorkshopInviteAnonymously(*first.InviteToken))

	second := Must(head.CreateWorkshopInvite(wsID))
	s.NotEqual(first.Id, second.Id, "a used-up invite must be replaced")
	s.Equal("pending", second.Status)
}

func (s *WorkshopWordTokensTestSuite) TestOwnGuestCode() {
	head, wsID := s.workshopSetup("word-own")
	invite := Must(head.CreateWorkshopInvite(wsID))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))

	code := Must(guest.GetGuestCode())
	s.Require().NotNil(code.Token)
	s.Require().NotNil(code.Words)
	s.Equal(Must(head.GetGuestToken(guest.ID)), *code.Token, "the guest sees the code their leader sees")
	s.Equal("guest-"+*code.Words, *code.Token)

	// After a reset the guest sees the new code.
	reset := Must(head.ResetGuestToken(guest.ID))
	s.Equal(reset, s.codeOf(Must(s.GuestLogin(reset))))

	// Users without a guest token get an explicit empty answer.
	none := Must(head.GetGuestCode())
	s.Nil(none.Token)
	s.Nil(none.Words)

	_, err := s.Public().GetGuestCode()
	s.Unauthorized(err)
}

// TestAGuestStaysInTheirWorkshop pins what tells the two kinds of guest
// apart (spec, Roles): a guest's place is fixed. A recognised guest who
// accepts another workshop's invite is refused and stays; their own
// workshop's invite answers "you are in already" (the page sends them
// there). Sent anonymously, the same invite makes a new guest, which is
// right for a newcomer only.
func (s *WorkshopWordTokensTestSuite) TestAGuestStaysInTheirWorkshop() {
	headA, wsA := s.workshopSetup("ident-a")
	headB, wsB := s.workshopSetup("ident-b")
	inviteA := Must(headA.CreateWorkshopInvite(wsA))
	inviteB := Must(headB.CreateWorkshopInvite(wsB))

	guest := Must(s.JoinWorkshop(*inviteA.InviteToken))

	s.Refused(guest.AcceptWorkshopInviteByToken(*inviteB.InviteToken), testutil.CodeGuestStays, "a guest moves to no other workshop")
	me := Must(guest.GetMe())
	s.Equal(guest.ID, me.Id.String())
	s.Require().NotNil(me.Role)
	s.Require().NotNil(me.Role.Workshop)
	s.Equal(wsA, me.Role.Workshop.Id.String(), "the guest stays in their workshop")

	s.Conflict(guest.AcceptWorkshopInviteByToken(*inviteA.InviteToken), "their own workshop: in already")

	second := Must(s.AcceptWorkshopInviteAnonymously(*inviteB.InviteToken))
	s.Require().NotNil(second.User)
	s.NotEqual(guest.ID, second.User.Id.String(), "an anonymous join is a new guest")
}

// A guest leaves no workshop: leaving workshop mode is refused, and they
// stay in it.
func (s *WorkshopWordTokensTestSuite) TestAGuestLeavesNoWorkshop() {
	head, ws := s.workshopSetup("stay")
	guest := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(ws)).InviteToken))

	_, err := guest.SetActiveWorkshop(nil)
	s.Forbidden(err, "a guest may not leave their workshop")
	me := Must(guest.GetMe())
	s.Require().NotNil(me.Role)
	s.Require().NotNil(me.Role.Workshop)
	s.Equal(ws, me.Role.Workshop.Id.String(), "the guest stays in their workshop")
}

// TestResetFailsWhenThereIsNoCode covers the reset of someone who joined with a real
// account: they keep their own role and get no word code, because only an anonymous
// join issues one. A reset that answered 200 there would hand the leader four words
// to read out that nothing would ever accept.
func (s *WorkshopWordTokensTestSuite) TestResetFailsWhenThereIsNoCode() {
	head, wsID := s.workshopSetup("word-nocode")
	invite := Must(head.CreateWorkshopInvite(wsID))

	registered := s.CreateUser("word-nocode-registered")
	s.Require().NoError(registered.AcceptWorkshopInviteByToken(*invite.InviteToken))

	// The read side already said so; the reset must now agree.
	_, err := head.GetGuestToken(registered.ID)
	s.NotFound(err)
	_, err = head.ResetGuestToken(registered.ID)
	s.NotFound(err)

	// And it wrote nothing: still no token afterwards.
	_, err = head.GetGuestToken(registered.ID)
	s.NotFound(err)

	// A Guest still gets a fresh code, and the old one dies.
	anonymous := Must(s.JoinWorkshop(*invite.InviteToken))
	oldCode := s.codeOf(anonymous)
	reset := Must(head.ResetGuestToken(anonymous.ID))
	s.Regexp(fourWords, reset)
	s.NotEqual(oldCode, reset)
	_, err = s.GuestLogin(oldCode)
	s.Unauthorized(err, "the old code must stop working")
	_, err = anonymous.WhoAmI()
	s.Unauthorized(err, "the session started with the old code must end")
}
