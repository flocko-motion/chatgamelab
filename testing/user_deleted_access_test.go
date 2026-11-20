package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// UserDeletedAccessTestSuite tests that a deleted account's token is refused
// on every later call: the account behind it is gone.
type UserDeletedAccessTestSuite struct {
	testutil.BaseSuite
}

func TestUserDeletedAccessSuite(t *testing.T) {
	s := &UserDeletedAccessTestSuite{}
	s.SuiteName = "User Deleted Access Tests"
	suite.Run(t, s)
}

func (s *UserDeletedAccessTestSuite) TestDeletedUserCannotListGames() {
	user := s.CreateUser("del-access-list")
	Must(user.ListGames())

	MustSucceed(s.DevUser().DeleteUser(user.ID))

	_, err := user.ListGames()
	s.Forbidden(err)
}

func (s *UserDeletedAccessTestSuite) TestDeletedUserCannotCreateGame() {
	user := s.CreateUser("del-access-create")
	Must(user.UploadGame("alien-first-contact"))

	MustSucceed(s.DevUser().DeleteUser(user.ID))

	_, err := user.CreateGame("Should Fail")
	s.Forbidden(err)
}

// Asking who is calling refuses a deleted account too; it doesn't pass for
// anonymous.
func (s *UserDeletedAccessTestSuite) TestDeletedUserCannotAccessProfile() {
	user := s.CreateUser("del-access-profile")
	s.Equal(user.Name, Must(user.GetMe()).Name)

	MustSucceed(s.DevUser().DeleteUser(user.ID))

	_, err := user.WhoAmI()
	s.Forbidden(err)
}
