package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GamePermissionsTestSuite tests game deletion cleanup and workshop game permissions.
type GamePermissionsTestSuite struct {
	testutil.BaseSuite
}

func TestGamePermissionsSuite(t *testing.T) {
	s := &GamePermissionsTestSuite{}
	s.SuiteName = "Game Permissions Tests"
	suite.Run(t, s)
}

// TestStaffCanDeleteWorkshopGame tests that a staff member of the org can delete
// a guest's workshop game.
func (s *GamePermissionsTestSuite) TestStaffCanDeleteWorkshopGame() {
	admin := s.DevUser()

	// Setup: institution + head + staff + workshop + guest
	inst := Must(admin.CreateInstitution("Game Perm Org"))
	head := s.CreateUser("gp-head")
	staff := s.CreateUser("gp-staff")

	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))
	staffInvite := Must(head.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Game Perm Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))
	s.T().Logf("Guest joined: %s", guest.Name)

	// Guest creates a game (auto-assigned to workshop)
	game := Must(guest.UploadGame("alien-first-contact"))
	s.NotEmpty(game.Id)
	s.T().Logf("Guest created game: %s (ID: %s)", game.Name, game.Id)

	// Staff deletes the guest's workshop game — should succeed
	MustSucceed(staff.DeleteGame(game.Id.String()))
	s.T().Logf("Staff deleted guest's workshop game")

	// Game should no longer be visible
	games := Must(head.ListGames())
	for _, g := range games {
		s.NotEqual(game.Id, g.Id, "game should be deleted")
	}
	s.T().Logf("Game correctly deleted by staff")
}

// TestHeadCanDeleteWorkshopGame tests that a head of the org can delete
// a guest's workshop game.
func (s *GamePermissionsTestSuite) TestHeadCanDeleteWorkshopGame() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Head Game Org"))
	head := s.CreateUser("gp-head2")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Head Game Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))

	game := Must(guest.UploadGame("alien-first-contact"))
	s.T().Logf("Guest created game: %s", game.Id)

	// Head deletes the guest's workshop game — should succeed
	MustSucceed(head.DeleteGame(game.Id.String()))
	s.T().Logf("Head deleted guest's workshop game")

	games := Must(head.ListGames())
	for _, g := range games {
		s.NotEqual(game.Id, g.Id, "game should be deleted")
	}
}

// TestGuestCannotDeleteOtherGuestGame tests that a guest
// cannot delete another guest's game in the same workshop.
func (s *GamePermissionsTestSuite) TestGuestCannotDeleteOtherGuestGame() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Part Game Org"))
	head := s.CreateUser("gp-head3")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Part Game Workshop"))
	wsIDStr := workshop.Id.String()

	// Create two guests
	invite1 := Must(head.CreateWorkshopInvite(wsIDStr))
	guest1 := Must(s.JoinWorkshop(*invite1.InviteToken))

	invite2 := Must(head.CreateWorkshopInvite(wsIDStr))
	guest2 := Must(s.JoinWorkshop(*invite2.InviteToken))

	// Guest1 creates a game
	game := Must(guest1.UploadGame("alien-first-contact"))
	s.T().Logf("Guest1 created game: %s", game.Id)

	// Guest2 sees the game (same workshop) but may not delete it
	err := guest2.DeleteGame(game.Id.String())
	s.Forbidden(err, "guest should not be able to delete another guest's game")

	// Game should still exist (check via owner)
	games := Must(guest1.ListGames())
	found := false
	for _, g := range games {
		if g.Id == game.Id {
			found = true
			break
		}
	}
	s.True(found, "game should still exist after failed deletion")
}

// TestAdminCanDeleteAnyGame tests that an admin can delete any user's game,
// including games outside any workshop context.
func (s *GamePermissionsTestSuite) TestAdminCanDeleteAnyGame() {
	admin := s.DevUser()

	// Regular user creates a private game (no workshop)
	user := s.CreateUser("admin-del-target")
	game := Must(user.UploadGame("alien-first-contact"))
	s.T().Logf("User created game: %s (ID: %s)", game.Name, game.Id)

	// Admin deletes the user's game — should succeed
	MustSucceed(admin.DeleteGame(game.Id.String()))
	s.T().Logf("Admin deleted another user's game successfully")

	// Game should be gone
	games := Must(user.ListGames())
	for _, g := range games {
		s.NotEqual(game.Id, g.Id, "game should be deleted by admin")
	}
	s.T().Logf("Game correctly deleted by admin")
}

// TestAdminCanUpdateAnyGame tests that an admin can update any user's game.
func (s *GamePermissionsTestSuite) TestAdminCanUpdateAnyGame() {
	admin := s.DevUser()

	// Regular user creates a game
	user := s.CreateUser("admin-upd-target")
	game := Must(user.UploadGame("alien-first-contact"))
	s.T().Logf("User created game: %s (ID: %s)", game.Name, game.Id)

	// Admin updates the game's name — should succeed
	updatedName := game.Name + " (admin-edited)"
	updated, err := admin.RenameGame(game.Id.String(), updatedName)
	s.NoError(err, "admin should be able to update any game")
	s.Equal(updatedName, updated.Name, "game name should be updated")
	s.T().Logf("Admin updated game name to: %s", updated.Name)
}

