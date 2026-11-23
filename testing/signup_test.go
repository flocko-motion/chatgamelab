package testing

import (
	"fmt"
	"testing"
	"time"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// SignUpTestSuite checks the sign-up state (spec, Registration): an account
// is in sign-up from its first sign-in until registration completes it, and
// the janitor removes sign-ups left unfinished for a day (migration 0009,
// core.Janitor).
type SignUpTestSuite struct {
	testutil.BaseSuite
}

func TestSignUpSuite(t *testing.T) {
	s := &SignUpTestSuite{}
	s.SuiteName = "Sign-up"
	suite.Run(t, s)
}

func (s *SignUpTestSuite) TestRegistrationEndsTheSignUp() {
	admin := s.DevUser()
	u := s.CreateRegisteringUser()

	before := Must(admin.GetUser(u.ID))
	s.Require().NotNil(before.Registered)
	s.False(*before.Registered, "an account in sign-up is not registered")
	s.Nil(before.Role, "an account in sign-up holds no role")

	Must(u.Register(testutil.Registration{Name: testutil.Unique("signup"), AgeGroup: "u18"}))

	after := Must(admin.GetUser(u.ID))
	s.Require().NotNil(after.Registered)
	s.True(*after.Registered, "registration completes the account")
	s.Require().NotNil(after.Role)
	s.Equal("individual", after.Role.Role)
}

func (s *SignUpTestSuite) TestTheJanitorRemovesSignUpsLeftForADay() {
	backdate := func(id string) {
		s.DatabaseAccepts(fmt.Sprintf("UPDATE app_user SET created_at = now() - interval '25 hours' WHERE id = '%s'", id))
	}
	exists := func(id string) bool {
		return s.DatabaseCount(fmt.Sprintf("SELECT count(*) FROM app_user WHERE id = '%s'", id)) == 1
	}

	stale := s.CreateRegisteringUser()
	backdate(stale.ID)

	fresh := s.CreateRegisteringUser()

	// a registered account that lost its role (a bug): registered stays
	lostRole := s.CreateUser(testutil.Unique("lost-role"))
	s.DatabaseAccepts(fmt.Sprintf("DELETE FROM user_role WHERE user_id = '%s'", lostRole.ID))
	backdate(lostRole.ID)

	// an account in sign-up that holds a role: no unfinished sign-up
	withRole := s.CreateRegisteringUser()
	backdate(withRole.ID)
	s.DatabaseAccepts(fmt.Sprintf("INSERT INTO user_role (id, user_id, role) VALUES (gen_random_uuid(), '%s', 'individual')", withRole.ID))

	// the janitor's first pass runs at start
	s.RestartBackend()
	deadline := time.Now().Add(15 * time.Second)
	for exists(stale.ID) && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
	}

	s.False(exists(stale.ID), "a sign-up older than a day is removed")
	s.True(exists(fresh.ID), "a fresh sign-up stays")
	s.True(exists(lostRole.ID), "a registered account stays, role or not")
	s.True(exists(withRole.ID), "an account in sign-up holding a role stays")
}
