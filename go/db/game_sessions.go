// package: db / database access and repository layer
// type:    data
// job:     stores the engine's sessions beside the platform's bookkeeping, and lists and deletes them.
// limits:  reacts to nothing it stores; launching, resuming and access belong to game and api.
package db

import (
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

// NewGuest is the account of a guest playing through a share link.
type NewGuest struct {
	Name    string
	ShareID uuid.UUID
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

	tx, err := sqlDb().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin session creation: %w", err)
	}
	defer tx.Rollback()
	q := queries().WithTx(tx)
	if g := l.meta.Guest; g != nil {
		if _, err := q.CreateGuestUser(ctx, CreateGuestUserParams{
			ID: l.meta.UserID, Name: userName(g.Name), PrivateShareID: uuid.NullUUID{UUID: g.ShareID, Valid: true},
		}); err != nil {
			return fmt.Errorf("create the guest: %w", err)
		}
	}
	if l.meta.ConsumesPlay && l.meta.GameShareID != nil {
		if _, err := q.DecrementGameShareRemaining(ctx, *l.meta.GameShareID); err != nil {
			return ErrNoPlaysLeft
		}
	}
	var replaced []uuid.UUID
	if l.meta.ReplaceEarlier {
		ids, err := q.GetUserGameSessionIDsForGame(ctx, GetUserGameSessionIDsForGameParams{UserID: l.meta.UserID, GameID: l.meta.GameID})
		if err != nil {
			return fmt.Errorf("find earlier sessions: %w", err)
		}
		for _, old := range ids {
			if err := q.DeleteGameSession(ctx, old); err != nil {
				return fmt.Errorf("replace earlier session: %w", err)
			}
		}
		replaced = ids
	}
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