// TestAdminCanReadAnyPrivateGame tests that an admin can fetch any private game by ID.
func (s *GamePermissionsTestSuite) TestAdminCanReadAnyPrivateGame() {
	admin := s.DevUser()

	// Regular user creates a private game
	user := s.CreateUser("admin-read-target")
	game := Must(user.UploadGame("alien-first-contact"))
	s.T().Logf("User created private game: %s (ID: %s)", game.Name, game.Id)

	// Admin reads the game by ID — should succeed
	fetched, err := admin.GetGameByID(game.Id.String())
	s.NoError(err, "admin should be able to read any private game")
	s.Equal(game.Id, fetched.Id, "fetched game ID should match")
	s.T().Logf("Admin read private game: %s", fetched.Name)
}

// TestAdminSeesAllGamesInList tests that an admin sees all platform games
// (not just their own) when listing games with the default filter.
func (s *GamePermissionsTestSuite) TestAdminSeesAllGamesInList() {
	admin := s.DevUser()

	// Create two unrelated users each with a private game
	user1 := s.CreateUser("admin-list-u1")
	user2 := s.CreateUser("admin-list-u2")

	game1 := Must(user1.UploadGame("alien-first-contact"))
	game2 := Must(user2.UploadGame("alien-first-contact"))
	s.T().Logf("User1 game: %s, User2 game: %s", game1.Id, game2.Id)

	// Admin lists all games — should see both
	adminGames := Must(admin.ListGames())
	s.True(containsGame(adminGames, game1.Id), "admin should see user1's private game in list")
	s.True(containsGame(adminGames, game2.Id), "admin should see user2's private game in list")
	s.T().Logf("Admin sees %d games — includes other users' private games", len(adminGames))

	// Non-admin user cannot see each other's private games
	u1Games := Must(user1.ListGames())
	s.False(containsGame(u1Games, game2.Id), "regular user should not see another user's private game")
	s.T().Logf("User1 sees %d games — correctly cannot see user2's game", len(u1Games))
}

// TestStaffCannotDeleteNonWorkshopGame tests that a staff member cannot delete
// a game that doesn't belong to their org's workshop (e.g. an individual's personal game).
func (s *GamePermissionsTestSuite) TestStaffCannotDeleteNonWorkshopGame() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Non WS Game Org"))
	staff := s.CreateUser("gp-staff2")
	staffInvite := Must(admin.InviteToInstitution(inst.Id.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.Id.String()))

	// Create an individual user (not in any workshop) with a game
	individual := s.CreateUser("gp-individual")
	game := Must(individual.UploadGame("alien-first-contact"))
	s.T().Logf("Individual created game: %s (not in any workshop)", game.Id)

	// Staff can't see the individual's private game, so for them it doesn't exist
	err := staff.DeleteGame(game.Id.String())
	s.NotFound(err, "staff should not be able to delete a non-workshop game")
}

// TestOtherUserCannotAccessNonPublicGame tests that a user cannot fetch
// another user's non-public game by ID (outside any workshop context).
func (s *GamePermissionsTestSuite) TestOtherUserCannotAccessNonPublicGame() {
	owner := s.CreateUser("gp-owner")
	other := s.CreateUser("gp-other")

	game := Must(owner.UploadGame("alien-first-contact"))
	s.T().Logf("Owner created game: %s (ID: %s)", game.Name, game.Id)

	// Other user tries to fetch the game by ID: for them it doesn't exist
	_, err := other.GetGameByID(game.Id.String())
	s.NotFound(err, "other user should not be able to access a non-public game")
}

// TestOtherUserCannotSeeNonPublicGameInList tests that a non-public game
// does not appear in another user's game list.
func (s *GamePermissionsTestSuite) TestOtherUserCannotSeeNonPublicGameInList() {
	owner := s.CreateUser("gp-owner2")
	other := s.CreateUser("gp-other2")

	game := Must(owner.UploadGame("alien-first-contact"))
	s.T().Logf("Owner created game: %s (ID: %s)", game.Name, game.Id)

	// Other user lists games — should not see the owner's private game
	games := Must(other.ListGames())
	for _, g := range games {
		s.NotEqual(game.Id, g.Id, "other user should not see a non-public game in their list")
	}
	s.T().Logf("Other user sees %d games, none of which is the private game", len(games))
}

// TestOwnerCanAccessOwnNonPublicGame verifies the owner CAN access their
// own non-public game (sanity check).
func (s *GamePermissionsTestSuite) TestOwnerCanAccessOwnNonPublicGame() {
	owner := s.CreateUser("gp-owner3")

	game := Must(owner.UploadGame("alien-first-contact"))

	fetched, err := owner.GetGameByID(game.Id.String())
	s.NoError(err, "owner should be able to access their own game")
	s.Equal(game.Id, fetched.Id)
	s.T().Logf("Owner accessed own game: %s", fetched.Name)
}

