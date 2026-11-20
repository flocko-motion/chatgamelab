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
// a participant's workshop game.
func (s *GamePermissionsTestSuite) TestStaffCanDeleteWorkshopGame() {
	admin := s.DevUser()

	// Setup: institution + head + staff + workshop + participant
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
	participant := Must(s.JoinWorkshop(*invite.InviteToken))
	s.T().Logf("Participant joined: %s", participant.Name)

	// Participant creates a game (auto-assigned to workshop)
	game := Must(participant.UploadGame("alien-first-contact"))
	s.NotEmpty(game.Id)
	s.T().Logf("Participant created game: %s (ID: %s)", game.Name, game.Id)

	// Staff deletes the participant's workshop game — should succeed
	MustSucceed(staff.DeleteGame(game.Id.String()))
	s.T().Logf("Staff deleted participant's workshop game")

	// Game should no longer be visible
	games := Must(head.ListGames())
	for _, g := range games {
		s.NotEqual(game.Id, g.Id, "game should be deleted")
	}
	s.T().Logf("Game correctly deleted by staff")
}

// TestHeadCanDeleteWorkshopGame tests that a head of the org can delete
// a participant's workshop game.
func (s *GamePermissionsTestSuite) TestHeadCanDeleteWorkshopGame() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Head Game Org"))
	head := s.CreateUser("gp-head2")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Head Game Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	participant := Must(s.JoinWorkshop(*invite.InviteToken))

	game := Must(participant.UploadGame("alien-first-contact"))
	s.T().Logf("Participant created game: %s", game.Id)

	// Head deletes the participant's workshop game — should succeed
	MustSucceed(head.DeleteGame(game.Id.String()))
	s.T().Logf("Head deleted participant's workshop game")

	games := Must(head.ListGames())
	for _, g := range games {
		s.NotEqual(game.Id, g.Id, "game should be deleted")
	}
}

// TestParticipantCannotDeleteOtherParticipantGame tests that a participant
// cannot delete another participant's game in the same workshop.
func (s *GamePermissionsTestSuite) TestParticipantCannotDeleteOtherParticipantGame() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("Part Game Org"))
	head := s.CreateUser("gp-head3")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Part Game Workshop"))
	wsIDStr := workshop.Id.String()

	// Create two participants
	invite1 := Must(head.CreateWorkshopInvite(wsIDStr))
	participant1 := Must(s.JoinWorkshop(*invite1.InviteToken))

	invite2 := Must(head.CreateWorkshopInvite(wsIDStr))
	participant2 := Must(s.JoinWorkshop(*invite2.InviteToken))

	// Participant1 creates a game
	game := Must(participant1.UploadGame("alien-first-contact"))
	s.T().Logf("Participant1 created game: %s", game.Id)

	// Participant2 sees the game (same workshop) but may not delete it
	err := participant2.DeleteGame(game.Id.String())
	s.Forbidden(err, "participant should not be able to delete another participant's game")

	// Game should still exist (check via owner)
	games := Must(participant1.ListGames())
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

// TestWhoCanSetGamePublic tests who may switch a game's public flag on: for a
// workshop game its creator and whoever may edit it (head, staff, admin); for a
// game outside a workshop only its creator.
func (s *GamePermissionsTestSuite) TestWhoCanSetGamePublic() {
	admin := s.DevUser()

	// Setup: institution + head + workshop + participant
	inst := Must(admin.CreateInstitution("Public Flag Org"))
	head := s.CreateUser("pub-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Public Flag Workshop"))
	wsIDStr := workshop.Id.String()

	invite := Must(head.CreateWorkshopInvite(wsIDStr))
	participant := Must(s.JoinWorkshop(*invite.InviteToken))

	// Participant creates a game (private by default)
	game := Must(participant.UploadGame("alien-first-contact"))
	s.False(game.Public, "game should be private by default")
	s.T().Logf("Participant created game: %s (public=%v)", game.Id, game.Public)

	// Head sets public on a participant's workshop game.
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
	updated, err = participant.SetGamePublic(game.Id.String(), true)
	s.NoError(err, "creator should be able to publish their own game")
	s.True(updated.Public)

	// Another participant of the same workshop cannot touch it.
	other := Must(s.JoinWorkshop(*invite.InviteToken))
	Must(head.SetGamePublic(game.Id.String(), false))
	_, err = other.SetGamePublic(game.Id.String(), true)
	s.Forbidden(err, "another participant must not publish someone else's game")

	// Outside a workshop only the creator may publish, admins included.
	individual := s.CreateUser("pub-individual")
	personal := Must(individual.UploadGame("alien-first-contact"))
	s.Nil(personal.WorkshopId)
	_, err = admin.SetGamePublic(personal.Id.String(), true)
	s.Forbidden(err, "admin must not publish someone else's personal game")
	updated, err = individual.SetGamePublic(personal.Id.String(), true)
	s.NoError(err)
	s.True(updated.Public)
}

// TestWorkshopShareRequiresPublicGame tests that workshop shares can only
// be created for public games, even by head/staff.
func (s *GamePermissionsTestSuite) TestWorkshopShareRequiresPublicGame() {
	admin := s.DevUser()

	// Setup: institution + head + workshop + participant + API key
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
	participant := Must(s.JoinWorkshop(*invite.InviteToken))

	// Participant creates a private game
	game := Must(participant.UploadGame("alien-first-contact"))
	gameID := game.Id.String()
	s.T().Logf("Game created (public=%v): %s", game.Public, gameID)

	// Head tries to share the private game — should FAIL (non-owner, game not public)
	_, err := head.CreateWorkshopGameShare(gameID, wsIDStr, nil)
	s.Forbidden(err, "head should not be able to share a private game they don't own")
	s.T().Logf("Head correctly rejected from sharing private game: %v", err)

	// Participant (owner) can share their own private game — should SUCCEED
	share, err := participant.CreateWorkshopGameShare(gameID, wsIDStr, nil)
	s.NoError(err, "owner should be able to share their own private game")
	s.NotEmpty(share.Token)
	s.T().Logf("Owner shared own private game: token=%s", share.Token)

	// Participant makes game public, now head can share too
	Must(participant.SetGamePublic(gameID, true))

	// Head shares the now-public game — should SUCCEED
	headShare, err := head.CreateWorkshopGameShare(gameID, wsIDStr, nil)
	s.NoError(err, "head should be able to share a public game")
	s.NotEmpty(headShare.Token)
	s.T().Logf("Head shared public game: token=%s", headShare.Token)
}
