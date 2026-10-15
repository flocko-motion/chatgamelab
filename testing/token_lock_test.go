package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"fmt"
	"os"
	"strings"
	"testing"

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
	if os.Getenv("TOKEN_LOCK_TEST") != "1" || os.Getenv("TOKEN_LOCK_MAX_FAILURES") != "20" {
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
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "Lock Workshop"))
	wsID := workshop.ID.String()
	Must(head.UpdateWorkshop(wsID, map[string]interface{}{"name": "Lock Workshop", "active": true}))
	invite := Must(head.CreateWorkshopInvite(wsID, string(obj.RoleParticipant)))
	resp := Must(s.AcceptWorkshopInviteAnonymously(*invite.InviteToken))
	token := *resp.AuthToken
	participant := s.CreateUserWithToken(token)
	login := func(tok string) error {
		return s.Public().Post("auth/participant-login", map[string]string{"token": tok}, nil)
	}

	// A whole workshop logging in, many times over, never counts.
	for range 50 {
		s.Require().NoError(login(token))
	}

	// 20 guesses are tolerated, the 21st engages the lock.
	for i := range 20 {
		err := login(fmt.Sprintf("apfel-otter-turm-%d", i))
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "(401)")
	}
	s.Require().Error(login("apfel-otter-turm-last"))

	// Locked: correct tokens fail too, on every token route.
	err := login(token)
	s.Require().Error(err)
	s.True(strings.Contains(err.Error(), "(429)"), err.Error())
	var me obj.User
	err = participant.Get("users/me", &me)
	s.Require().Error(err)
	s.Contains(err.Error(), "(429)")
	var inv map[string]interface{}
	err = s.Public().Get("invites/"+*invite.InviteToken, &inv)
	s.Require().Error(err)
	s.Contains(err.Error(), "(429)")
}
