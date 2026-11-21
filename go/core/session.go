package core

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/legal"
	"cgl/log"

	"engine"

	"github.com/google/uuid"
)

// A session is a game being played. An account's session is its own: only
// it lists, plays and deletes it. A guest who came through a share link has
// an account made for the play, and plays its session by the session's id:
// whoever holds the id plays it.
//
// A session goes on only while its player may still play it: on every turn
// and every resume the guard checks the game (still there, still playable
// by its player), the organisation it belongs to (not suspended) and its
// workshop (not paused). Its insights (prompts, graph) are part of playing:
// they take reading the game's content, which a share link gives for its
// game.

// SessionCard is a session as its owner lists it.
type SessionCard struct {
	ID           uuid.UUID
	GameID       uuid.UUID
	GameName     string // set in the caller's list
	UserID       uuid.UUID
	WorkshopID   *uuid.UUID
	GameShareID  *uuid.UUID
	KeyShareID   *uuid.UUID // the share that paid at creation
	Platform     string
	Language     string
	MessagesSent int
	CreatedAt    time.Time
	ModifiedAt   time.Time
}

// MySessions are the caller's sessions, newest activity first, at most 20:
// {List, Session, Own}. search narrows them to games whose name holds it;
// sortBy "game" or "model" orders them by game name or platform.
func MySessions(ctx context.Context, search, sortBy string) ([]SessionCard, error) {
	me := account(ctx)
	if !me.Valid || !granted(ctx, rules.List, rules.Session, rules.Own) {
		return []SessionCard{}, nil
	}
	rows, err := store.Queries().GetUserGameSessions(ctx, db.GetUserGameSessionsParams{
		UserID: me.UUID, Search: search, SortBy: sortBy,
	})
	if err != nil {
		return nil, err
	}
	sessions := make([]SessionCard, len(rows))
	for i, r := range rows {
		sessions[i] = SessionCard{
			ID: r.ID, GameID: r.GameID, GameName: r.GameName, UserID: r.UserID, WorkshopID: ptrUUID(r.WorkshopID),
			Platform: r.Platform, Language: r.Language, MessagesSent: int(r.MessagesSent),
			CreatedAt: r.CreatedAt, ModifiedAt: r.ModifiedAt,
		}
	}
	return sessions, nil
}

// MyGameSessions are the caller's sessions of a game they may read; one they
// may not does not exist for them (ErrNotFound).
func MyGameSessions(ctx context.Context, game uuid.UUID) ([]SessionCard, error) {
	if _, err := ReadGameCard(ctx, game); err != nil {
		return nil, err
	}
	me := account(ctx)
	if !me.Valid || !granted(ctx, rules.List, rules.Session, rules.Own) {
		return []SessionCard{}, nil
	}
	rows, err := store.Queries().V2OwnGameSessions(ctx, db.V2OwnGameSessionsParams{UserID: me.UUID, GameID: game})
	if err != nil {
		return nil, err
	}
	sessions := make([]SessionCard, len(rows))
	for i, r := range rows {
		sessions[i] = SessionCard{
			ID: r.ID, GameID: r.GameID, UserID: r.UserID, WorkshopID: ptrUUID(r.WorkshopID),
			GameShareID: ptrUUID(r.GameShareID), KeyShareID: ptrUUID(r.ApiKeyShareID),
			Platform: r.Platform, Language: r.Language, MessagesSent: int(r.MessagesSent),
			CreatedAt: r.CreatedAt, ModifiedAt: r.ModifiedAt,
		}
	}
	return sessions, nil
}

// DeleteSession deletes one of the caller's sessions: {Delete, Session,
// Own}. Someone else's does not exist for them (ErrNotFound).
func DeleteSession(ctx context.Context, id uuid.UUID) error {
	owner, err := store.Queries().V2SessionOwner(ctx, id)
	me := account(ctx)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (!me.Valid || owner != me.UUID)) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !granted(ctx, rules.Delete, rules.Session, rules.Own) {
		return forbidden("you may not delete this session")
	}
	if err := store.Queries().V2DeleteSession(ctx, id); err != nil {
		return err
	}
	log.Info("session deleted", "session_id", id, "by", me.UUID)
	return nil
}

// SessionUse is what a request does with a session: play it, or look into
// it (its prompts, graph and nodes).
type SessionUse int

const (
	PlaySession SessionUse = iota
	InspectSession
)

