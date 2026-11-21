package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// GameCreateTestSuite tests creating a game through the secure core: the
// template is a game not yet created, and the first submission creates it.
type GameCreateTestSuite struct {
	testutil.BaseSuite
}

func TestGameCreateSuite(t *testing.T) {
	s := &GameCreateTestSuite{}
	s.SuiteName = "Game Create Tests"
	suite.Run(t, s)
}

// The template is what a new game starts with and what its creator may do.
func (s *GameCreateTestSuite) TestTheTemplateCarriesDefaultsAndRights() {
	alice := s.CreateUser("create-template")

	t := Must(alice.NewGame())
	s.Equal("adventure", t.Genre)
	s.Empty(t.Name)
	s.False(t.Public)
	s.True(t.HasContent)
	s.True(t.MayEditCard)
	s.True(t.MayEditContent)
	s.True(t.MayPublish, "a creator may publish their own game")
}

// Creating takes what was filled in; the rest keeps the template's values.
func (s *GameCreateTestSuite) TestCreateTakesWhatWasFilledIn() {
	alice := s.CreateUser("create-filled")

	game := Must(alice.CreateGameWith(testutil.GameCreate{
		Name:                  some("Filled In"),
		SystemMessageScenario: some("A quiet village."),
		Public:                some(true),
	}))
	s.Equal("Filled In", game.Name)
	s.True(game.Public)
	s.Equal("adventure", game.Genre, "the template's genre")
	s.Require().NotNil(game.SystemMessageScenario)
	s.Equal("A quiet village.", *game.SystemMessageScenario)
	s.True(game.MayEditCard)
}

// A game needs a name.
func (s *GameCreateTestSuite) TestAGameNeedsAName() {
	alice := s.CreateUser("create-noname")

	_, err := alice.CreateGameWith(testutil.GameCreate{Name: some("  ")})
	s.Refused(err, testutil.CodeValidation)
}

// A game made in a workshop belongs to it.
func (s *GameCreateTestSuite) TestAGameMadeInAWorkshopBelongsToIt() {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution("Create Org"))
	head := s.CreateUser("create-head")
	Must(head.AcceptInvite(Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID)).Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Create Workshop"))
	participant := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshop.Id.String())).InviteToken))

	game := Must(participant.CreateGameWith(testutil.GameCreate{Name: some("Workshop Game")}))
	s.Require().NotNil(game.WorkshopId)
	s.Equal(workshop.Id, *game.WorkshopId)
}
