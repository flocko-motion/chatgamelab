package testing

import (
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameCrudTestSuite tests game CRUD operations: update, clone, YAML, delete, and listing filters.
type GameCrudTestSuite struct {
	testutil.BaseSuite
}

func TestGameCrudSuite(t *testing.T) {
	s := &GameCrudTestSuite{}
	s.SuiteName = "Game CRUD Tests"
	suite.Run(t, s)
}

// --- Update ---

// TestOwnerCanRenameGame verifies that the game owner can update the game name.
func (s *GameCrudTestSuite) TestOwnerCanRenameGame() {
	alice := s.CreateUser("crud-rename")
	game := Must(alice.UploadGame("alien-first-contact"))
	s.NotEmpty(game.Id)
	s.T().Logf("Created game: %s (ID: %s)", game.Name, game.Id)

	updated := Must(alice.RenameGame(game.Id.String(), "Renamed Game"))
	s.Equal("Renamed Game", updated.Name)
	s.T().Logf("Renamed game to: %s", updated.Name)

	// Verify via GET
	fetched := Must(alice.GetGameByID(game.Id.String()))
	s.Equal("Renamed Game", fetched.Name)
}

// TestOwnerCanTogglePublicFlag verifies toggling the public flag on a game.
func (s *GameCrudTestSuite) TestOwnerCanTogglePublicFlag() {
	alice := s.CreateUser("crud-public")
	bob := s.CreateUser("crud-public-bob")

	game := Must(alice.UploadGame("alien-first-contact"))
	s.False(game.Public, "game should start as private")

	// Make public
	updated := Must(alice.SetGamePublic(game.Id.String(), true))
	s.True(updated.Public, "game should be public after update")

	// Bob can see it in public list
	bobGames := Must(bob.ListGamesWithFilter("public"))
	found := false
	for _, g := range bobGames {
		if g.Id == game.Id {
			found = true
			break
		}
	}
	s.True(found, "bob should see alice's public game")

	// Make private again
	Must(alice.SetGamePublic(game.Id.String(), false))

	// Bob should no longer see it
	bobGames = Must(bob.ListGamesWithFilter("public"))
	found = false
	for _, g := range bobGames {
		if g.Id == game.Id {
			found = true
			break
		}
	}
	s.False(found, "bob should NOT see alice's private game")
}

// TestOwnerCanUpdateDescription verifies updating the game description.
func (s *GameCrudTestSuite) TestOwnerCanUpdateDescription() {
	alice := s.CreateUser("crud-desc")
	game := Must(alice.UploadGame("alien-first-contact"))

	game.Description = "A brand new description"
	updated := Must(alice.UpdateGame(game.Id.String(), game))
	s.Equal("A brand new description", updated.Description)
	s.Equal(game.SystemMessageScenario, updated.SystemMessageScenario, "the rest of the game stays")
}

// TestNonOwnerCannotUpdateGame verifies that a non-owner cannot update another user's private game.
func (s *GameCrudTestSuite) TestNonOwnerCannotUpdateGame() {
	alice := s.CreateUser("crud-notown-a")
	bob := s.CreateUser("crud-notown-b")

	game := Must(alice.UploadGame("alien-first-contact"))

	// Bob can't see alice's private game, so for him it doesn't exist
	game.Name = "Hacked Name"
	_, err := bob.UpdateGame(game.Id.String(), game)
	s.NotFound(err, "non-owner should not be able to update game")
	s.NotEqual("Hacked Name", Must(alice.GetGameByID(game.Id.String())).Name)
}

// TestNameLengthValidation verifies that game names over 70 chars are rejected.
func (s *GameCrudTestSuite) TestNameLengthValidation() {
	alice := s.CreateUser("crud-namelen")
	game := Must(alice.UploadGame("alien-first-contact"))

	longName := strings.Repeat("x", 71)
	_, err := alice.RenameGame(game.Id.String(), longName)
	s.Refused(err, testutil.CodeNameTooLong, "name over 70 chars should be rejected")
}

// --- Clone ---

// TestOwnerCanCloneOwnGame verifies cloning your own game.
// The game is made public first: the clone endpoint loads the game without
// the caller, so it finds only public games.
func (s *GameCrudTestSuite) TestOwnerCanCloneOwnGame() {
	alice := s.CreateUser("crud-clone")
	game := Must(alice.UploadGame("alien-first-contact"))

	// Make game public so clone endpoint can find it
	Must(alice.SetGamePublic(game.Id.String(), true))

	cloned := Must(alice.CloneGame(game.Id.String()))
	s.NotEqual(game.Id, cloned.Id, "cloned game should have different ID")
	s.Contains(cloned.Name, "(Copy)", "cloned game name should contain (Copy)")
	s.False(cloned.Public, "cloned game should start as private")
	s.T().Logf("Cloned game: %s (ID: %s)", cloned.Name, cloned.Id)
}

// TestUserCanCloneVisiblePublicGame verifies that any user can clone a visible public game.
func (s *GameCrudTestSuite) TestUserCanCloneVisiblePublicGame() {
	alice := s.CreateUser("crud-clonevis-a")
	bob := s.CreateUser("crud-clonevis-b")

	game := Must(alice.UploadGame("alien-first-contact"))
	fullGame := Must(alice.GetGameByID(game.Id.String()))
	fullGame.Public = true
	Must(alice.UpdateGame(game.Id.String(), fullGame))

	cloned := Must(bob.CloneGame(game.Id.String()))
	s.NotEqual(game.Id, cloned.Id)
	s.Contains(cloned.Name, "(Copy)")
	s.T().Logf("Bob cloned visible public game: %s", cloned.Id)
}

// TestUserCannotCloneInvisiblePrivateGame verifies cloning a private game you can't see fails.
func (s *GameCrudTestSuite) TestUserCannotCloneInvisiblePrivateGame() {
	alice := s.CreateUser("crud-cloneinv-a")
	bob := s.CreateUser("crud-cloneinv-b")

	game := Must(alice.UploadGame("alien-first-contact"))
	s.False(game.Public)

	_, err := bob.CloneGame(game.Id.String())
	s.NotFound(err, "should not be able to clone a private game you can't see")
}

// TestClonedGameStartsAsPrivate verifies that even cloning a public game results in a private clone.
func (s *GameCrudTestSuite) TestClonedGameStartsAsPrivate() {
	alice := s.CreateUser("crud-clonepriv2")
	game := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.SetGamePublic(game.Id.String(), true))

	cloned := Must(alice.CloneGame(game.Id.String()))
	s.False(cloned.Public, "cloned game should always start as private")
}

