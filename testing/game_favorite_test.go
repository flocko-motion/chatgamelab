package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameFavoriteTestSuite tests favorites through the secure core: a favorite
// is the caller's own mark; marking takes reading the game, unmarking gives
// nothing away.
type GameFavoriteTestSuite struct {
	testutil.BaseSuite
}

func TestGameFavoriteSuite(t *testing.T) {
	s := &GameFavoriteTestSuite{}
	s.SuiteName = "Game Favorite Tests"
	suite.Run(t, s)
}

// A game says whether the caller marked it, and only for that caller.
func (s *GameFavoriteTestSuite) TestAGameSaysWhetherItIsAFavorite() {
	alice := s.CreateUser("fav-flag-a")
	bob := s.CreateUser("fav-flag-b")
	game := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.SetGamePublic(game.Id.String(), true))

	s.False(Must(alice.ReadGame(game.Id.String())).IsFavorite)
	s.Require().NoError(alice.FavoriteGame(game.Id.String()))
	s.Require().NoError(alice.FavoriteGame(game.Id.String()), "marking twice changes nothing")
	s.True(Must(alice.ReadGame(game.Id.String())).IsFavorite)
	s.False(Must(bob.ReadGame(game.Id.String())).IsFavorite, "alice's mark is hers")

	s.Require().NoError(alice.UnfavoriteGame(game.Id.String()))
	s.False(Must(alice.ReadGame(game.Id.String())).IsFavorite)
}

// Marking a game the caller may not read finds nothing; unmarking it answers
// as for any game, so neither tells whether it exists.
func (s *GameFavoriteTestSuite) TestAnUnreadableGameGivesNothingAway() {
	alice := s.CreateUser("fav-hidden-a")
	bob := s.CreateUser("fav-hidden-b")
	game := Must(alice.UploadGame("alien-first-contact"))

	s.NotFound(bob.FavoriteGame(game.Id.String()))
	s.NoError(bob.UnfavoriteGame(game.Id.String()))
	s.NoError(bob.UnfavoriteGame("00000000-0000-0000-0000-00000000beef"), "the same for no game at all")
}
