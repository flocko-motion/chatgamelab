package db

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"log"
	"net/http"
	"webapp-server/obj"
)

type User struct {
	gorm.Model
	Auth0ID           string `json:"-" gorm:"uniqueIndex"` // Unique identifier from Auth0
	Email             string `json:"-"`
	Name              string `json:"name"`
	OpenAiKeyPublish  string `json:"openaiKeyPublish"`
	OpenAiKeyPersonal string `json:"openaiKeyPersonal"`
	Games             []Game
}

// CreateUser creates a new user in the database
func CreateUser(user *User) error {
	return db.Create(user).Error
}

// GetUserByID gets a user by ID
func GetUserByID(id uint) (*User, error) {
	var user User
	err := db.First(&user, id).Error
	return &user, err
}

// GetUserByAuth0ID gets a user by Auth0 ID
func GetUserByAuth0ID(auth0ID string) (*User, error) {
	var user User
	err := db.Where("auth0_id = ?", auth0ID).First(&user).Error
	return &user, err
}

// DeleteUser deletes a user
func DeleteUser(id uint) error {
	return db.Delete(&User{}, id).Error
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

	return game.update()
}

func (user *User) Export() *obj.User {
	return &obj.User{
		ID:                user.ID,
		Name:              user.Name,
		OpenAiKeyPersonal: shortenOpenaiKey(user.OpenAiKeyPersonal),
		OpenAiKeyPublish:  shortenOpenaiKey(user.OpenAiKeyPublish),
	}
}

func shortenOpenaiKey(key string) string {
	if len(key) != 51 {
		return ""
	}
	return "sk-..." + key[47:51]
}

func (user *User) Update(name string, email string) {
	user.Name = name
	user.Email = email
	db.Save(user)
}

func (user *User) UpdateApiKeyPublish(publish string) {
	user.OpenAiKeyPublish = publish
	db.Save(user)
}

func (user *User) UpdateApiKeyPersonal(personal string) {
	user.OpenAiKeyPersonal = personal
	db.Save(user)
}

func (user *User) GetApiKey(session *obj.Session, game *obj.Game) (*string, error) {
	if session == nil || game == nil || session.GameID != game.ID {
		return nil, fmt.Errorf("invalid parameters for fetchin api key")
	}

	// TODO: fetch key dependent on play type
	key := user.OpenAiKeyPersonal

	return &key, nil

}
