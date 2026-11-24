package core

import (
	"context"

	"cgl/functional"
	"cgl/obj"
	"engine"

	"github.com/google/uuid"
)

// ImportGameFile replaces a game's card and content with a game file's and
// returns the game as stored. It is an update (UpdateGame) that changes every
// field the file holds, with the same rights; the file's id is ignored. A
// file without a genre is an adventure, as every game was before genres; one
// without a theme takes the engine's default.
func ImportGameFile(ctx context.Context, id uuid.UUID, file obj.GameFile) (*GameCard, error) {
	genre := file.Genre
	if genre == "" {
		genre = string(engine.GenreAdventure)
	}
	theme := ""
	if file.Theme != nil {
		theme = *file.Theme
	}
	statusFields := functional.NormalizeJson(file.StatusFields, &[]obj.StatusField{})
	css := functional.NormalizeJson(file.CSS, &obj.CSS{})

	return UpdateGame(ctx, id, GameChanges{
		Name:         &file.Name,
		Description:  &file.Description,
		Genre:        &genre,
		Theme:        &theme,
		Scenario:     &file.SystemMessageScenario,
		GameStart:    &file.SystemMessageGameStart,
		ImageStyle:   &file.ImageStyle,
		VoiceStyle:   &file.VoiceStyle,
		CSS:          &css,
		StatusFields: &statusFields,
	})
}