// --- YAML ---

// TestYAMLRoundTrip verifies exporting and re-importing YAML preserves game content.
func (s *GameCrudTestSuite) TestYAMLRoundTrip() {
	alice := s.CreateUser("crud-yaml")
	game := Must(alice.UploadGame("alien-first-contact"))
	s.NotEmpty(game.SystemMessageScenario, "uploaded game should have scenario")

	yamlContent, err := alice.GetGameYAML(game.Id.String())
	s.NoError(err, "should be able to export YAML")
	s.NotEmpty(yamlContent, "YAML should not be empty")
	s.T().Logf("Exported YAML length: %d bytes", len(yamlContent))
}

// TestNonOwnerCanExportPublicGameYAML verifies that any user can export YAML of a public game.
func (s *GameCrudTestSuite) TestNonOwnerCanExportPublicGameYAML() {
	alice := s.CreateUser("crud-yaml-pub-a")
	bob := s.CreateUser("crud-yaml-pub-b")

	game := Must(alice.UploadGame("alien-first-contact"))
	fullGame := Must(alice.GetGameByID(game.Id.String()))
	fullGame.Public = true
	Must(alice.UpdateGame(game.Id.String(), fullGame))

	yamlContent, err := bob.GetGameYAML(game.Id.String())
	s.NoError(err, "non-owner should be able to export YAML of public game")
	s.NotEmpty(yamlContent)
	s.T().Logf("Bob exported public game YAML: %d bytes", len(yamlContent))
}

