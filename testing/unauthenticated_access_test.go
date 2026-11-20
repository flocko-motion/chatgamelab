package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// UnauthenticatedAccessTestSuite tests that callers without any credentials
// are refused on protected endpoints, a sanity check for the sign-in.
type UnauthenticatedAccessTestSuite struct {
	testutil.BaseSuite
}

func TestUnauthenticatedAccessSuite(t *testing.T) {
	s := &UnauthenticatedAccessTestSuite{}
	s.SuiteName = "Unauthenticated Access Tests"
	suite.Run(t, s)
}

func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListUsers() {
	_, err := s.Public().GetUsers()
	s.Unauthorized(err)
}

// Asking who is calling answers an anonymous caller: anonymous, without an
// account.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedIsAnonymous() {
	me := Must(s.Public().WhoAmI())
	s.Equal("anonymous", me.State)
	s.Nil(me.User)

	_, err := s.Public().GetStats()
	s.Unauthorized(err, "the caller's own statistics need a sign-in")
}

func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotGetUserByID() {
	user := s.CreateUser("unauth-target")

	_, err := s.Public().GetUser(user.ID)
	s.Unauthorized(err)
}

func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListOwnGames() {
	_, err := s.Public().ListGamesWithFilter("own")
	s.Unauthorized(err)
}

func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotCreateGame() {
	_, err := s.Public().CreateGame("Unauthorized Game")
	s.Unauthorized(err)
}

func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListApiKeys() {
	_, err := s.Public().GetApiKeys()
	s.Unauthorized(err)
}

func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListInstitutions() {
	_, err := s.Public().ListInstitutions()
	s.Unauthorized(err)
}

func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListWorkshops() {
	_, err := s.Public().ListAllWorkshops()
	s.Unauthorized(err)
}

// A caller without credentials can neither change nor delete someone's game;
// the game stays as it was.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotChangeOrDeleteGame() {
	user := s.CreateUser("unauth-del")
	game := Must(user.UploadGame("alien-first-contact"))

	hacked := game
	hacked.Name = "Hacked"
	_, err := s.Public().UpdateGame(game.Id.String(), hacked)
	s.Unauthorized(err)

	s.Unauthorized(s.Public().DeleteGame(game.Id.String()))

	s.Equal(game.Name, Must(user.GetGameByID(game.Id.String())).Name)
}

// No platform access without a sign-in: not even the game list.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListGames() {
	_, err := s.Public().ListGames()
	s.Unauthorized(err)
}

// Public games aren't listed to a caller without a sign-in either.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotListPublicGames() {
	_, err := s.Public().ListGamesWithFilter("public")
	s.Unauthorized(err)
}

func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCannotGetGameByID() {
	user := s.CreateUser("unauth-game")
	game := Must(user.UploadGame("alien-first-contact"))

	_, err := s.Public().GetGameByID(game.Id.String())
	s.Unauthorized(err)
}

// The status report answers without a sign-in.
func (s *UnauthenticatedAccessTestSuite) TestUnauthenticatedCanAccessStatus() {
	status := Must(s.Public().Status())
	s.Equal("ok", status["status"])
}
