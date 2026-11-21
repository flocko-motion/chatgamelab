package testing

import (
	"fmt"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// TokenLockTestSuite drives the site-wide token lock to its threshold, which
// locks every token route in the process. It runs only on its own:
//
//	TOKEN_LOCK_TEST=1 TOKEN_LOCK_MAX_FAILURES=20 TOKEN_LOCK_DURATION=3s go test -run TestTokenLockSuite
type TokenLockTestSuite struct {
	testutil.BaseSuite
}

func TestTokenLockSuite(t *testing.T) {
	if !testutil.TokenLockTestRequested() {
		t.Skip("set TOKEN_LOCK_TEST=1 TOKEN_LOCK_MAX_FAILURES=20 TOKEN_LOCK_DURATION=3s and run alone")
	}
	s := &TokenLockTestSuite{}
	s.SuiteName = "Token Lock Tests"
	suite.Run(t, s)
}

func (s *TokenLockTestSuite) TestOnlyGuessesCountAndLockBlocksAll() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Lock Org"))
	head := s.CreateUser("lock-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Lock Workshop"))
	invite := Must(head.CreateWorkshopInvite(workshop.Id.String()))
	inviteToken := *invite.InviteToken
	joined := Must(s.AcceptWorkshopInviteAnonymously(inviteToken))
	token := *joined.AuthToken

	// A whole workshop logging in, many times over, never counts.
	for range 50 {
		_, err := s.GuestLogin(token)
		s.Require().NoError(err)
	}

	// 20 guesses are tolerated; the 21st is refused like them and engages
	// the lock.
	for i := range 21 {
		_, err := s.GuestLogin(fmt.Sprintf("apfel-otter-turm-%d", i))
		s.Require().True(s.Unauthorized(err), "guess %d", i+1)
	}

	// Locked: correct tokens are refused too, on every token route.
	_, err := s.GuestLogin(token)
	s.Refused(err, testutil.CodeTokenLocked, "signing in with the guest's token")
	_, err = s.Public().GetInvite(inviteToken)
	s.Refused(err, testutil.CodeTokenLocked, "reading the invite by its token")
	_, err = s.AcceptWorkshopInviteAnonymously(inviteToken)
	s.Refused(err, testutil.CodeTokenLocked, "joining by the invite's token")
}
