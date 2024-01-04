package db

import (
	"gorm.io/gorm"
	"webapp-server/obj"
)

type Game struct {
	gorm.Model
	Title               string `json:"title"`
	TitleImage          []byte
	Description         string `json:"description"`
	Scenario            string `json:"scenario"`
	SessionStartSyscall string `json:"sessionStartSyscall"`
	PostActionSyscall   string `json:"postActionSyscall"`
	ImageStyle          string `json:"imageStyle"`
	UserID              uint   `json:"-"`
	User                User   `json:"user" gorm:"foreignKey:UserID"`
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

func (game *Game) ToObjGame() *obj.Game {
	return &obj.Game{
		ID:                  game.ID,
		Title:               game.Title,
		Description:         game.Description,
		Scenario:            game.Scenario,
		SessionStartSyscall: game.SessionStartSyscall,
		PostActionSyscall:   game.PostActionSyscall,
		ImageStyle:          game.ImageStyle,
		User:                game.User.ToObjUser(),
	}
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
