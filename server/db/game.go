package db

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"gorm.io/gorm"
	"strings"
	"webapp-server/obj"
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
