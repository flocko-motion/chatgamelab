package db

import (
	"gorm.io/gorm"
	"webapp-server/obj"
)

type Session struct {
	gorm.Model
	GameID                uint
	Game                  Game
	UserID                *uint
	User                  User
	AssistantID           string
	AssistantInstructions string
	ThreadID              string
	Hash                  string
}

func (session *Session) export() *obj.Session {
	return &obj.Session{
		ID:                    session.ID,
		GameID:                session.GameID,
		UserID:                *session.UserID,
		AssistantID:           session.AssistantID,
		AssistantInstructions: session.AssistantInstructions,
		ThreadID:              session.ThreadID,
		Hash:                  session.Hash,
	}
}

func GetSessionByHash(hash string) (*obj.Session, error) {
	var session Session
	err := db.Where("hash = ?", hash).First(&session).Error
	return session.export(), err
}

func CreateSession(session *obj.Session) (*obj.Session, error) {
	userId := session.UserID
	sessionDb := Session{
		GameID:                session.GameID,
		UserID:                &userId,
		AssistantID:           session.AssistantID,
		AssistantInstructions: session.AssistantInstructions,
		ThreadID:              session.ThreadID,
		Hash:                  generateHash(),
	}
	err := db.Create(&sessionDb).Error
	return sessionDb.export(), err
}
