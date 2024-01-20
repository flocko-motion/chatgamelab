package db

import (
	"gorm.io/gorm"
	"net/http"
	"webapp-server/lang"
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

type Chapter struct {
	gorm.Model
	SessionID   uint
	Session     Session
	Chapter     uint
	Input       string
	Output      string
	ImagePrompt string
	Image       []byte
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

func (chapter *Chapter) export() *obj.Chapter {
	return &obj.Chapter{
		SessionID:   chapter.SessionID,
		Chapter:     chapter.Chapter,
		Input:       chapter.Input,
		Output:      chapter.Output,
		ImagePrompt: chapter.ImagePrompt,
		Image:       chapter.Image,
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

func AddChapter(sessionId, chapterId uint, input, output, imagePrompt string) (*Chapter, error) {
	chapterDb := Chapter{
		SessionID:   sessionId,
		Chapter:     chapterId,
		Input:       input,
		Output:      output,
		ImagePrompt: imagePrompt,
		Image:       []byte{},
	}
	err := db.Create(&chapterDb).Error
	if err != nil {
		return nil, err
	}
	return &chapterDb, nil
}

func GetChapter(sessionId, chapterId uint) (*obj.Chapter, error) {
	var chapter Chapter
	err := db.Where("session_id = ? AND chapter = ?", sessionId, chapterId).First(&chapter).Error
	if err != nil {
		return nil, err
	}
	return chapter.export(), nil
}

func SetImage(sessionId, chapterId uint, image []byte) *obj.HTTPError {
	var chapter Chapter
	err := db.Where("session_id = ? AND chapter = ?", sessionId, chapterId).First(&chapter).Error
	if err != nil {
		return &obj.HTTPError{StatusCode: http.StatusNotFound, Message: lang.ErrorFailedLoadingGameData}
	}
	chapter.Image = image
	if err = db.Save(&chapter).Error; err != nil {
		return &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: lang.ErrorFailedUpdatingGameData}
	}
	return nil
}
