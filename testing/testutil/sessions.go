// package: testutil / the words for playing
// type:    logic
// job:     creating and playing sessions on the engine's routes as any caller does (an account's bearer token, a browser's session cookie, a guest without credentials), and the platform's lists of sessions
// limits:  session words only; the session stream and the WebSocket aren't spoken here
package testutil

import (
	"cgl/apiclient"

	"github.com/google/uuid"
)

// NewSession is what a session is created from: a game, or a share link.
type NewSession struct {
	game, share, language, once string
	hostPlayer, hostRequest     string
	mode                        string
}

// ForGame creates a session of a game.
func ForGame(gameID string) NewSession { return NewSession{game: gameID} }

// ForShare creates a session from a share link's token.
func ForShare(token string) NewSession { return NewSession{share: token} }

// FromHost names the host page's visitor and request, as the playback-engine
// embedded on a verified page sends them.
func (n NewSession) FromHost(player, request string) NewSession {
	n.hostPlayer, n.hostRequest = player, request
	return n
}

// InLanguage plays the session in a language.
func (n NewSession) InLanguage(language string) NewSession { n.language = language; return n }

// InMode asks to play live or in rounds, for a game whose genre plays both.
func (n NewSession) InMode(mode string) NewSession { n.mode = mode; return n }

// Once sends a one-time key; without one, every creation gets a fresh key.
func (n NewSession) Once(key string) NewSession { n.once = key; return n }

func (n NewSession) request() apiclient.CreateRequest {
	req := apiclient.CreateRequest{Once: n.once}
	if req.Once == "" {
		req.Once = uuid.NewString()
	}
	if n.game != "" {
		req.Game = &n.game
	}
	if n.share != "" {
		req.Share = &n.share
	}
	if n.language != "" {
		req.Language = &n.language
	}
	if n.hostPlayer != "" {
		req.Player = &n.hostPlayer
	}
	if n.hostRequest != "" {
		req.Request = &n.hostRequest
	}
	if n.mode != "" {
		mode := apiclient.Mode(n.mode)
		req.Mode = &mode
	}
	return req
}

// CreateSession creates a session, as the caller's playback-engine does, and
// answers its id.
func (c caller) CreateSession(n NewSession) (string, error) {
	resp, err := decode[apiclient.CreateResponse](c.api.PostApiEngineSessionsWithResponse(ctx(), n.request()))
	return resp.Id, err
}

// SessionState is a session's state.
func (c caller) SessionState(sessionID string) (Snapshot, error) {
	return decode[Snapshot](c.api.GetApiEngineSessionsIdStateWithResponse(ctx(), sessionID))
}

// SessionTopology is the shape of a session's game.
func (c caller) SessionTopology(sessionID string) (Topology, error) {
	return decode[Topology](c.api.GetApiEngineSessionsIdTopologyWithResponse(ctx(), sessionID))
}

// SessionHistory is what has happened in a session.
func (c caller) SessionHistory(sessionID string) ([]SessionEvent, error) {
	return decode[[]SessionEvent](c.api.GetApiEngineSessionsIdHistoryWithResponse(ctx(), sessionID))
}

// SessionJournal is what happened in a session, as its journal holds it.
func (c caller) SessionJournal(sessionID string) ([]JournalEntry, error) {
	return decode[[]JournalEntry](c.api.GetApiEngineSessionsIdJournalWithResponse(ctx(), sessionID))
}

// Say sends the player's text into a session.
func (c caller) Say(sessionID, text string) error {
	return done(c.api.PostApiEngineSessionsIdInputWithResponse(ctx(), sessionID, apiclient.InputRequest{Text: &text}))
}

// ListSessions is the caller's sessions.
func (c caller) ListSessions() ([]UserSession, error) {
	return decode[[]UserSession](c.api.GetApiSessionsWithResponse(ctx(), &apiclient.GetApiSessionsParams{}))
}

// GetGameSessions is the caller's own sessions of a game.
func (c caller) GetGameSessions(gameID string) ([]GameSession, error) {
	return decode[[]GameSession](c.api.GetApiGamesIdSessionsWithResponse(ctx(), gameID))
}

// DeleteSession deletes one of the caller's sessions.
func (c caller) DeleteSession(sessionID string) error {
	return done(c.api.DeleteApiSessionsIdWithResponse(ctx(), sessionID))
}