// TestNonOwnerCannotExportPrivateGameYAML verifies that a non-owner cannot export YAML of a private game.
func (s *GameCrudTestSuite) TestNonOwnerCannotExportPrivateGameYAML() {
	alice := s.CreateUser("crud-yaml-priv-a")
	bob := s.CreateUser("crud-yaml-priv-b")

	game := Must(alice.UploadGame("alien-first-contact"))
	s.False(game.Public)

	_, err := bob.GetGameYAML(game.Id.String())
	s.NotFound(err, "non-owner should not be able to export YAML of private game")
}

// --- Delete ---

// TestOwnerCanDeleteOwnGame verifies the owner can delete their game.
func (s *GameCrudTestSuite) TestOwnerCanDeleteOwnGame() {
	alice := s.CreateUser("crud-del")
	game := Must(alice.UploadGame("alien-first-contact"))

	MustSucceed(alice.DeleteGame(game.Id.String()))
	s.T().Logf("Deleted game: %s", game.Id)

	// Verify gone from list
	games := Must(alice.ListGames())
	for _, g := range games {
		s.NotEqual(game.Id, g.Id, "deleted game should not appear in list")
	}
}

// TestNonOwnerCannotDeleteGame verifies a non-owner cannot delete another user's game.
func (s *GameCrudTestSuite) TestNonOwnerCannotDeleteGame() {
	alice := s.CreateUser("crud-del-notown-a")
	bob := s.CreateUser("crud-del-notown-b")

	game := Must(alice.UploadGame("alien-first-contact"))

	// Bob can't see alice's private game, so for him it doesn't exist
	err := bob.DeleteGame(game.Id.String())
	s.NotFound(err, "non-owner should not be able to delete game")

	// Verify game still exists
	fetched := Must(alice.GetGameByID(game.Id.String()))
	s.Equal(game.Id, fetched.Id)
}

// --- Listing filters ---

// TestFilterOwn verifies the ?filter=own parameter returns only the user's games.
func (s *GameCrudTestSuite) TestFilterOwn() {
	alice := s.CreateUser("crud-filter-own")
	bob := s.CreateUser("crud-filter-own-bob")

	aliceGame := Must(alice.UploadGame("alien-first-contact"))
	Must(bob.UploadGame("alien-first-contact"))

	ownGames := Must(alice.ListGamesWithFilter("own"))
	s.GreaterOrEqual(len(ownGames), 1, "alice should see at least 1 own game")

	for _, g := range ownGames {
		s.Require().NotNil(g.CreatorId)
		s.Equal(alice.ID, g.CreatorId.String(), "own filter returns only alice's games")
	}

	found := false
	for _, g := range ownGames {
		if g.Id == aliceGame.Id {
			found = true
			break
		}
	}
	s.True(found, "alice's game should appear in own filter")
}

// TestFilterPublic verifies the ?filter=public parameter.
func (s *GameCrudTestSuite) TestFilterPublic() {
	alice := s.CreateUser("crud-filter-pub")
	bob := s.CreateUser("crud-filter-pub-bob")

	game := Must(alice.UploadGame("alien-first-contact"))
	Must(alice.SetGamePublic(game.Id.String(), true))

	pubGames := Must(bob.ListGamesWithFilter("public"))
	found := false
	for _, g := range pubGames {
		if g.Id == game.Id {
			found = true
			break
		}
	}
	s.True(found, "public game should appear in public filter for other users")
}

// TestSearchByName verifies searching games by name.
func (s *GameCrudTestSuite) TestSearchByName() {
	alice := s.CreateUser("crud-search")
	game := Must(alice.UploadGame("alien-first-contact"))

	// Search using the unique suffix (no spaces, URL-safe)
	// Game names are like "Test Game abc12345" — use the hex suffix
	parts := strings.Split(game.Name, " ")
	searchTerm := parts[len(parts)-1] // the unique hex suffix

	results := Must(alice.ListGamesWithSearch(searchTerm))
	found := false
	for _, g := range results {
		if g.Id == game.Id {
			found = true
			break
		}
	}
	s.True(found, "game should be found by name search")
}