// TestWhoCanSetGamePublic tests who may switch a game's public flag on: its
// creator, head and staff for their organisation's workshop games, and the
// admin for any game, being liable for what is public.
func (s *GamePermissionsTestSuite) TestWhoCanSetGamePublic() {
	admin := s.DevUser()

	// Setup: institution + head + workshop + guest
	inst := Must(admin.CreateInstitution("Public Flag Org"))
	head := s.CreateUser("pub-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Public Flag Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))

	// Guest creates a game (private by default)
	game := Must(guest.UploadGame("alien-first-contact"))
	s.False(game.Public, "game should be private by default")
	s.T().Logf("Guest created game: %s (public=%v)", game.Id, game.Public)

	// Head sets public on a guest's workshop game.
	updated, err := head.SetGamePublic(game.Id.String(), true)
	s.NoError(err, "head should be able to publish a workshop game")
	s.True(updated.Public)

	updated, err = head.SetGamePublic(game.Id.String(), false)
	s.NoError(err, "head should be able to unpublish a workshop game")
	s.False(updated.Public)

	// Admin sets it too.
	updated, err = admin.SetGamePublic(game.Id.String(), true)
	s.NoError(err, "admin should be able to publish a workshop game")
	s.True(updated.Public)
	Must(head.SetGamePublic(game.Id.String(), false))

	// The creator sets it.
	updated, err = guest.SetGamePublic(game.Id.String(), true)
	s.NoError(err, "creator should be able to publish their own game")
	s.True(updated.Public)

	// Another guest of the same workshop cannot touch it.
	other := Must(s.JoinWorkshop(*invite.InviteToken))
	Must(head.SetGamePublic(game.Id.String(), false))
	_, err = other.SetGamePublic(game.Id.String(), true)
	s.Forbidden(err, "another guest must not publish someone else's game")

	// Outside a workshop: the creator, and the admin.
	individual := s.CreateUser("pub-individual")
	personal := Must(individual.UploadGame("alien-first-contact"))
	s.Nil(personal.WorkshopId)
	updated, err = individual.SetGamePublic(personal.Id.String(), true)
	s.NoError(err)
	s.True(updated.Public)
	Must(individual.SetGamePublic(personal.Id.String(), false))
	updated, err = admin.SetGamePublic(personal.Id.String(), true)
	s.NoError(err, "the admin may publish any game")
	s.True(updated.Public)
}

// TestWorkshopShareRequiresPublicGame tests that someone else in the
// workshop shares only public games; the designer shares their own, and the
// organisation's head and staff share its games (#564).
func (s *GamePermissionsTestSuite) TestWorkshopShareRequiresPublicGame() {
	admin := s.DevUser()

	// Setup: institution + head + workshop + guest + API key
	inst := Must(admin.CreateInstitution("WS Share Public Org"))
	head := s.CreateUser("wspub-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	keyShare := Must(head.AddApiKey("mock-wspub", "WS Pub Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), "WS Share Public Workshop"))
	wsIDStr := workshop.Id.String()
	orgShareIDStr := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))
	Must(head.UpdateWorkshop(wsIDStr, testutil.WorkshopUpdate{
		Name:             "WS Share Public Workshop",
		Active:           true,
		AllowGameSharing: true,
	}))

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	guest := Must(s.JoinWorkshop(*invite.InviteToken))
	other := Must(s.JoinWorkshop(*invite.InviteToken))

	// Guest creates a private game
	game := Must(guest.UploadGame("alien-first-contact"))
	gameID := game.Id.String()
	s.T().Logf("Game created (public=%v): %s", game.Public, gameID)

	// Another guest tries to share the private game — should FAIL (not theirs, not public)
	_, err := other.CreateWorkshopGameShare(gameID, wsIDStr, nil)
	s.Forbidden(err, "another guest should not be able to share a private game they don't own")

	// The head shares it: a game of the organisation's workshop is the organisation's
	_, err = head.CreateWorkshopGameShare(gameID, wsIDStr, nil)
	s.NoError(err, "head shares the organisation's games")

	// Guest (owner) can share their own private game — should SUCCEED
	share, err := guest.CreateWorkshopGameShare(gameID, wsIDStr, nil)
	s.NoError(err, "owner should be able to share their own private game")
	s.NotEmpty(share.Token)
	s.T().Logf("Owner shared own private game: token=%s", share.Token)

	// Guest makes game public, now the other guest can share it too
	Must(guest.SetGamePublic(gameID, true))

	otherShare, err := other.CreateWorkshopGameShare(gameID, wsIDStr, nil)
	s.NoError(err, "anyone in the workshop should be able to share a public game")
	s.NotEmpty(otherShare.Token)
}
