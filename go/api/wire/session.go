package wire

import (
	"time"

	"cgl/core"

	"github.com/google/uuid"
)

// SessionView is a session as its owner lists it. Field names follow v1's
// session; gameName is set in the caller's list of all their sessions.
type SessionView struct {
	ID            uuid.UUID  `json:"id"`
	GameID        uuid.UUID  `json:"gameId"`
	GameName      string     `json:"gameName,omitempty"`
	UserID        uuid.UUID  `json:"userId"`
	WorkshopID    *uuid.UUID `json:"workshopId,omitempty"`
	GameShareID   *uuid.UUID `json:"gameShareId,omitempty"`
	ApiKeyShareID *uuid.UUID `json:"apiKeyShareId,omitempty"`
	Platform      string     `json:"platform"`
	Language      string     `json:"language"`
	MessagesSent  int        `json:"messagesSent"`
	CreatedAt     time.Time  `json:"createdAt"`
	ModifiedAt    time.Time  `json:"modifiedAt"`
}

// SessionViewsOf converts a list of sessions, passing a loading error
// through.
func SessionViewsOf(sessions []core.SessionCard, err error) ([]SessionView, error) {
	if err != nil {
		return nil, err
	}
	views := make([]SessionView, len(sessions))
	for i, s := range sessions {
		views[i] = SessionView{
			ID: s.ID, GameID: s.GameID, GameName: s.GameName, UserID: s.UserID, WorkshopID: s.WorkshopID,
			GameShareID: s.GameShareID, ApiKeyShareID: s.KeyShareID, Platform: s.Platform, Language: s.Language,
			MessagesSent: s.MessagesSent, CreatedAt: s.CreatedAt, ModifiedAt: s.ModifiedAt,
		}
	}
	return views, nil
}
