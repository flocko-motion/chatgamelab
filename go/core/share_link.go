package core

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/functional"
	"cgl/log"

	"github.com/google/uuid"
)

// A share link is a voucher: plays of one game for whoever has its token,
// paid by a share drawn for it from a share of a key (#451). Money leaves
// the organisation here, so a link may be limited by plays, and whoever pays
// sees and stops it: the key's owner, and, when an organisation's key pays,
// its head and staff. Its maker and the game's owner manage it too. Those in
// a workshop see its link, so they can hand it on, but do not change it.
//
// A sponsored game is the same without a token: the game's public plays
// draw from a share drawn for it.

// ShareLinkCard is a share link as the caller sees it: with its token, the
// game, who pays, and the caller's right to change or stop it.
type ShareLinkCard struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	CreatedBy  *uuid.UUID
	Token      string
	Remaining  *int32  // plays left; nil: unlimited
	Tier       *string // the AI quality tier it plays at; nil: its source's
	PublicPage bool    // made by a workshop's public page

	GameID       uuid.UUID
	GameName     string
	WorkshopID   *uuid.UUID
	WorkshopName *string

	ShareID            uuid.UUID  // the share drawn for it, which pays
	SourceShareID      *uuid.UUID // the share that is drawn from; nil: the key itself
	KeyID              uuid.UUID
	KeyName            string
	Platform           string
	OwnerID            uuid.UUID  // the key's owner: who pays
	PayingOrganisation *uuid.UUID // the organisation whose key pays, if one does

	mayManage bool
}

// MayManage says whether the caller may change its limits or stop it:
// {Manage, ShareLink}.
func (l *ShareLinkCard) MayManage() bool { return l.mayManage }

// ShareLinkFilter narrows a list of share links.
type ShareLinkFilter struct {
	Game *uuid.UUID // one game's
	Key  *uuid.UUID // paid by one key
}

// ListShareLinks lists the share links the caller may see, {List,
// ShareLink}, narrowed by f.
func ListShareLinks(ctx context.Context, f ShareLinkFilter) ([]*ShareLinkCard, error) {
	p := shareLinkGates(ctx)
	p.Game = nullUUID(f.Game)
	p.KeyID = nullUUID(f.Key)
	return loadShareLinks(ctx, p)
}

// ReadShareLink opens one share link, if the caller may see it; otherwise
// it does not exist for them (ErrNotFound).
func ReadShareLink(ctx context.Context, id uuid.UUID) (*ShareLinkCard, error) {
	p := shareLinkGates(ctx)
	p.ID = uuid.NullUUID{UUID: id, Valid: true}
	links, err := loadShareLinks(ctx, p)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, ErrNotFound
	}
	return links[0], nil
}

func loadShareLinks(ctx context.Context, p db.V2ShareLinksParams) ([]*ShareLinkCard, error) {
	rows, err := store.Queries().V2ShareLinks(ctx, p)
	if err != nil {
		return nil, err
	}
	links := make([]*ShareLinkCard, len(rows))
	for i, r := range rows {
		l := &ShareLinkCard{
			ID: r.ID, CreatedAt: r.CreatedAt, CreatedBy: ptrUUID(r.CreatedBy), Token: r.Token,
			Tier: nullString(r.AiQualityTier), PublicPage: r.PublicPage,
			GameID: r.GameID, GameName: r.GameName,
			WorkshopID: ptrUUID(r.WorkshopID), WorkshopName: nullString(r.WorkshopName),
			ShareID: r.ApiKeyShareID, SourceShareID: ptrUUID(r.SourceShareID),
			KeyID: r.KeyID, KeyName: r.KeyName, Platform: r.Platform, OwnerID: r.OwnerID,
			PayingOrganisation: ptrUUID(r.PayingOrganisation),
			mayManage:          r.MayManage,
		}
		if r.Remaining.Valid {
			l.Remaining = &r.Remaining.Int32
		}
		links[i] = l
	}
	return links, nil
}

// ShareLinkDraft is a new share link for a game: paid either by a share the
// caller may draw from (From), or by a workshop's key (Workshop); with an
// optional limit of plays and a tier.
type ShareLinkDraft struct {
	Game     uuid.UUID
	From     *uuid.UUID
	Workshop *uuid.UUID
	MaxPlays *int32
	Tier     *string
}

