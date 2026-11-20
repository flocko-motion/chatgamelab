// package: obj / game file
// type:    data
// job:     defines a game as a file: the fields a game file carries, under the names the file uses
// limits:  data shape only; reading a file into a game lives in db (-> db UpdateGameFromFile)
package obj

import "github.com/google/uuid"

// GameFile is a game as users download, share and upload it: its content
// without its access rights, statistics or generated first scene.
type GameFile struct {
	ID                     uuid.UUID `json:"id"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	SystemMessageScenario  string    `json:"system_message_scenario"`
	SystemMessageGameStart string    `json:"system_message_game_start"`
	ImageStyle             string    `json:"image_style"`
	CSS                    string    `json:"css"`
	StatusFields           string    `json:"status_fields"`
	// Genre is the engine genre; a file without one is an adventure.
	Genre string `json:"genre"`
	// Theme names the engine theme; without one the engine's default.
	Theme *string `json:"theme,omitempty"`
}

// NewGameFile is game as a file.
func NewGameFile(game Game) GameFile {
	return GameFile{
		ID:                     game.ID,
		Name:                   game.Name,
		Description:            game.Description,
		SystemMessageScenario:  game.SystemMessageScenario,
		SystemMessageGameStart: game.SystemMessageGameStart,
		ImageStyle:             game.ImageStyle,
		CSS:                    game.CSS,
		StatusFields:           game.StatusFields,
		Genre:                  game.Genre,
		Theme:                  game.Theme,
	}
}
