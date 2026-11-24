// package: core / embedding
// type:    logic
// job:     a game's embed link, who may frame it, and starting it on a verified page's yes
// limits:  verified pages themselves (-> verified_page.go); serving the page (-> api/engine.go)
package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"syscall"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/functional"
	"cgl/log"
	"cgl/tokenlock"

	"engine"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// A game's embed link (#340): one per game, its token random and kept for
// good, as the game id is public and the link spends a sponsor's key. Off
// keeps the token, so on revives the same snippet. Whoever may share the
// game controls it. A game embeds only on a verified page of its
// organisation, and only the host's yes starts it there (EmbedPlay); the
// token alone starts nothing.

// EmbedMode is who may show an embedded game.
type EmbedMode string

// EmbedVerified: one verified page, on its host's yes. It is the only mode:
// for now only organisations embed, on their own verified page.
const EmbedVerified EmbedMode = "verified"

// GameEmbed is a game's embed link as the designer sees it.
type GameEmbed struct {
	ID              uuid.UUID
	Token           string
	Enabled         bool
	Mode            EmbedMode
	VerifiedPageID  *uuid.UUID
	VerifiedPageURL *string
	PageApproved    bool       // an admin approved the verified page; until then nothing starts
	ShareID         uuid.UUID  // the share drawn for it, which pays
	SourceShareID   *uuid.UUID // the share it is drawn from
	KeyID           uuid.UUID
	KeyName         string
	Platform        string

	payingOrganisation uuid.NullUUID
}

// ReadGameEmbed is game's embed link, nil when it has none, for whoever may
// share the game.
func ReadGameEmbed(ctx context.Context, game uuid.UUID) (*GameEmbed, error) {
	card, err := ReadGameCard(ctx, game)
	if err != nil {
		return nil, err
	}
	if !card.MayShare() {
		return nil, forbidden("you may not share this game")
	}
	return loadGameEmbed(ctx, game)
}

func loadGameEmbed(ctx context.Context, game uuid.UUID) (*GameEmbed, error) {
	rows, err := store.Queries().V2GameEmbed(ctx, game)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	r := rows[0]
	return &GameEmbed{
		ID: r.ID, Token: r.Token, Enabled: r.Enabled, Mode: EmbedMode(r.EmbedMode.String),
		VerifiedPageID: ptrUUID(r.VerifiedPageID), VerifiedPageURL: nullString(r.VerifiedPageUrl),
		PageApproved: r.VerifiedPageApproved,
		ShareID:      r.ApiKeyShareID, SourceShareID: ptrUUID(r.SourceShareID),
		KeyID: r.KeyID, KeyName: r.KeyName, Platform: r.Platform,
		payingOrganisation: r.InstitutionID,
	}, nil
}

// EmbedSettings set a game's embed link. From is the share to pay with;
// nil keeps the one paying now.
type EmbedSettings struct {
	Enabled      bool
	Mode         EmbedMode
	From         *uuid.UUID
	VerifiedPage *uuid.UUID
}