// CreateShareLink makes a share link for a game the caller may share, {Share,
// GameCard}, and returns it.
//
//   - Paid by a share the caller may draw from ({Share, ApiKey, Own} for
//     their own key, {Share, ApiKeyShare} for their organisation's).
//   - Paid by a workshop's key: the game must be the workshop's, and the
//     caller its organisation's head or staff, or in it while it allows
//     sharing. A workshop has one link per game: an existing one is
//     returned.
//
// A key known to fail is refused.
func CreateShareLink(ctx context.Context, d ShareLinkDraft) (*ShareLinkCard, error) {
	game, err := ReadGameCard(ctx, d.Game)
	if err != nil {
		return nil, err
	}
	if !game.MayShare() {
		return nil, forbidden("you may not share this game")
	}
	if d.Tier != nil && !slices.Contains(aiQualityTiers, *d.Tier) {
		return nil, refuse("unknown aiQualityTier %q; known: high, medium, low", *d.Tier)
	}
	source, workshop, org, err := linkSource(ctx, game, d)
	if err != nil {
		return nil, err
	}
	if workshop != nil {
		existing, err := store.Queries().V2WorkshopShareLinkOf(ctx, db.V2WorkshopShareLinkOfParams{
			GameID: d.Game, WorkshopID: uuid.NullUUID{UUID: *workshop, Valid: true},
		})
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			return ReadShareLink(ctx, existing[0])
		}
	}
	if source.Key.LastUsageSuccess != nil && !*source.Key.LastUsageSuccess {
		return nil, refuse("the key must work before it pays for a share link")
	}
	token, err := functional.GenerateSecureToken(20)
	if err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = store.InTx(ctx, func(q *db.Queries) error {
		drawn, err := q.V2CreateKeyShare(ctx, db.V2CreateKeyShareParams{
			CreatedBy: account(ctx).UUID, ApiKeyID: source.Key.ID,
			GameID: uuid.NullUUID{UUID: d.Game, Valid: true}, SourceShareID: uuid.NullUUID{UUID: source.ID, Valid: true},
		})
		if err != nil {
			return err
		}
		params := db.V2CreateShareLinkParams{
			GameID: d.Game, Token: token, ApiKeyShareID: drawn, CreatedBy: account(ctx).UUID,
			InstitutionID: nullUUID(org), WorkshopID: nullUUID(workshop), AiQualityTier: sqlString(d.Tier),
		}
		if d.MaxPlays != nil {
			params.Remaining = sql.NullInt32{Int32: *d.MaxPlays, Valid: true}
		}
		id, err = q.V2CreateShareLink(ctx, params)
		return err
	})
	if err != nil {
		return nil, err
	}
	log.Info("share link made", "game_id", d.Game, "share_link_id", id, "by", account(ctx).UUID)
	return ReadShareLink(ctx, id)
}

// linkSource is the share a new link draws from, for a caller who may draw
// from it, and the workshop and organisation it belongs to.
func linkSource(ctx context.Context, game *GameCard, d ShareLinkDraft) (*KeyShareCard, *uuid.UUID, *uuid.UUID, error) {
	if d.Workshop != nil {
		w, err := ReadWorkshop(ctx, *d.Workshop)
		if err != nil {
			return nil, nil, nil, err
		}
		if game.row.workshopID == nil || *game.row.workshopID != w.ID() {
			return nil, nil, nil, refuse("the game is not this workshop's")
		}
		if w.row.keyShare == nil {
			return nil, nil, nil, refuse("the workshop has no key to pay with")
		}
		org := w.OrganisationID()
		if !mayDrawFromWorkshopKey(ctx, org, w.ID()) {
			return nil, nil, nil, forbidden("you may not share this workshop's games with its key")
		}
		// drawing from the workshop's key takes no right to see it: its
		// facts are read here
		k, err := store.Queries().V2PayerShare(ctx, *w.row.keyShare)
		if err != nil {
			return nil, nil, nil, err
		}
		source := &KeyShareCard{ID: k.ID, Key: KeySummary{ID: k.KeyID, Name: k.Name, Platform: k.Platform, LastUsageSuccess: nullBool(k.LastUsageSuccess)}}
		return source, d.Workshop, &org, nil
	}
	if d.From == nil {
		return nil, nil, nil, refuse("sponsorKeyShareId or workshopId is required")
	}
	source, err := ReadKeyShare(ctx, *d.From)
	if err != nil {
		return nil, nil, nil, err
	}
	if !source.MayShare() || source.GameID != nil {
		return nil, nil, nil, forbidden("you may not pay for share links with this key")
	}
	return source, nil, source.OrganisationID, nil
}

// ShareLinkChanges are the changes to a share link: its limit of plays and
// its tier; nil leaves one as it is.
type ShareLinkChanges struct {
	MaxPlays    *int32 // plays left
	Unlimited   bool   // no limit of plays
	Tier        *string
	DefaultTier bool // its source's tier
}