// OpenSession is the session guard: whether the caller may play session id
// now, or look into it, as an engine refusal the playback-engine tells the
// player in its own words. Unknown: there is no such session. Denied: it is
// someone else's, or its player may no longer play it. Paused: its workshop
// is paused.
func OpenSession(ctx context.Context, id uuid.UUID, use SessionUse) error {
	declaration := legal.CurrentDeclaration()
	f, err := store.Queries().V2SessionFacts(ctx, db.V2SessionFactsParams{
		ID: id, DeclarationVersion: declaration.Version, DeclarationHash: declaration.Hash,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return engine.Refuse(engine.ProblemUnknown, "there is no session "+id.String())
	}
	if err != nil {
		return err
	}
	if f.GameGone {
		return engine.Refuse(engine.ProblemUnknown, "the game of this session is gone")
	}
	if f.LinkGuest {
		// played by its id, for its link's game
		if !granted(ctx, rules.Play, rules.Session, rules.ViaGuestSessionID) {
			return engine.Refuse(engine.ProblemDenied, "you may not play this session")
		}
		if f.OrganisationSuspended {
			return engine.Refuse(engine.ProblemDenied, "the organisation of this game is suspended")
		}
		if use == InspectSession && !granted(ctx, rules.Read, rules.GameContent, rules.ViaShareLink) {
			return engine.Refuse(engine.ProblemDenied, "you may not look into this game")
		}
	} else {
		me := account(ctx)
		if !me.Valid || me.UUID != f.UserID || !granted(ctx, rules.Play, rules.Session, rules.Own) {
			return engine.Refuse(engine.ProblemDenied, "this session belongs to a signed-in account")
		}
		// the owner may still play its game: still in its workshop, their
		// organisation not suspended
		// a session started through a share link plays by the link while it
		// exists: {Play, GameCard, ViaShareLink}
		viaLink := f.GameShareID.Valid
		game, err := ReadGameCard(ctx, f.GameID)
		mayPlay := err == nil && game.MayPlay()
		if !mayPlay && !(viaLink && granted(ctx, rules.Play, rules.GameCard, rules.ViaShareLink)) {
			return engine.Refuse(engine.ProblemDenied, "you may no longer play this game")
		}
		mayInspect := (err == nil && game.mayReadContent) ||
			(viaLink && granted(ctx, rules.Read, rules.GameContent, rules.ViaShareLink))
		if use == InspectSession && !mayInspect {
			return engine.Refuse(engine.ProblemDenied, "you may not look into this game")
		}
	}
	if f.WorkshopPaused {
		return engine.Refuse(engine.ProblemPaused, "the workshop is paused")
	}
	return nil
}

// MayStartPlaying refuses a new session of a game the caller may not play
// ({Play, GameCard}), and any while the workshop they are in is paused. A
// guest through a share link starts by the link (MayStartPlayingByLink).
func MayStartPlaying(ctx context.Context, game uuid.UUID) error {
	card, err := ReadGameCard(ctx, game)
	if err != nil || !card.MayPlay() {
		return engine.Refuse(engine.ProblemDenied, "this game cannot be played here")
	}
	return checkNotPaused(ctx, from(ctx).CurrentWorkshop())
}

// MayStartPlayingByLink refuses a new session through a share link while
// the link's workshop, if any, is paused.
func MayStartPlayingByLink(ctx context.Context, workshop *uuid.UUID) error {
	return checkNotPaused(ctx, workshop)
}

func checkNotPaused(ctx context.Context, workshop *uuid.UUID) error {
	if workshop == nil {
		return nil
	}
	paused, err := store.Queries().V2WorkshopPaused(ctx, *workshop)
	if err != nil {
		return err
	}
	if paused {
		return engine.Refuse(engine.ProblemPaused, "the workshop is paused")
	}
	return nil
}

// CountMessage counts a player's message in a session the guard let them
// play (OpenSession).
func CountMessage(ctx context.Context, id uuid.UUID) {
	if err := store.Queries().CountGameSessionMessage(ctx, id); err != nil {
		log.Warn("failed to count a message", "session_id", id, "error", err)
	}
}


// MayTuneRuleFrequency is a flag for the playback-engine: whether the caller
// may change, while playing, how often a session repeats the
// youth-protection rule. Head and staff may ({Edit, SessionRule, Own}).
func MayTuneRuleFrequency(ctx context.Context) bool {
	return granted(ctx, rules.Edit, rules.SessionRule, rules.Own)
}
