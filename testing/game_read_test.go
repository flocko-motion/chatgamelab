package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameReadTestSuite tests reading one game: it carries the caller's rights
// like a listed game, and a game the caller may not read does not exist for
// them. It also tests the share right and the others filter.
type GameReadTestSuite struct {
	testutil.BaseSuite
}

func TestGameReadSuite(t *testing.T) {
	s := &GameReadTestSuite{}
	s.SuiteName = "Game Read Tests"
	suite.Run(t, s)
}

// A game opened by its id carries the caller's rights, as in the list.
func (s *GameReadTestSuite) TestAReadGameCarriesItsRights() {
	alice := s.CreateUser("read-owner")
	game := Must(alice.UploadGame("alien-first-contact"))

	read := Must(alice.ReadGame(game.Id.String()))
	s.Equal(game.Id, read.Id)
	s.True(read.HasContent)
	s.True(read.HasSettings)
	s.True(read.MayEditCard)
	s.True(read.MayShare)
}

// Someone else's private game does not exist for the caller.
func (s *GameReadTestSuite) TestAnotherUsersPrivateGameIsNotFound() {
	alice := s.CreateUser("read-private-owner")
	bob := s.CreateUser("read-private-other")
	game := Must(alice.UploadGame("alien-first-contact"))

	_, err := bob.ReadGame(game.Id.String())
	s.NotFound(err)
}

// Another account may share a public game, not edit it.
func (s *GameReadTestSuite) TestOthersMayShareAPublicGame() {
	alice := s.CreateUser("read-public-owner")
	bob := s.CreateUser("read-public-other")
	game := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.SetGamePublic(game.Id.String(), true))

	read := Must(bob.ReadGame(game.Id.String()))
	s.True(read.MayShare)
	s.False(read.MayEditCard)
}

// A guest may share only while their workshop allows it.
func (s *GameReadTestSuite) TestAGuestSharesOnlyWhileTheWorkshopAllows() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Read Share Org"))
	head := s.CreateUser("read-share-head")
	Must(head.AcceptInvite(Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID)).Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Read Share Workshop"))
	workshopID := workshop.Id.String()
	guest := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshopID)).InviteToken))
	game := Must(guest.UploadGame("alien-first-contact"))

	before := Must(guest.ReadGame(game.Id.String()))
	s.False(before.MayShare, "sharing is off by default")

	Must(head.UpdateWorkshop(workshopID, testutil.WorkshopUpdate{
		Name:             "Read Share Workshop",
		Active:           true,
		AllowGameSharing: true,
	}))
	after := Must(guest.ReadGame(game.Id.String()))
	s.True(after.MayShare, "the workshop allows sharing now")
}

// The others filter leaves out the caller's own games.
func (s *GameReadTestSuite) TestOthersFilter() {
	alice := s.CreateUser("read-others")
	bob := s.CreateUser("read-others-bob")
	mine := Must(alice.UploadGame("alien-first-contact"))
	theirs := Must(bob.UploadGame("alien-first-contact"))
	Must(bob.SetGamePublic(theirs.Id.String(), true))

	page := Must(alice.ListGamesPage(0, 0, "others"))
	var sawTheirs bool
	for _, g := range page.Items {
		s.NotEqual(mine.Id, g.Id, "own games are left out")
		if g.Id == theirs.Id {
			sawTheirs = true
		}
	}
	s.True(sawTheirs, "others' public games are in")
}
