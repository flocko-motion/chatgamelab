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

func (user *User) GetGames() ([]Game, *obj.HTTPError) {
	var games []Game
	err := db.Model(&user).Association("Games").Find(&games)
	if err != nil {
		return nil, obj.ErrorToHTTPError(http.StatusInternalServerError, err)
	}
	for i := range games {
		if games[i].User.Name == "" {
			games[i].User.Name = fmt.Sprintf("user_%d", games[i].UserID)
		}
	}
	return games, nil
}

func (user *User) GetGame(id uint) (*obj.Game, *obj.HTTPError) {
	var game Game
	err := db.Where("id = ?", id).First(&game).Error
	if err != nil {
		return nil, obj.ErrorToHTTPError(http.StatusInternalServerError, err)
	}
	if game.UserID != user.ID {
		return nil, obj.NewHTTPErrorf(http.StatusUnauthorized, "unauthorized")
	}
	// convert to obj.Game

	return game.ToObjGame(), nil
}

func (user *User) CreateGame(game *Game) error {
	return db.Model(&user).Association("Games").Append(game)
}

func (user *User) UpdateGame(updatedGame obj.Game) error {
	// Check that the game belongs to the user
	game, err := user.GetGame(updatedGame.ID)
	if err != nil {
		return err
	}
	// remove the user from the game
	updatedGame.User = obj.User{}
	// Update the game
	return db.Save(game).Error
}

func (user *User) ToObjUser() obj.User {
	return obj.User{
		ID:   user.ID,
		Name: user.Name,
	}
}
