// package: db / database access and repository layer
// type:    data
// job:     stores the engine's sessions beside the platform's bookkeeping, and lists and deletes them.
// limits:  reacts to nothing it stores; launching, resuming and access belong to game and api.
package db

import (
	"cgl/obj"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"engine"

	"github.com/google/uuid"
)

// SessionStore keeps the engine's sessions on the platform's tables: it is the
// engine.Persist the platform hands every session. It stores what it is given
// and reacts to none of it.
type SessionStore struct{}

var _ engine.Persist = SessionStore{}

// NewSession is the bookkeeping of a session about to launch.
type NewSession struct {
	GameID        uuid.UUID
	UserID        uuid.UUID
	WorkshopID    *uuid.UUID
	GameShareID   *uuid.UUID
	ApiKeyShareID *uuid.UUID
	Platform      string
	Language      string
}

// Launching is the store for a session about to launch. The engine saves a
// session's first state before Launch returns and only once it has validated
// it, so that first save creates the row and counts the game's play in one
// transaction: a launch the engine refuses leaves nothing behind.
func Launching(meta NewSession) engine.Persist {
	return &launchStore{meta: meta}
}

type launchStore struct {
	SessionStore
	mu      sync.Mutex
	meta    NewSession
	created bool
}

func (l *launchStore) SaveState(ctx context.Context, sessionID string, stored engine.Stored) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.created {
		return l.SessionStore.SaveState(ctx, sessionID, stored)
	}
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("session id %q: %w", sessionID, err)
	}
	spec, state, err := marshalStored(stored)
	if err != nil {
		return err
	}

	tx, err := sqlDb().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin session creation: %w", err)
	}
	defer tx.Rollback()
	q := queries().WithTx(tx)
	if err := q.CreateGameSession(ctx, CreateGameSessionParams{
		ID:            id,
		GameID:        l.meta.GameID,
		UserID:        l.meta.UserID,
		WorkshopID:    uuidPtrToNullUUID(l.meta.WorkshopID),
		GameShareID:   uuidPtrToNullUUID(l.meta.GameShareID),
		ApiKeyShareID: uuidPtrToNullUUID(l.meta.ApiKeyShareID),
		Platform:      l.meta.Platform,
		Language:      l.meta.Language,
		SchemaVersion: int32(stored.Version),
		Spec:          spec,
		State:         state,
	}); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	// Every created session is a play.
	if err := q.IncrementGamePlayCount(ctx, l.meta.GameID); err != nil {
		return fmt.Errorf("count the play: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit session creation: %w", err)
	}
	l.created = true
	return nil
}

func marshalStored(stored engine.Stored) (spec, state json.RawMessage, err error) {
	if spec, err = json.Marshal(stored.Spec); err != nil {
		return nil, nil, fmt.Errorf("encode spec: %w", err)
	}
	if state, err = json.Marshal(stored.State); err != nil {
		return nil, nil, fmt.Errorf("encode state: %w", err)
	}
	return spec, state, nil
}

// unknownSession is the engine's refusal for a session the store does not hold.
func unknownSession(sessionID string) error {
	return engine.Refuse(engine.ProblemUnknown, fmt.Sprintf("there is no session %q", sessionID))
}

func (SessionStore) SaveEvent(ctx context.Context, sessionID string, e engine.SessionEvent) error {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return unknownSession(sessionID)
	}
	if err := queries().AppendGameSessionEvent(ctx, AppendGameSessionEventParams{
		SessionID: id,
		Stream:    e.Stream,
		Value:     e.Value,
	}); err != nil {
		return fmt.Errorf("store event: %w", err)
	}
	return queries().TouchGameSession(ctx, id)
}

func (SessionStore) SaveState(ctx context.Context, sessionID string, stored engine.Stored) error {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return unknownSession(sessionID)
	}
	spec, state, err := marshalStored(stored)
	if err != nil {
		return err
	}
	n, err := queries().SaveGameSessionState(ctx, SaveGameSessionStateParams{
		ID:            id,
		SchemaVersion: int32(stored.Version),
		Spec:          spec,
		State:         state,
	})
	if err != nil {
		return fmt.Errorf("store state: %w", err)
	}
	if n == 0 {
		return unknownSession(sessionID)
	}
	return nil
}

func (SessionStore) LoadState(ctx context.Context, sessionID string) (engine.Stored, error) {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return engine.Stored{}, unknownSession(sessionID)
	}
	row, err := queries().GetGameSessionByID(ctx, id)
	if err != nil {
		return engine.Stored{}, unknownSession(sessionID)
	}
	stored := engine.Stored{Version: int(row.SchemaVersion)}
	// A spec or state the engine can no longer read is a stale session.
	if err := json.Unmarshal(row.Spec, &stored.Spec); err != nil {
		return engine.Stored{}, engine.Refuse(engine.ProblemStale, "the stored spec no longer reads: "+err.Error())
	}
	if err := json.Unmarshal(row.State, &stored.State); err != nil {
		return engine.Stored{}, engine.Refuse(engine.ProblemStale, "the stored state no longer reads: "+err.Error())
	}
	return stored, nil
}

func (SessionStore) LoadHistory(ctx context.Context, sessionID string) ([]engine.SessionEvent, error) {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, unknownSession(sessionID)
	}
	rows, err := queries().GetGameSessionEvents(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}
	events := make([]engine.SessionEvent, 0, len(rows))
	for _, r := range rows {
		events = append(events, engine.SessionEvent{Stream: r.Stream, Value: r.Value})
	}
	return events, nil
}

