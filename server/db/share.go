package db

import "gorm.io/gorm"

type Share struct {
	gorm.Model
	GameID uint
	Game   Game
	Hash   string
	Edit   bool
	Secret bool
}
