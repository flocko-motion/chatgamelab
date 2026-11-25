// package: core / playing
// type:    logic
// job:     what a play reads and writes: its share link, its rule's facts, the stores a session lives in
// limits:  building and running sessions (-> session/)
package core

import (
	"context"
	"database/sql"
	"errors"

	"cgl/core/internal/sessionstore"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/functional"
	"cgl/obj"

	"engine"

	ung "github.com/dillonstreator/go-unique-name-generator"
	"github.com/dillonstreator/go-unique-name-generator/dictionaries"
	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// Playing is session/'s: it builds a session from a game, picks an offer,
// resolves the youth-protection rule and launches or resumes. This file is
// what it reads and writes through the core, always for the caller in its
// context: the share link a play goes through, the facts the rule reads,
// and the stores a session is saved in.

// PlayerLanguage is the caller's profile language; empty for a guest
// without an account.
func PlayerLanguage(ctx context.Context) string {
	me := from(ctx).Account()
	if me == nil {
		return ""
	}
	user, err := ReadUser(ctx, *me)
	if err != nil {
		return ""
	}
	account, err := user.AsUserAccount()
	if err != nil {
		return ""
	}
	return account.Language()
}

// LinkPlay is a share link as a play through it needs it: whoever holds its
// token plays its game, paid by its share.
type LinkPlay struct {
	LinkID         uuid.UUID
	Game           *GameCard
	WorkshopID     *uuid.UUID
	OrganisationID *uuid.UUID
	AuthorID       *uuid.UUID
	Remaining      *int32 // plays left; nil: unlimited
	Payer          Payer  // its share's key, at the link's tier
	// OutsideID is the guest a verified page's visitor is, the host's player
	// ID hashed with the link (EmbedPlay); empty otherwise.
	OutsideID string
}

// ShareLinkPlay opens the play behind a share link's token. An embed link
// never starts by its token alone: only by its host's yes (EmbedPlay).
func ShareLinkPlay(ctx context.Context, token string) (*LinkPlay, error) {
	play, err := openToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if play.mode != "" {
		return nil, engine.Refuse(engine.ProblemHostRefused, "this game starts only on its host page")
	}
	return openLinkPlay(ctx, play, token)
}

// openLinkPlay reads the game a link plays and checks it is ready.
func openLinkPlay(ctx context.Context, play *linkPlayRow, token string) (*LinkPlay, error) {
	game, err := ReadGameCardByLink(ctx, play.gameID, token)
	if err != nil || !game.MayPlay() {
		return nil, ErrNotFound
	}
	content, err := game.AsGameContent()
	if err != nil || content.Scenario() == "" || content.GameStart() == "" {
		return nil, refuse("this game is not ready to be played")
	}
	play.Game = game
	return &play.LinkPlay, nil
}

// linkPlayRow is a LinkPlay with the game it opens, before the game is read.
type linkPlayRow struct {
	LinkPlay
	gameID      uuid.UUID
	enabled     bool      // an embed link switched off plays no more
	mode        EmbedMode // an embed link's mode; empty for any other link
	pageKey     string    // the verified page's page ID
	approved    bool      // the verified page is approved by an admin
	callbackURL string    // where the verified page's host is asked
}

// linkPlay loads a share link by its token or its id, with its payer.
func linkPlay(ctx context.Context, token sql.NullString, id uuid.NullUUID) (*linkPlayRow, error) {
	rows, err := store.Queries().V2ShareLinkPlay(ctx, db.V2ShareLinkPlayParams{Token: token, ID: id})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	r := rows[0]
	if r.OrganisationSuspended {
		return nil, forbiddenWith(obj.ErrCodeOrganisationSuspended, "the organisation of this share link is suspended")
	}
	k, err := store.Queries().V2PayerShare(ctx, r.ApiKeyShareID)
	if err != nil {
		return nil, badRequest(obj.ErrCodeNoApiKey, "the share link's key is gone")
	}
	if !acceptsKeys(k.Platform) {
		return nil, refuse("the share link's key is for a platform not supported yet")
	}
	// the tier: the workshop's for a workshop's link, else the link's own,
	// else the site's default
	tier := r.AiQualityTier
	if r.WorkshopID.Valid {
		tier = r.WorkshopTier
	}
	if !tier.Valid || tier.String == "" {
		settings, err := store.Queries().V2GetSystemSettings(ctx)
		tier = sql.NullString{String: string(engine.TierMedium), Valid: true}
		if err == nil && settings.DefaultAiQualityTier != "" {
			tier.String = settings.DefaultAiQualityTier
		}
	}
	play := &linkPlayRow{
		LinkPlay: LinkPlay{
			LinkID: r.ID, WorkshopID: ptrUUID(r.WorkshopID), OrganisationID: ptrUUID(r.InstitutionID),
			AuthorID: ptrUUID(r.CreatedBy),
			Payer: Payer{
				ShareID: &k.ID, KeyID: k.KeyID, KeyName: k.Name, Platform: k.Platform,
				Tier: tier.String, Source: obj.ApiKeyTypeSponsor, chosen: true,
			}.lastChecked(k.LastUsageSuccess, k.LastErrorCode),
		},
		gameID: r.GameID, enabled: r.Enabled, mode: EmbedMode(r.EmbedMode.String),
		pageKey: r.VerifiedPageKey.String, callbackURL: r.CallbackUrl.String, approved: r.VerifiedPageApproved,
	}
	if r.Remaining.Valid {
		play.Remaining = &r.Remaining.Int32
	}
	return play, nil
}

// RuleSource is where a youth-protection rule comes from: a workshop or an
// organisation, its name and its rule; nil rule: it sets none.
type RuleSource struct {
	Name string
	Rule *string
}

// PlayerRules are the facts the rule reads about a player.
type PlayerRules struct {
	AgeGroup     *string
	Workshop     *RuleSource // the current workshop: the one they take part in
	Organisation *RuleSource // their own (head, staff), else the current workshop's
}

// PlayRuleFacts are what a play's youth-protection rule reads; the cascade
// is session/rule.go's.
type PlayRuleFacts struct {
	LinkGuest        bool // the player came through a share link: a guest, not known
	Player           *PlayerRules
	LinkWorkshop     *RuleSource
	LinkOrganisation *RuleSource
	Author           *PlayerRules // the link's author, whose own rule protects a guest last
	Site             map[string]*string
}

// ReadPlayRuleFacts reads them for the caller's play only: the rule is part
// of playing, whoever may see the workshops behind it.
func ReadPlayRuleFacts(ctx context.Context, link *LinkPlay) (PlayRuleFacts, error) {
	q := store.Queries()
	facts := PlayRuleFacts{LinkGuest: true}
	if me := from(ctx).Account(); me != nil {
		player, guest, err := playerRules(ctx, *me)
		if err != nil {
			return facts, err
		}
		facts.LinkGuest = guest
		if !guest {
			facts.Player = player
		}
	}
	if link != nil {
		if link.WorkshopID != nil {
			w, err := q.V2RuleSource(ctx, *link.WorkshopID)
			if err == nil {
				facts.LinkWorkshop = &RuleSource{Name: workshopSourceName(w.WorkshopName, w.WorkshopOrganisationName), Rule: nullString(w.WorkshopRule)}
			}
		}
		if link.OrganisationID != nil {
			o, err := q.V2GetInstitutionByID(ctx, *link.OrganisationID)
			if err == nil {
				facts.LinkOrganisation = &RuleSource{Name: o.Name, Rule: nullString(o.PromptConstraints)}
			}
		}
		if link.AuthorID != nil {
			if author, _, err := playerRules(ctx, *link.AuthorID); err == nil {
				facts.Author = author
			}
		}
	}
	settings, err := q.V2GetSystemSettings(ctx)
	if err != nil {
		return facts, err
	}
	site := siteRulesOf(settings)
	facts.Site = map[string]*string{
		obj.AgeGroupU13:  site.U13,
		obj.AgeGroupU13p: site.U13p,
		obj.AgeGroupU18:  site.U18,
	}
	return facts, nil
}

// playerRules are a user's rule facts, and whether they came through a
// share link.
func playerRules(ctx context.Context, user uuid.UUID) (*PlayerRules, bool, error) {
	r, err := store.Queries().V2RuleFacts(ctx, user)
	if err != nil {
		return nil, false, err
	}
	rules := &PlayerRules{AgeGroup: nullString(r.AgeGroup)}
	if r.WorkshopName.Valid {
		rules.Workshop = &RuleSource{Name: workshopSourceName(r.WorkshopName.String, r.WorkshopOrganisationName), Rule: nullString(r.WorkshopRule)}
	}
	if r.OrganisationName.Valid {
		rules.Organisation = &RuleSource{Name: r.OrganisationName.String, Rule: nullString(r.OrganisationRule)}
	}
	return rules, r.LinkGuest, nil
}

// workshopSourceName is a workshop as a rule's source names it:
// "Workshop (Organisation)".
func workshopSourceName(name string, org sql.NullString) string {
	if org.Valid && org.String != "" {
		return name + " (" + org.String + ")"
	}
	return name
}

// NewPlay is what a new session is saved with. Who plays the core decides:
// the caller, or through Link without an account a guest made for it.
type NewPlay struct {
	Game           uuid.UUID
	Link           *LinkPlay // played through this share link
	Payer          Payer
	Platform       string
	Language       string
	ReplaceEarlier bool // starting anew replaces the player's earlier sessions of the game
}

// LaunchStore is the store a session launches with: its first save does all
// the start takes in one transaction, so a refused launch leaves nothing.
func LaunchStore(ctx context.Context, p NewPlay) (engine.Persist, uuid.UUID, error) {
	meta := sessionstore.NewSession{
		GameID: p.Game, ApiKeyShareID: p.Payer.ShareID, Platform: p.Platform, Language: p.Language,
		ReplaceEarlier: p.ReplaceEarlier,
	}
	c := from(ctx)
	if p.Link != nil {
		if err := MayStartPlayingByLink(ctx, p.Link.WorkshopID); err != nil {
			return nil, uuid.Nil, err
		}
		meta.GameShareID, meta.WorkshopID, meta.ConsumesPlay = &p.Link.LinkID, p.Link.WorkshopID, true
	} else {
		if err := MayStartPlaying(ctx, p.Game); err != nil {
			return nil, uuid.Nil, err
		}
		meta.WorkshopID = c.CurrentWorkshop()
	}
	switch me := c.Account(); {
	case me != nil:
		meta.UserID = *me
	case p.Link != nil:
		meta.UserID = uuid.New()
		meta.Guest = &sessionstore.NewGuest{Name: guestName(), ShareID: p.Link.LinkID, OutsideID: p.Link.OutsideID}
	default:
		return nil, uuid.Nil, engine.Refuse(engine.ProblemDenied, "sign in to play this game")
	}
	return sessionstore.Launching(meta), meta.UserID, nil
}

// Replaced are the earlier sessions a launch through store replaced.
func Replaced(store engine.Persist) []uuid.UUID { return sessionstore.Replaced(store) }

// ErrNoPlaysLeft: the share link ran out of plays as the session started.
var ErrNoPlaysLeft = sessionstore.ErrNoPlaysLeft

// Resuming is a stored session as its resume needs it.
type Resuming struct {
	GameID   uuid.UUID
	Platform string
	LinkPays *Payer // the link's key, while the link exists and its key is on the session's platform
	Store    engine.Persist
	// Cover is the game's cover, nil for a game without one. It is part of
	// the game's card, so whoever may play the session sees it.
	Cover []byte
}

// ResumeSession opens a stored session to resume it, for a caller the
// session guard lets play it (OpenSession).
func ResumeSession(ctx context.Context, id uuid.UUID) (*Resuming, error) {
	if err := OpenSession(ctx, id, PlaySession); err != nil {
		return nil, err
	}
	row, err := store.Queries().V2GetGameSessionByID(ctx, id)
	if err != nil {
		return nil, engine.Refuse(engine.ProblemUnknown, "there is no session "+id.String())
	}
	r := &Resuming{GameID: row.GameID, Platform: row.Platform, Store: sessionstore.SessionStore{}}
	if cover, err := store.Queries().V2GameCover(ctx, row.GameID); err == nil {
		r.Cover = cover.Image
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if row.GameShareID.Valid {
		// an embed link switched off pays no more, for a session begun before either
		if link, err := linkPlay(ctx, sql.NullString{}, row.GameShareID); err == nil && link.enabled && link.Payer.Platform == row.Platform {
			r.LinkPays = &link.Payer
		}
	}
	return r, nil
}

// guestName is like "guest-red-dragon-a1b2": the suffix keeps it unique, as
// colour and animal pairs are few.
func guestName() string {
	names := ung.NewUniqueNameGenerator(
		ung.WithDictionaries([][]string{dictionaries.Colors, dictionaries.Animals}),
		ung.WithSeparator("-"),
	)
	return "guest-" + names.Generate() + "-" + functional.First(functional.GenerateSecureToken(2))
}

// PlayRefusal turns the core's errors into the refusals the playback-engine
// tells the player.
func PlayRefusal(err error) error {
	var refusal *engine.Refusal
	var coded interface{ Code() string }
	switch {
	case err == nil:
		return nil
	case errors.As(err, &refusal):
		return err
	case errors.As(err, &coded) && coded.Code() == obj.ErrCodeNoApiKey:
		return engine.Refuse(engine.ProblemNoKey, err.Error())
	case errors.Is(err, schemafapi.ErrNotFound), errors.Is(err, schemafapi.ErrForbidden):
		return engine.Refuse(engine.ProblemDenied, err.Error())
	}
	return engine.Refuse(engine.ProblemFailed, err.Error())
}

// SessionMaker names who creates a session, so a one-time key answers its
// maker only; a guest's the transport binds to the share token.
func SessionMaker(ctx context.Context) string {
	if me := from(ctx).Account(); me != nil {
		return "user:" + me.String()
	}
	return ""
}
