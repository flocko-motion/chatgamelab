package db

type Session struct {
	GameID      uint
	Game        Game
	UserID      *uint
	User        User
	Hash        string
	AssistantID string
	ThreadID    string
}

func GetSessionByHash(hash string) (*Session, error) {
	var session Session
	// Preload Game when retrieving the session
	err := db.Preload("Game").Preload("User").Where("hash = ?", hash).First(&session).Error
	return &session, err
}
