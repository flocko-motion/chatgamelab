package db

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"webapp-server/obj"

	"gorm.io/gorm"
)

type Game struct {
	gorm.Model
	Title               string `json:"title"`
	TitleImage          []byte
	Description         string `json:"description"`
	Scenario            string `json:"scenario"`
	SessionStartSyscall string `json:"sessionStartSyscall"`
	ImageStyle          string `json:"imageStyle"`
	StatusFields        string `json:"statusProperties"`
	SharePlayActive     bool   `json:"sharePlayActive"`
	SharePlayHash       string `json:"sharePlayHash"`
	ShareEditActive     bool   `json:"shareEditActive"`
	ShareEditHash       string `json:"shareEditHash"`
	UserID              uint   `json:"-"`
	User                User   `json:"user" gorm:"foreignKey:UserID"`
}

func (user *User) GetGames() ([]obj.Game, *obj.HTTPError) {
	var games []Game
	err := db.Preload("User").Model(&user).Association("Games").Find(&games)
	if err != nil {
		return nil, obj.ErrorToHTTPError(http.StatusInternalServerError, err)
	}
	gamesObj := make([]obj.Game, len(games))
	for i := range games {
		if games[i].User.Name == "" {
			games[i].User.Name = fmt.Sprintf("user_%d", games[i].UserID)
		}
		gamesObj[i] = *games[i].Export()
	}
	return gamesObj, nil
}

// GetGame gets a game by ID, formatted for external use
func (user *User) GetGame(id uint) (*obj.Game, *obj.HTTPError) {
	log.Printf("Getting game %d from db", id)
	game, err := user.getGame(id)
	if err != nil {
		return nil, err
	}
	log.Printf("Got game %d from db", id)
	return game.Export(), nil
}

// getGame is for internal use only
func (user *User) getGame(id uint) (*Game, *obj.HTTPError) {
	var game Game
	err := db.Preload("User").Where("id = ?", id).First(&game).Error
	if err != nil {
		return nil, obj.ErrorToHTTPError(http.StatusInternalServerError, err)
	}
	if game.UserID != user.ID {
		return nil, obj.NewHTTPErrorf(http.StatusUnauthorized, "unauthorized")
	}
	return &game, nil
}

func (user *User) DeleteGame(gameId uint) *obj.HTTPError {
	// assert access rights
	game, httpErr := user.getGame(gameId)
	if httpErr != nil {
		return httpErr
	}
	if game.UserID != user.ID {
		return obj.NewHTTPErrorf(http.StatusUnauthorized, "access denied - this game is owned by another user")
	}

	// Perform the deletion
	err := db.Delete(&game).Error
	if err != nil {
		return obj.ErrorToHTTPError(http.StatusInternalServerError, err)
	}

	return nil
}

func (user *User) CreateGame(game *obj.Game) error {
	statusFieldsSerialized, _ := json.Marshal(game.StatusFields)
	gameDb := &Game{
		Title:               game.Title,
		StatusFields:        string(statusFieldsSerialized),
		Description:         "This is a new game.",
		Scenario:            "An adventure in a fantasy world. The player must find a way out of a castle.",
		SessionStartSyscall: "Introduce the player to the game and write the first scene.",
		ImageStyle:          "illustration, watercolor, fantastic",
		SharePlayHash:       randomHash(),
	}
	if err := db.Model(&user).Association("Games").Append(gameDb); err != nil {
		return err
	}
	game.ID = gameDb.ID
	return nil
}

func (user *User) UpdateGame(updatedGame obj.Game) error {
	game, err := user.getGame(updatedGame.ID)
	if err != nil {
		return err
	}

	statusFieldsSerialized, _ := json.Marshal(updatedGame.StatusFields)

	game.Title = updatedGame.Title
	game.Description = updatedGame.Description
	game.Scenario = updatedGame.Scenario
	game.SessionStartSyscall = updatedGame.SessionStartSyscall
	game.StatusFields = string(statusFieldsSerialized)
	game.ImageStyle = updatedGame.ImageStyle
	game.SharePlayActive = updatedGame.SharePlayActive
	game.ShareEditActive = updatedGame.ShareEditActive

	if game.SharePlayHash == "" {
		game.SharePlayHash = randomHash()
	}

	return game.update()
}

// CreateGame creates a new game in the database
func CreateGame(game *Game) error {
	return db.Create(game).Error
}

// GetGameByID gets a game by ID
func GetGameByID(id uint) (*obj.Game, error) {
	var game Game
	err := db.First(&game, id).Error
	return game.Export(), err
}

func GetGameByPublicHash(hash string) (*obj.Game, *obj.HTTPError) {
	var game Game
	err := db.Where("share_play_hash = ?", hash).Where("share_play_active = ?", true).First(&game).Error
	if err != nil {
		return nil, &obj.HTTPError{StatusCode: 404, Message: "Game not found"}
	}
	return game.Export(), nil
}

func (game *Game) update() error {
	game.SharePlayHash = strings.TrimSpace(game.SharePlayHash)
	if game.SharePlayHash == "" {
		game.SharePlayHash = randomHash()
	}
	return db.Save(game).Error
}

func (game *Game) Export() *obj.Game {
	var statusFields []obj.StatusField
	if err := json.Unmarshal([]byte(game.StatusFields), &statusFields); err != nil {
		statusFields = []obj.StatusField{}
	}
	return &obj.Game{
		ID:                  game.ID,
		Title:               game.Title,
		Description:         game.Description,
		Scenario:            game.Scenario,
		SessionStartSyscall: game.SessionStartSyscall,
		StatusFields:        statusFields,
		ImageStyle:          game.ImageStyle,
		SharePlayActive:     game.SharePlayActive,
		SharePlayHash:       game.SharePlayHash,
		ShareEditActive:     game.ShareEditActive,
		ShareEditHash:       game.ShareEditHash,
		UserId:              game.UserID,
		UserName:            game.User.Name,
	}
}

func randomHash() string {
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	return enc.EncodeToString(randomBytes)
}
