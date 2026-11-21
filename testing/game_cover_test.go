package testing

import (
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameCoverTestSuite has games' covers drawn on the mock platform: paid
// through the caller's key chain, made by whoever may edit the card, seen by
// whoever may see the game.
type GameCoverTestSuite struct {
	testutil.BaseSuite
}

func TestGameCoverSuite(t *testing.T) {
	s := &GameCoverTestSuite{}
	s.SuiteName = "Game Cover Tests"
	suite.Run(t, s)
}

// designer creates a user whose default key is the mock's, and a game of
// theirs. Returns (user, gameID).
func (s *GameCoverTestSuite) designer(name string) (*testutil.UserClient, string) {
	user := s.CreateUser(name)
	Must(user.AddApiKey("mock-"+name, name+" Key", "mock"))
	game := Must(user.UploadGame("alien-first-contact"))
	return user, game.Id.String()
}

// A new game has no cover; its designer has one drawn, and it shows.
func (s *GameCoverTestSuite) TestTheDesignerHasACoverDrawn() {
	user, gameID := s.designer("cover-draw")
	s.Nil(Must(user.ReadGame(gameID)).CoverAt, "a game starts without a cover")
	_, err := user.GetCover(gameID)
	s.NotFound(err)

	game := Must(user.MakeCover(gameID, "the alien ship over a city at night"))
	s.NotNil(game.CoverAt)

	picture := Must(user.GetCover(gameID))
	s.True(testutil.IsJPEG(picture), "the cover is a JPEG")
	s.Equal(game.CoverAt, Must(user.ReadGame(gameID)).CoverAt)
	listed := Must(user.GameInList(gameID))
	s.Equal(game.CoverAt, listed.CoverAt, "lists name the cover too")
}

// Drawing again replaces the cover.
func (s *GameCoverTestSuite) TestDrawingAgainReplacesTheCover() {
	user, gameID := s.designer("cover-again")

	first := Must(user.MakeCover(gameID, "a red planet"))
	firstPicture := Must(user.GetCover(gameID))
	second := Must(user.MakeCover(gameID, "a blue ocean"))

	s.False(second.CoverAt.Before(*first.CoverAt))
	s.NotEqual(firstPicture, Must(user.GetCover(gameID)), "a new hint draws a new picture")
}

// Without a key that draws pictures there is no cover, and the game stays
// as it was.
func (s *GameCoverTestSuite) TestNoKeyNoCover() {
	user := s.CreateUser("cover-nokey")
	game := Must(user.UploadGame("alien-first-contact"))
	gameID := game.Id.String()

	_, err := user.MakeCover(gameID, "")
	s.Refused(err, testutil.CodeNoApiKey)
	s.Nil(Must(user.ReadGame(gameID)).CoverAt)
}

// Someone who may not see the game can neither have its cover drawn nor
// see it.
func (s *GameCoverTestSuite) TestOthersCannotTouchAPrivateGamesCover() {
	owner, gameID := s.designer("cover-owner")
	Must(owner.MakeCover(gameID, ""))
	stranger, _ := s.designer("cover-stranger")

	_, err := stranger.MakeCover(gameID, "")
	s.NotFound(err)
	_, err = stranger.GetCover(gameID)
	s.NotFound(err)
}

// The hint is at most 300 characters.
func (s *GameCoverTestSuite) TestALongHintIsRefused() {
	user, gameID := s.designer("cover-hint")

	_, err := user.MakeCover(gameID, strings.Repeat("x", 301))
	s.BadRequest(err)
	s.Nil(Must(user.ReadGame(gameID)).CoverAt)
}