// ErrSessionNotFound is a session the platform does not hold.
var ErrSessionNotFound = errors.New("session not found")

// GetGameSession is a session's bookkeeping, with no permission check: the
// caller decides who may reach it.
func GetGameSession(ctx context.Context, sessionID uuid.UUID) (*obj.GameSession, error) {
	row, err := queries().GetGameSessionByID(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	return &obj.GameSession{
		ID:            row.ID,
		GameID:        row.GameID,
		UserID:        row.UserID,
		WorkshopID:    nullUUIDToPtr(row.WorkshopID),
		GameShareID:   nullUUIDToPtr(row.GameShareID),
		ApiKeyShareID: nullUUIDToPtr(row.ApiKeyShareID),
		Platform:      row.Platform,
		Language:      row.Language,
		MessagesSent:  int(row.MessagesSent),
		CreatedAt:     row.CreatedAt,
		ModifiedAt:    row.ModifiedAt,
	}, nil
}

// CountSessionMessage counts one input the player sent.
func CountSessionMessage(ctx context.Context, sessionID uuid.UUID) error {
	return queries().CountGameSessionMessage(ctx, sessionID)
}

// UserSessionWithGame is a session with its game's name, for the player's list.
type UserSessionWithGame struct {
	obj.GameSession
	GameName string `json:"gameName"`
}

// GetUserSessionsFilters narrows the player's list.
type GetUserSessionsFilters struct {
	Search    string // part of the game's name
	SortField string // game, model (the AI platform), lastPlayed (default)
}

// GetGameSessionsByUserID is a player's most recent sessions with their games' names.
func GetGameSessionsByUserID(ctx context.Context, userID uuid.UUID, filters GetUserSessionsFilters) ([]UserSessionWithGame, error) {
	rows, err := queries().GetUserGameSessions(ctx, GetUserGameSessionsParams{
		UserID: userID,
		Search: filters.Search,
		SortBy: filters.SortField,
	})
	if err != nil {
		return nil, obj.ErrServerError("failed to get user sessions")
	}
	sessions := make([]UserSessionWithGame, 0, len(rows))
	for _, r := range rows {
		sessions = append(sessions, UserSessionWithGame{
			GameSession: obj.GameSession{
				ID:           r.ID,
				GameID:       r.GameID,
				UserID:       r.UserID,
				WorkshopID:   nullUUIDToPtr(r.WorkshopID),
				Platform:     r.Platform,
				Language:     r.Language,
				MessagesSent: int(r.MessagesSent),
				CreatedAt:    r.CreatedAt,
				ModifiedAt:   r.ModifiedAt,
			},
			GameName: r.GameName,
		})
	}
	return sessions, nil
}

// GetGameSessionsByGameID lists the player's own sessions of a game they may read.
func GetGameSessionsByGameID(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) ([]obj.GameSession, error) {
	game, err := loadGameByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if err := canAccessGame(ctx, userID, OpRead, game, nil); err != nil {
		return nil, err
	}
	rows, err := queries().GetGameSessionsByGameID(ctx, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	sessions := []obj.GameSession{}
	for _, r := range rows {
		// A session id plays the session, so only the player's own are listed.
		if r.UserID != userID {
			continue
		}
		sessions = append(sessions, obj.GameSession{
			ID:           r.ID,
			GameID:       r.GameID,
			UserID:       r.UserID,
			WorkshopID:   nullUUIDToPtr(r.WorkshopID),
			Platform:     r.Platform,
			Language:     r.Language,
			MessagesSent: int(r.MessagesSent),
			CreatedAt:    r.CreatedAt,
			ModifiedAt:   r.ModifiedAt,
		})
	}
	return sessions, nil
}

// DeleteGameSession deletes a session and its events; userID must own it.
func DeleteGameSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	session, err := GetGameSession(ctx, sessionID)
	if err != nil {
		return obj.ErrNotFound("session not found")
	}
	if session.UserID != userID {
		return obj.ErrForbidden("not the owner of this session")
	}
	return queries().DeleteGameSession(ctx, sessionID)
}

// DeleteUserGameSessions deletes a player's sessions of one game, as starting
// the game anew does, and answers their ids.
func DeleteUserGameSessions(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := queries().GetUserGameSessionIDsForGame(ctx, GetUserGameSessionIDsForGameParams{UserID: userID, GameID: gameID})
	if err != nil {
		return nil, fmt.Errorf("failed to find sessions: %w", err)
	}
	for _, id := range ids {
		if err := queries().DeleteGameSession(ctx, id); err != nil {
			return nil, fmt.Errorf("failed to delete session: %w", err)
		}
	}
	return ids, nil
}

// purgeBatchSize caps how many sessions one DELETE statement touches, keeping
// lock duration and WAL volume bounded on installations with a large backlog.
const purgeBatchSize = 200

// PurgeExpiredSessions deletes every session idle since cutoff, with its
// events, and answers their ids.
func PurgeExpiredSessions(ctx context.Context, cutoff time.Time) ([]uuid.UUID, error) {
	var purged []uuid.UUID
	for {
		ids, err := queries().PurgeExpiredGameSessions(ctx, PurgeExpiredGameSessionsParams{
			ModifiedAt: cutoff,
			Limit:      purgeBatchSize,
		})
		if err != nil {
			return purged, fmt.Errorf("failed to purge expired sessions: %w", err)
		}
		purged = append(purged, ids...)
		if len(ids) < purgeBatchSize {
			return purged, nil
		}
		if err := ctx.Err(); err != nil {
			return purged, err
		}
	}
}
