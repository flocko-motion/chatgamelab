package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameListTestSuite tests the game list's contract: pages and their total,
// the filters the list knows, its orders, and refusing what it does not know.
type GameListTestSuite struct {
	testutil.BaseSuite
}

func TestGameListSuite(t *testing.T) {
	s := &GameListTestSuite{}
	s.SuiteName = "Game List Tests"
	suite.Run(t, s)
}

// Pages of the own list add up to its total, without overlap.
func (s *GameListTestSuite) TestPagesAddUpToTheTotal() {
	alice := s.CreateUser("list-pages")
	for range 3 {
		Must(alice.UploadGame("alien-first-contact"))
	}

	first := Must(alice.ListGamesPage(2, 0, "own"))
	second := Must(alice.ListGamesPage(2, 2, "own"))

	s.Equal(3, first.Total, "total counts every match across pages")
	s.Equal(3, second.Total)
	s.Len(first.Items, 2)
	s.Len(second.Items, 1)
	for _, a := range first.Items {
		for _, b := range second.Items {
			s.NotEqual(a.Id, b.Id, "pages never overlap")
		}
	}
}

// A page past the end is empty and still tells the total.
func (s *GameListTestSuite) TestPagePastTheEndIsEmpty() {
	alice := s.CreateUser("list-past-end")
	Must(alice.UploadGame("alien-first-contact"))

	page := Must(alice.ListGamesPage(10, 50, "own"))
	s.Empty(page.Items)
	s.Equal(1, page.Total)
}

// The favorites filter lists only the caller's favorites.
func (s *GameListTestSuite) TestFavoritesFilter() {
	alice := s.CreateUser("list-favorites")
	liked := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.UploadGame("alien-first-contact"))
	s.Require().NoError(alice.FavoriteGame(liked.Id.String()))

	page := Must(alice.ListGamesPage(0, 0, "favorites"))
	s.Equal(1, page.Total)
	s.Require().Len(page.Items, 1)
	s.Equal(liked.Id, page.Items[0].Id)
}

// Sorting by name orders the list, ascending and descending.
func (s *GameListTestSuite) TestSortByName() {
	alice := s.CreateUser("list-sort")
	Must(alice.UploadGame("alien-first-contact"))
	Must(alice.UploadGame("alien-first-contact"))

	// her own only: other suites add public games meanwhile
	asc := Must(alice.ListGamesSorted("name", "asc", "own"))
	desc := Must(alice.ListGamesSorted("name", "desc", "own"))
	s.Require().NotEmpty(asc)
	s.Require().Equal(len(asc), len(desc))
	s.Equal(asc[0].Id, desc[len(desc)-1].Id, "descending is ascending reversed")
}

// A filter the list does not know is refused, not ignored.
func (s *GameListTestSuite) TestUnknownFilterIsRefused() {
	alice := s.CreateUser("list-unknown-filter")
	_, err := alice.ListGamesPage(0, 0, "organization")
	s.Refused(err, testutil.CodeValidation)
}

// Filters combine: each narrows the list further.
func (s *GameListTestSuite) TestFiltersCombine() {
	alice := s.CreateUser("list-combine")
	bob := s.CreateUser("list-combine-bob")
	mine := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.UploadGame("alien-first-contact"))
	theirs := Must(bob.UploadGame("alien-first-contact"))
	Must(bob.SetGamePublic(theirs.Id.String(), true))
	s.Require().NoError(alice.FavoriteGame(mine.Id.String()))
	s.Require().NoError(alice.FavoriteGame(theirs.Id.String()))

	page := Must(alice.ListGamesPage(0, 0, "own", "favorites"))
	s.Equal(1, page.Total, "own and favorites: only alice's own favourite")
	s.Require().Len(page.Items, 1)
	s.Equal(mine.Id, page.Items[0].Id)
}

// Every parameter is optional: a list asked for nothing answers.
func (s *GameListTestSuite) TestEveryParameterIsOptional() {
	alice := s.CreateUser("list-unasked")
	Must(alice.UploadGame("alien-first-contact"))

	page := Must(alice.ListGamesUnasked())
	s.GreaterOrEqual(page.Total, 1)
}

// A game shows its creator's public profile name, and the list sorts by it.
func (s *GameListTestSuite) TestCreatorNameAndSort() {
	alice := s.CreateUser("list-creator")
	game := Must(alice.UploadGame("alien-first-contact"))

	games := Must(alice.ListGamesSorted("creator", "asc"))
	var found bool
	for _, g := range games {
		if g.Id == game.Id {
			found = true
			s.Require().NotNil(g.CreatorName)
			s.Equal("list-creator", *g.CreatorName)
		}
	}
	s.True(found, "alice's game is in her list")
}

// The genre filters narrow the list to games of the genres asked for; several
// genres are one filter, met by a game of any of them.
func (s *GameListTestSuite) TestGenreFilter() {
	alice := s.CreateUser("list-genre")
	adventure := Must(alice.UploadGame("alien-first-contact"))
	troll := Must(alice.UploadGame("bridge-troll"))

	live := Must(alice.ListGamesPage(0, 0, "own", "genre:npc"))
	s.Equal(1, live.Total, "own and npc: only the troll")
	s.Require().Len(live.Items, 1)
	s.Equal(troll.Id, live.Items[0].Id)

	both := Must(alice.ListGamesPage(0, 0, "own", "genre:npc", "genre:adventure"))
	s.Equal(2, both.Total, "two genres: games of either")
	s.ElementsMatch([]string{adventure.Id.String(), troll.Id.String()},
		[]string{both.Items[0].Id.String(), both.Items[1].Id.String()})

	none := Must(alice.ListGamesPage(0, 0, "own", "genre:no-such-genre"))
	s.Equal(0, none.Total, "a genre no game has lists nothing")
}
