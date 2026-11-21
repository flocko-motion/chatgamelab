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
	s.Equal(invite.Id.String(), found.Id)

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

// TestAuthenticatedAcceptKeepsIdentity pins the contract the invite-accept page
// relies on: a recognised guest who accepts another workshop's invite keeps
// their account and moves. Sent anonymously, the very same call mints a second
// account instead, which is right for a new guest only: the page must send
// the caller's credentials.
func (s *WorkshopWordTokensTestSuite) TestAuthenticatedAcceptKeepsIdentity() {
	headA, wsA := s.workshopSetup("ident-a")
	headB, wsB := s.workshopSetup("ident-b")
	inviteA := Must(headA.CreateWorkshopInvite(wsA))
	inviteB := Must(headB.CreateWorkshopInvite(wsB))

	guest := Must(s.JoinWorkshop(*inviteA.InviteToken))
	code := s.codeOf(guest)

	// Recognised: one account, workshop moved.
	s.Require().NoError(guest.AcceptWorkshopInviteByToken(*inviteB.InviteToken))
	me := Must(guest.GetMe())
	s.Equal(guest.ID, me.Id.String(), "the account must survive the switch")
	s.Require().NotNil(me.Role)
	s.Require().NotNil(me.Role.Workshop)
	s.Equal(wsB, me.Role.Workshop.Id.String(), "the workshop must have changed")

	// The same code still logs the same person in - nothing was reissued.
	s.Equal(code, Must(headB.GetGuestToken(guest.ID)))

	// Accepting the invite of the workshop they are already in is a conflict, not a
	// silent second account. The page turns this into "you are already in".
	s.Conflict(guest.AcceptWorkshopInviteByToken(*inviteB.InviteToken))

	// Anonymously, the same invite still creates a separate account.
	second := Must(s.AcceptWorkshopInviteAnonymously(*inviteB.InviteToken))
	s.Require().NotNil(second.User)
	s.NotEqual(guest.ID, second.User.Id.String(),
		"an anonymous join must remain a new guest")
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
