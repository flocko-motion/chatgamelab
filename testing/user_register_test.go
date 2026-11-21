package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// UserRegisterTestSuite tests registering through the secure core: an account
// from a first sign-in holds nothing of an account's rights until it
// registers, and it registers once.
type UserRegisterTestSuite struct {
	testutil.BaseSuite
}

func TestUserRegisterSuite(t *testing.T) {
	s := &UserRegisterTestSuite{}
	s.SuiteName = "User Register Tests"
	suite.Run(t, s)
}

// An account still registering is reported as such, and may not do what an
// account may: create a game.
func (s *UserRegisterTestSuite) TestRegisteringHoldsNoAccountRights() {
	newcomer := s.CreateRegisteringUser()
	me := Must(newcomer.WhoAmI())
	s.Equal("registering", me.State)
	s.Nil(me.User)

	_, err := newcomer.CreateGame("Too Early")
	s.Forbidden(err, "creating a game takes a registered account")
}

// Registering completes the account: from then on it is a registered
// individual with an account's rights.
func (s *UserRegisterTestSuite) TestRegisteringCompletesTheAccount() {
	newcomer := s.CreateRegisteringUser()
	user := Must(newcomer.Register(testutil.Registration{Name: "reg-complete", AgeGroup: "u18"}))
	s.Equal("reg-complete", user.Name)

	me := Must(newcomer.GetMe())
	s.Equal("reg-complete", me.Name)
	s.Require().NotNil(me.AgeGroup)
	s.Equal("u18", *me.AgeGroup)
	s.Require().NotNil(me.Role)
	s.Equal("individual", me.Role.Role)

	Must(newcomer.CreateGame("Now Allowed"))
}

// An account registers once.
func (s *UserRegisterTestSuite) TestAnAccountRegistersOnce() {
	newcomer := s.CreateRegisteringUser()
	Must(newcomer.Register(testutil.Registration{Name: "reg-once", AgeGroup: "u18"}))
	_, err := newcomer.Register(testutil.Registration{Name: "reg-twice", AgeGroup: "u18"})
	s.Refused(err, testutil.CodeConflict)
}

// A name another account has is refused; so are a missing age group and an
// overlong name.
func (s *UserRegisterTestSuite) TestRegistrationIsValidated() {
	s.CreateUser("reg-taken")
	newcomer := s.CreateRegisteringUser()

	_, err := newcomer.Register(testutil.Registration{Name: "reg-taken", AgeGroup: "u18"})
	s.Refused(err, testutil.CodeConflict)
	_, err = newcomer.Register(testutil.Registration{Name: "reg-no-age"})
	s.Refused(err, testutil.CodeValidation)
	_, err = newcomer.Register(testutil.Registration{Name: "abcdefghijklmnopqrstuvwxy", AgeGroup: "u18"})
	s.Refused(err, testutil.CodeNameTooLong)

	me := Must(newcomer.WhoAmI())
	s.Equal("registering", me.State, "refused registrations change nothing")
}
