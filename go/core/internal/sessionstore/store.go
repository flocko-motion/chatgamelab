// Package sessionstore keeps the engine's sessions on the platform's
// tables: the engine.Persist the core hands a session it let start or
// resume (core.LaunchStore, core.ResumeSession). It stores what it is given
// and decides nothing; who may play is the core's.
package sessionstore

import (
	"cgl/core/internal/store"
	"cgl/db"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"engine"

	"github.com/google/uuid"
)

// SessionStore keeps the engine's sessions on the platform's tables: it is the
// engine.Persist the platform hands every session. It stores what it is given
// and reacts to none of it.
type SessionStore struct{}

var _ engine.Persist = SessionStore{}

// NewSession is the bookkeeping of a session about to launch, and what its
// start takes besides: a guest's account, a play of its share link, the
// player's earlier sessions of the game. All of it happens with the first
// save, in one transaction, or not at all.
type NewSession struct {
	GameID        uuid.UUID
	UserID        uuid.UUID
	WorkshopID    *uuid.UUID
	GameShareID   *uuid.UUID
	ApiKeyShareID *uuid.UUID
	Platform      string
	Language      string

	Guest          *NewGuest // the account made for a guest playing through a link (UserID)
	ConsumesPlay   bool      // takes a play from the share link (GameShareID)
	ReplaceEarlier bool      // deletes the player's earlier sessions of the game
}

// NewGuest is the account of a guest playing through a share link. A guest
// with an OutsideID (a verified page's visitor) is the one guest that ID
// already is on the link, if there is one.
type NewGuest struct {
	Name      string
	ShareID   uuid.UUID
	OutsideID string
}

// ErrNoPlaysLeft: the share link has no plays left.
var ErrNoPlaysLeft = errors.New("the share link has no plays left")

// Launching is the store for a session about to launch. The engine saves a
// session's first state before Launch returns and only once it has validated
// it, so that first save creates the row and counts the game's play in one
// transaction: a launch the engine refuses leaves nothing behind.
func Launching(meta NewSession) engine.Persist {
	return &launchStore{meta: meta}
}

type launchStore struct {
	SessionStore
	mu       sync.Mutex
	meta     NewSession
	created  bool
	replaced []uuid.UUID
}

// Replaced are the earlier sessions the launch deleted.
func Replaced(store engine.Persist) []uuid.UUID {
	if l, ok := store.(*launchStore); ok {
		l.mu.Lock()
		defer l.mu.Unlock()
		return l.replaced
	}
	return nil
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

	var replaced []uuid.UUID
	err = store.InTx(ctx, func(q *db.Queries) error {
		if g := l.meta.Guest; g != nil && g.OutsideID != "" {
			known, err := q.V2OutsideGuest(ctx, db.V2OutsideGuestParams{
				ID: l.meta.UserID, Name: sql.NullString{String: g.Name, Valid: true},
				PrivateShareID: uuid.NullUUID{UUID: g.ShareID, Valid: true}, OutsideID: g.OutsideID,
			})
			if err != nil {
				return fmt.Errorf("find the guest: %w", err)
			}
			l.meta.UserID = known
		} else if g != nil {
			if _, err := q.V2CreateGuestUser(ctx, db.V2CreateGuestUserParams{
				ID: l.meta.UserID, Name: sql.NullString{String: g.Name, Valid: true}, PrivateShareID: uuid.NullUUID{UUID: g.ShareID, Valid: true},
			}); err != nil {
				return fmt.Errorf("create the guest: %w", err)
			}
		}
		if l.meta.ConsumesPlay && l.meta.GameShareID != nil {
			if _, err := q.V2DecrementGameShareRemaining(ctx, *l.meta.GameShareID); err != nil {
				return ErrNoPlaysLeft
			}
		}
		if l.meta.ReplaceEarlier {
			ids, err := q.V2GetUserGameSessionIDsForGame(ctx, db.V2GetUserGameSessionIDsForGameParams{UserID: l.meta.UserID, GameID: l.meta.GameID})
			if err != nil {
				return fmt.Errorf("find earlier sessions: %w", err)
			}
			for _, old := range ids {
				if err := q.V2DeleteGameSession(ctx, old); err != nil {
					return fmt.Errorf("replace earlier session: %w", err)
				}
			}
			replaced = ids
		}
		if err := q.V2CreateGameSession(ctx, db.V2CreateGameSessionParams{
			ID:            id,
			GameID:        l.meta.GameID,
			UserID:        l.meta.UserID,
			WorkshopID:    nullUUID(l.meta.WorkshopID),
			GameShareID:   nullUUID(l.meta.GameShareID),
			ApiKeyShareID: nullUUID(l.meta.ApiKeyShareID),
			Platform:      l.meta.Platform,
			Language:      l.meta.Language,
			SchemaVersion: int32(stored.Version),
			Spec:          spec,
			State:         state,
		}); err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		// Every created session is a play.
		if err := q.V2IncrementGamePlayCount(ctx, l.meta.GameID); err != nil {
			return fmt.Errorf("count the play: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	l.created = true
	l.replaced = replaced
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
	if err := store.Queries().V2AppendGameSessionEvent(ctx, db.V2AppendGameSessionEventParams{
		SessionID: id,
		Stream:    e.Stream,
		Value:     e.Value,
	}); err != nil {
		return fmt.Errorf("store event: %w", err)
	}
	return store.Queries().V2TouchGameSession(ctx, id)
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
	n, err := store.Queries().V2SaveGameSessionState(ctx, db.V2SaveGameSessionStateParams{
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
	row, err := store.Queries().V2GetGameSessionByID(ctx, id)
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
	rows, err := store.Queries().V2GetGameSessionEvents(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}
	events := make([]engine.SessionEvent, 0, len(rows))
	for _, r := range rows {
		events = append(events, engine.SessionEvent{Stream: r.Stream, Value: r.Value})
	}
	return events, nil
}

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}
