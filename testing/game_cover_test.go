package testing

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	coverpic "cgl/picture"
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

// A cover goes with its game: deleting an account deletes its games for
// good, covers included.
func (s *GameCoverTestSuite) TestDeletingAnAccountTakesItsCovers() {
	designer := s.CreateUser("cover-gone")
	Must(designer.AddApiKey("mock-cover-gone", "Cover Key", "mock"))
	game := Must(designer.UploadGame("alien-first-contact")).Id.String()
	Must(designer.MakeCover(game, ""))

	s.NoError(s.DevUser().DeleteUser(designer.ID), "an account with a covered game is deleted")
}

// A picture brought in from a game's .cgl file becomes the cover once it passes
// the youth-protection check, stored at cover size as a JPEG.
func (s *GameCoverTestSuite) TestABroughtInPictureBecomesTheCover() {
	user, gameID := s.designer("cover-upload")

	picture := testutil.PNG(1200, 900)
	game := Must(user.SetCover(gameID, picture, coverpic.Sign(picture)))
	s.NotNil(game.CoverAt)
	stored := Must(user.GetCover(gameID))
	s.True(testutil.IsJPEG(stored), "the cover is stored as a JPEG")
	s.Less(len(stored), 200_000, "the cover is stored at cover size")
}

// What is no picture, or too large for a game file, is refused, and the
// game keeps its cover.
func (s *GameCoverTestSuite) TestABadUploadIsRefused() {
	user, gameID := s.designer("cover-bad")

	notAPicture := []byte("no picture at all")
	_, err := user.SetCover(gameID, notAPicture, coverpic.Sign(notAPicture))
	s.Refused(err, testutil.CodeNotAPicture)
	_, err = user.SetCover(gameID, make([]byte, 1<<20+1), "")
	s.Refused(err, testutil.CodeFileTooLarge)
	s.Nil(Must(user.ReadGame(gameID)).CoverAt)
}

// Checking a picture is paid like drawing one: without a key, no cover.
func (s *GameCoverTestSuite) TestNoKeyNoBroughtInCover() {
	user := s.CreateUser("cover-upload-nokey")
	gameID := Must(user.UploadGame("alien-first-contact")).Id.String()

	picture := testutil.PNG(60, 40)
	_, err := user.SetCover(gameID, picture, coverpic.Sign(picture))
	s.Refused(err, testutil.CodeNoApiKey)
	s.Nil(Must(user.ReadGame(gameID)).CoverAt)
}

// Someone who may not edit the game cannot bring in its cover.
func (s *GameCoverTestSuite) TestOthersCannotBringInACover() {
	_, gameID := s.designer("cover-upload-owner")
	stranger, _ := s.designer("cover-upload-stranger")

	picture := testutil.PNG(60, 40)
	_, err := stranger.SetCover(gameID, picture, coverpic.Sign(picture))
	s.NotFound(err)
}

// A picture no ChatGameLab server signed, or one changed since, is turned
// away before anything is paid for checking it.
func (s *GameCoverTestSuite) TestAnUnsignedPictureIsRefused() {
	user, gameID := s.designer("cover-unsigned")
	picture := testutil.PNG(60, 40)

	_, err := user.SetCover(gameID, picture, "")
	s.Refused(err, testutil.CodeCoverUnsigned)
	_, err = user.SetCover(gameID, picture, coverpic.Sign(testutil.PNG(61, 40)))
	s.Refused(err, testutil.CodeCoverUnsigned)
	s.Nil(Must(user.ReadGame(gameID)).CoverAt)
}

// A cover exported in a game's .cgl file is taken back by its signature.
func (s *GameCoverTestSuite) TestAnExportedCoverComesBack() {
	user, gameID := s.designer("cover-roundtrip")
	Must(user.MakeCover(gameID, ""))
	files := zipFiles(Must(user.GetGameCgl(gameID)))

	game := Must(user.SetCover(gameID, files["cover.jpg"], string(files["cover.sig"])))
	s.NotNil(game.CoverAt)
}

// A game's .cgl file holds its game file and its cover; without a cover, the
// file alone.
func (s *GameCoverTestSuite) TestTheCglFileHoldsTheGameAndItsCover() {
	user, gameID := s.designer("cover-zip")

	s.Equal([]string{"game.yaml"}, zipNames(Must(user.GetGameCgl(gameID))))
	Must(user.MakeCover(gameID, ""))
	s.Equal([]string{"game.yaml", "cover.jpg", "cover.sig"}, zipNames(Must(user.GetGameCgl(gameID))))
}

// zipFiles reads a zip's files by name.
func zipFiles(body []byte) map[string][]byte {
	zr := Must(zip.NewReader(bytes.NewReader(body), int64(len(body))))
	files := map[string][]byte{}
	for _, f := range zr.File {
		rc := Must(f.Open())
		files[f.Name] = Must(io.ReadAll(rc))
		rc.Close()
	}
	return files
}

// zipNames lists a zip's files in order.
func zipNames(body []byte) []string {
	zr := Must(zip.NewReader(bytes.NewReader(body), int64(len(body))))
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}