// UpdateShareLink changes a share link's limit or tier, for a caller who may
// manage it: {Manage, ShareLink}.
func UpdateShareLink(ctx context.Context, id uuid.UUID, ch ShareLinkChanges) (*ShareLinkCard, error) {
	link, err := ReadShareLink(ctx, id)
	if err != nil {
		return nil, err
	}
	if !link.MayManage() {
		return nil, forbidden("you may not change this share link")
	}
	remaining, tier := link.Remaining, link.Tier
	switch {
	case ch.Unlimited:
		remaining = nil
	case ch.MaxPlays != nil:
		remaining = ch.MaxPlays
	}
	switch {
	case ch.DefaultTier:
		tier = nil
	case ch.Tier != nil:
		if !slices.Contains(aiQualityTiers, *ch.Tier) {
			return nil, refuse("unknown aiQualityTier %q; known: high, medium, low", *ch.Tier)
		}
		tier = ch.Tier
	}
	params := db.V2UpdateShareLinkParams{ID: id, AiQualityTier: sqlString(tier)}
	if remaining != nil {
		params.Remaining = sql.NullInt32{Int32: *remaining, Valid: true}
	}
	if err := store.Queries().V2UpdateShareLink(ctx, params); err != nil {
		return nil, err
	}
	return ReadShareLink(ctx, id)
}

// DeleteShareLink stops a share link, for a caller who may manage it: its
// token plays no more, its guests go, and so does the share drawn for it.
func DeleteShareLink(ctx context.Context, id uuid.UUID) error {
	link, err := ReadShareLink(ctx, id)
	if err != nil {
		return err
	}
	if !link.MayManage() {
		return forbidden("you may not stop this share link")
	}
	if err := endShares(ctx, []sharedLink{{share: link.ShareID, link: &link.ID}}); err != nil {
		return err
	}
	log.Info("share link stopped", "share_link_id", id, "by", account(ctx).UUID)
	return nil
}

// SponsorGame makes a share pay for a game's public plays, replacing what
// paid before. It takes {Edit, GameCard} on the game and the right to draw
// from the share ({Share, ApiKey, Own} or {Share, ApiKeyShare}). A key known
// to fail is refused.
func SponsorGame(ctx context.Context, game, from uuid.UUID) (*GameCard, error) {
	g, err := ReadGameCard(ctx, game)
	if err != nil {
		return nil, err
	}
	if !g.MayEditCard() {
		return nil, forbidden("you may not sponsor this game")
	}
	source, err := ReadKeyShare(ctx, from)
	if err != nil {
		return nil, err
	}
	if !source.MayShare() || source.GameID != nil {
		return nil, forbidden("you may not sponsor with this key")
	}
	if source.Key.LastUsageSuccess != nil && !*source.Key.LastUsageSuccess {
		return nil, refuse("the key must work before it sponsors a game")
	}
	if err := endSponsorship(ctx, game); err != nil {
		return nil, err
	}
	err = store.InTx(ctx, func(q *db.Queries) error {
		drawn, err := q.V2CreateKeyShare(ctx, db.V2CreateKeyShareParams{
			CreatedBy: account(ctx).UUID, ApiKeyID: source.Key.ID,
			GameID: uuid.NullUUID{UUID: game, Valid: true}, SourceShareID: uuid.NullUUID{UUID: source.ID, Valid: true},
		})
		if err != nil {
			return err
		}
		return q.V2SetGameSponsor(ctx, db.V2SetGameSponsorParams{ID: game, ShareID: uuid.NullUUID{UUID: drawn, Valid: true}})
	})
	if err != nil {
		return nil, err
	}
	log.Info("game sponsored", "game_id", game, "by", account(ctx).UUID)
	return ReadGameCard(ctx, game)
}

// EndSponsorship stops a share paying for a game's public plays: for those
// who edit the game, and for whoever may take that share back (the payer,
// and head and staff when their organisation's key pays).
func EndSponsorship(ctx context.Context, game uuid.UUID) error {
	sponsor, err := store.Queries().V2GameSponsor(ctx, game)
	if err != nil || !sponsor.Valid {
		return ErrNotFound
	}
	g, readErr := ReadGameCard(ctx, game)
	mayEdit := readErr == nil && g.MayEditCard()
	if !mayEdit {
		share, err := ReadKeyShare(ctx, sponsor.UUID)
		if err != nil {
			return err
		}
		if !share.MayRevoke() {
			return forbidden("you may not stop this sponsorship")
		}
	}
	return endSponsorship(ctx, game)
}

// endSponsorship ends what pays for a game's public plays, if anything.
func endSponsorship(ctx context.Context, game uuid.UUID) error {
	sponsor, err := store.Queries().V2GameSponsor(ctx, game)
	if err != nil {
		return err
	}
	if !sponsor.Valid {
		return nil
	}
	return endShares(ctx, []sharedLink{{share: sponsor.UUID}})
}
