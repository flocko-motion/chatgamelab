package db

import (
	"gorm.io/gorm"
)

type Game struct {
	gorm.Model
	Title               string
	TitleImage          []byte
	Description         string
	Scenario            string
	SessionStartSyscall string
	PostActionSyscall   string
	ImageStyle          string
	UserID              uint
	User                User
}

// CreateGame creates a new game in the database
func CreateGame(game *Game) error {
	return db.Create(game).Error
}

// GetGameByID gets a game by ID
func GetGameByID(id uint) (*Game, error) {
	var game Game
	err := db.First(&game, id).Error
	return &game, err
}

func (game *Game) Update() error {
	return db.Save(game).Error
}

func (game *Game) GetShares() ([]Share, error) {
	var shares []Share
	err := db.Model(&game).Association("Shares").Find(&shares)
	return shares, err
}

func (game *Game) CreateSession(user *User) (*Session, error) {
	session := Session{
		GameID: game.ID,
		UserID: &user.ID,
		Hash:   generateHash(),
	}
	err := db.Create(&session).Error
	return &session, err
}