// SetGameEmbed sets game's embed link, making it the first time, for
// whoever may share the game. A new paying share replaces the old one; the
// link, its token and its guests stay.
func SetGameEmbed(ctx context.Context, game uuid.UUID, s EmbedSettings) (*GameEmbed, error) {
	card, err := ReadGameCard(ctx, game)
	if err != nil {
		return nil, err
	}
	if !card.MayShare() {
		return nil, forbidden("you may not share this game")
	}
	page, err := embedPage(ctx, card, s)
	if err != nil {
		return nil, err
	}
	existing, err := loadGameEmbed(ctx, game)
	if err != nil {
		return nil, err
	}
	if existing == nil && s.From == nil {
		return nil, refuse("sponsorKeyShareId is required to embed a game")
	}
	source, err := embedSource(ctx, card, existing, s.From)
	if err != nil {
		return nil, err
	}

	var id uuid.UUID
	var oldShare *uuid.UUID
	err = store.InTx(ctx, func(q *db.Queries) error {
		var share uuid.UUID
		var org uuid.NullUUID
		if existing != nil {
			share, org = existing.ShareID, existing.payingOrganisation
		}
		if source != nil {
			drawn, err := q.V2CreateKeyShare(ctx, db.V2CreateKeyShareParams{
				CreatedBy: account(ctx).UUID, ApiKeyID: source.Key.ID,
				GameID: uuid.NullUUID{UUID: game, Valid: true}, SourceShareID: uuid.NullUUID{UUID: source.ID, Valid: true},
			})
			if err != nil {
				return err
			}
			if existing != nil {
				oldShare = &existing.ShareID
			}
			share, org = drawn, nullUUID(source.OrganisationID)
		}
		if existing != nil {
			id = existing.ID
			return q.V2UpdateEmbedLink(ctx, db.V2UpdateEmbedLinkParams{
				ID: id, Enabled: s.Enabled, EmbedMode: string(s.Mode), VerifiedPageID: page, ApiKeyShareID: share, InstitutionID: org,
			})
		}
		token, err := functional.GenerateSecureToken(20)
		if err != nil {
			return err
		}
		id, err = q.V2CreateEmbedLink(ctx, db.V2CreateEmbedLinkParams{
			GameID: game, Token: token, ApiKeyShareID: share, InstitutionID: org, CreatedBy: account(ctx).UUID,
			Enabled: s.Enabled, EmbedMode: string(s.Mode), VerifiedPageID: page,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	if oldShare != nil {
		if err := endShares(ctx, []sharedLink{{share: *oldShare}}); err != nil {
			return nil, err
		}
	}
	log.Info("game embedding set", "game_id", game, "share_link_id", id, "enabled", s.Enabled, "mode", s.Mode, "by", account(ctx).UUID)
	return loadGameEmbed(ctx, game)
}

// embedPage checks the mode and answers its verified page: one of the
// game's organisation's.
func embedPage(ctx context.Context, card *GameCard, s EmbedSettings) (uuid.NullUUID, error) {
	if s.Mode != EmbedVerified {
		return uuid.NullUUID{}, refuse("unknown mode %q; known: verified", s.Mode)
	}
	if s.VerifiedPage == nil {
		return uuid.NullUUID{}, refuse("verifiedPageId is required for the verified mode")
	}
	org := gameOrganisation(ctx, card)
	pageOrg, err := store.Queries().V2VerifiedPageOrganisation(ctx, *s.VerifiedPage)
	if err != nil || org == nil || pageOrg != *org {
		return uuid.NullUUID{}, refuse("the verified page must be one of the game's organisation's")
	}
	return uuid.NullUUID{UUID: *s.VerifiedPage, Valid: true}, nil
}

// embedSource is the share a new paying share is drawn from, nil when what
// pays stays.
func embedSource(ctx context.Context, card *GameCard, existing *GameEmbed, from *uuid.UUID) (*KeyShareCard, error) {
	if from == nil || (existing != nil && existing.SourceShareID != nil && *existing.SourceShareID == *from) {
		return nil, nil
	}
	source, _, _, err := linkSource(ctx, card, ShareLinkDraft{Game: card.ID(), From: from})
	if err != nil {
		return nil, err
	}
	if source.Key.LastUsageSuccess != nil && !*source.Key.LastUsageSuccess {
		return nil, refuse("the key must work before it pays for an embedded game")
	}
	return source, nil
}

// gameOrganisation is the organisation a game belongs to: its workshop's.
func gameOrganisation(ctx context.Context, card *GameCard) *uuid.UUID {
	if card.row.workshopID == nil {
		return nil
	}
	w, err := ReadWorkshop(ctx, *card.row.workshopID)
	if err != nil {
		return nil
	}
	org := w.OrganisationID()
	return &org
}

// Framing is the playback-engine page's frame-ancestors, and the origin
// besides its own that handed-over CSS may load from ("" for none).
type Framing struct {
	Ancestors   string
	AssetOrigin string
}

// Framed says whether a site other than the platform may frame the page.
func (f Framing) Framed() bool { return f.Ancestors != framedBySelf }

const framedBySelf = "'self'"

// EmbedFraming is who may frame the playback-engine opened for share token
// or session: an approved verified page's origin for its embed link,
// otherwise the platform alone.
func EmbedFraming(ctx context.Context, token string, session *uuid.UUID) Framing {
	self := Framing{Ancestors: framedBySelf}
	if token == "" && session == nil {
		return self
	}
	rows, err := store.Queries().V2EmbedFraming(ctx, db.V2EmbedFramingParams{
		Token: sql.NullString{String: token, Valid: token != ""}, Session: nullUUID(session),
	})
	if err != nil || len(rows) == 0 {
		return self
	}
	r := rows[0]
	switch {
	case !r.Embed || !r.Enabled:
		return self
	case EmbedMode(r.EmbedMode.String) == EmbedVerified && r.VerifiedOrigin.Valid && r.VerifiedPageApproved:
		return Framing{Ancestors: r.VerifiedOrigin.String, AssetOrigin: r.VerifiedOrigin.String}
	}
	return self
}

// HostIDs name a start on a host page: its visitor and this request.
type HostIDs struct {
	Player  string
	Request string
}

var (
	// hostID is a player or request ID: up to 128 of letters, digits, ._:-
	hostID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
	// shareToken is the most a share link token can be.
	shareToken = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
)

const hostTimeout = 5 * time.Second

// hostClient asks hosts. It follows no redirect, as only the registered
// callback answers, and reaches no internal address outside development.
var hostClient = &http.Client{
	Timeout:       hostTimeout,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	Transport: &http.Transport{
		DialContext: (&net.Dialer{Timeout: hostTimeout, Control: publicOnly}).DialContext,
	},
}

// publicOnly refuses a connection to a loopback, private or link-local
// address, unless in development.
func publicOnly(_, address string, _ syscall.RawConn) error {
	if schemafapi.DevMode() {
		return nil
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return errors.New("the callback must be on the public internet")
	}
	return nil
}

// EmbedPlay opens the play behind a share token, for the engine's start. An
// embed link starts on its approved page's yes only, asked with ids; its guest is
// the player ID hashed with the link. Any other link opens as by
// ShareLinkPlay.
func EmbedPlay(ctx context.Context, token string, ids HostIDs) (*LinkPlay, error) {
	row, err := openToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if row.mode == "" {
		return openLinkPlay(ctx, row, token)
	}
	refused := engine.Refuse(engine.ProblemHostRefused, "the host page did not let this game start")
	if !hostID.MatchString(ids.Player) || !hostID.MatchString(ids.Request) || row.callbackURL == "" || !row.approved {
		return nil, refused
	}
	if err := askHost(ctx, row.callbackURL, row.pageKey, ids); err != nil {
		log.Info("host refused an embedded start", "share_link_id", row.LinkID, "reason", err.Error())
		return nil, refused
	}
	play, err := openLinkPlay(ctx, row, token)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(row.LinkID.String() + "\x00" + ids.Player))
	play.OutsideID = hex.EncodeToString(sum[:])
	return play, nil
}

// openToken loads the link a token names. A token that opens nothing, or a
// link switched off, is not found and counts as a guess (tokenlock).
func openToken(ctx context.Context, token string) (*linkPlayRow, error) {
	if token == "" {
		return nil, refuse("a share link token is required")
	}
	if !shareToken.MatchString(token) {
		tokenlock.Fail()
		return nil, ErrNotFound
	}
	row, err := linkPlay(ctx, sql.NullString{String: token, Valid: true}, uuid.NullUUID{})
	if err == nil && !row.enabled {
		err = ErrNotFound
	}
	if errors.Is(err, ErrNotFound) {
		tokenlock.Fail()
	}
	return row, err
}

// askHost asks callback whether to start: yes is HTTP 200 with
// {"ok": true}; anything else, or no answer in time, is no.
func askHost(ctx context.Context, callback, pageID string, ids HostIDs) error {
	body, err := json.Marshal(map[string]string{"pageId": pageID, "playerId": ids.Player, "requestId": ids.Request})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, hostTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callback, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := hostClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("the host answered " + resp.Status)
	}
	var answer struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024)).Decode(&answer); err != nil || !answer.OK {
		return errors.New("the host said no")
	}
	return nil
}
