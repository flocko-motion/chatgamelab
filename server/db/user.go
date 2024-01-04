package db

import (
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"webapp-server/obj"
)

type User struct {
	gorm.Model
	Auth0ID string `json:"-" gorm:"uniqueIndex"` // Unique identifier from Auth0
	Email   string `json:"-"`
	Name    string `json:"name"`
	Games   []Game
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

// UpdateUser updates an existing user
func UpdateUser(user *User) error {
	return db.Save(user).Error
}

// DeleteUser deletes a user
func DeleteUser(id uint) error {
	return db.Delete(&User{}, id).Error
}

func (user *User) GetGames() ([]obj.Game, *obj.HTTPError) {
	var games []Game
	err := db.Model(&user).Association("Games").Find(&games)
	if err != nil {
		return nil, obj.ErrorToHTTPError(http.StatusInternalServerError, err)
	}
	gamesObj := make([]obj.Game, len(games))
	for i := range games {
		if games[i].User.Name == "" {
			games[i].User.Name = fmt.Sprintf("user_%d", games[i].UserID)
		}
		gamesObj[i] = *games[i].ToObjGame()
	}
	return gamesObj, nil
}

// GetGame gets a game by ID, formatted for external use
func (user *User) GetGame(id uint) (*obj.Game, *obj.HTTPError) {
	game, err := user.getGame(id)
	if err != nil {
		return nil, err
	}
	return game.ToObjGame(), nil
}

// getGame is for internal use only
func (user *User) getGame(id uint) (*Game, *obj.HTTPError) {
	var game Game
	err := db.Where("id = ?", id).First(&game).Error
	if err != nil {
		return nil, obj.ErrorToHTTPError(http.StatusInternalServerError, err)
	}
	if game.UserID != user.ID {
		return nil, obj.NewHTTPErrorf(http.StatusUnauthorized, "unauthorized")
	}
	return &game, nil
}

func (user *User) CreateGame(game obj.Game) error {
	return db.Model(&user).Association("Games").Append(game)
}

func (user *User) UpdateGame(updatedGame obj.Game) error {
	game, err := user.getGame(updatedGame.ID)
	if err != nil {
		return err
	}

	game.Title = updatedGame.Title
	game.Description = updatedGame.Description
	game.Scenario = updatedGame.Scenario
	game.SessionStartSyscall = updatedGame.SessionStartSyscall
	game.PostActionSyscall = updatedGame.PostActionSyscall
	game.ImageStyle = updatedGame.ImageStyle
	game.SharePlayActive = updatedGame.SharePlayActive
	game.ShareEditActive = updatedGame.ShareEditActive

	return game.update()
}

func (user *User) ToObjUser() *obj.User {
	return &obj.User{
		ID:   user.ID,
		Name: user.Name,
	}
}
