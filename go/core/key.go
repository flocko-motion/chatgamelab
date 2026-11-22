package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/events"
	"cgl/log"
	"cgl/obj"
	"engine"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// A key is money: whoever bought it owns it, and only they share it, take a
// share back or delete it, the admin included (#451). A share is permission
// to use a key:
//
//   - the owner's own use of it (the self-share), made with the key, gone
//     only with it;
//   - given to the owner's organisation: its workshops play with it, its
//     head and staff sponsor games and make share links with it; a head may
//     remove it from the organisation;
//   - drawn for a sponsored game or a share link: money leaving the
//     organisation, recorded with the share it draws from. Taking that share
//     back stops what draws from it.
//
// v1's endpoints name keys by one of their shares; so does the core.

// KeyShareCard is a share as the caller sees it: the key behind it (never
// its secret, only the last four characters), what it is given to, and the
// caller's rights on it.
type KeyShareCard struct {
	ID        uuid.UUID
	CreatedAt time.Time
	CreatedBy *uuid.UUID
	Source    *uuid.UUID // the share it draws from; nil: the key itself

	Key KeySummary

	Self             bool       // the owner's own use of their key
	OrganisationID   *uuid.UUID // given to an organisation
	OrganisationName *string
	GameID           *uuid.UUID // drawn for a sponsored game or a share link
	GameName         *string
	LinkID           *uuid.UUID // the share link it pays for
	LinkRemaining    *int32     // plays left through it; nil: unlimited
	LinkWorkshopID   *uuid.UUID

	mayShare  bool
	mayRevoke bool
}

// KeySummary is a key as its shares show it.
type KeySummary struct {
	ID               uuid.UUID
	Name             string
	Platform         string
	Tail             string // the secret's last four characters
	Default          bool   // the owner's default key
	LastUsageSuccess *bool
	LastErrorCode    *string
	OwnerID          uuid.UUID
	OwnerName        string
	CreatedAt        time.Time
	ModifiedAt       time.Time
}

// Shortened is the key as it is shown: its last four characters.
func (k KeySummary) Shortened() string { return ".." + k.Tail }

// MayShare says whether the caller may draw sponsorships and share links
// from it, or, the owner, share the key: {Share, ApiKey, Own} or
// {Share, ApiKeyShare, SharedWithOwnOrganisation}.
func (s *KeyShareCard) MayShare() bool { return s.mayShare }

// MayRevoke says whether the caller may take it back, stopping what draws
// from it: the owner, a head for their organisation's, head and staff for
// what draws from it.
func (s *KeyShareCard) MayRevoke() bool { return s.mayRevoke }

// KeyShareFilter narrows a list of shares.
type KeyShareFilter struct {
	Key          *uuid.UUID // the shares of one key
	Organisation *uuid.UUID // those given to one organisation
}

// ListKeyShares lists the shares the caller may list, {List, ApiKeyShare},
// narrowed by f: those of their keys, those given to their organisation, and
// what draws from those.
func ListKeyShares(ctx context.Context, f KeyShareFilter) ([]*KeyShareCard, error) {
	p := keyShareGates(ctx)
	p.KeyID = nullUUID(f.Key)
	p.Organisation = nullUUID(f.Organisation)
	return loadKeyShares(ctx, p)
}

// ReadKeyShare opens one share, if the caller may see it; otherwise it does
// not exist for them (ErrNotFound).
func ReadKeyShare(ctx context.Context, id uuid.UUID) (*KeyShareCard, error) {
	p := keyShareGates(ctx)
	p.ID = uuid.NullUUID{UUID: id, Valid: true}
	shares, err := loadKeyShares(ctx, p)
	if err != nil {
		return nil, err
	}
	if len(shares) == 0 {
		return nil, ErrNotFound
	}
	return shares[0], nil
}

func loadKeyShares(ctx context.Context, p db.V2KeySharesParams) ([]*KeyShareCard, error) {
	rows, err := store.Queries().V2KeyShares(ctx, p)
	if err != nil {
		return nil, err
	}
	shares := make([]*KeyShareCard, len(rows))
	for i, r := range rows {
		s := &KeyShareCard{
			ID: r.ID, CreatedAt: r.CreatedAt, CreatedBy: ptrUUID(r.CreatedBy), Source: ptrUUID(r.SourceShareID),
			Key: KeySummary{
				ID: r.ApiKeyID, Name: r.KeyName, Platform: r.Platform, Tail: r.KeyTail, Default: r.IsDefault,
				LastUsageSuccess: nullBool(r.LastUsageSuccess), LastErrorCode: nullString(r.LastErrorCode),
				OwnerID: r.OwnerID, OwnerName: r.OwnerName,
				CreatedAt: r.KeyCreatedAt, ModifiedAt: r.KeyModifiedAt,
			},
			Self:             r.IsSelf,
			OrganisationID:   ptrUUID(r.InstitutionID),
			OrganisationName: nullString(r.OrganisationName),
			GameID:           ptrUUID(r.GameID),
			GameName:         nullString(r.GameName),
			LinkID:           ptrUUID(r.LinkID),
			LinkWorkshopID:   ptrUUID(r.LinkWorkshopID),
			mayShare:         r.MayShare,
			mayRevoke:        r.MayRevoke,
		}
		if r.LinkRemaining.Valid {
			s.LinkRemaining = &r.LinkRemaining.Int32
		}
		shares[i] = s
	}
	return shares, nil
}

// MyKeys are the caller's keys, each by its self-share, and every share of
// them: what /api/apikeys lists.
func MyKeys(ctx context.Context) ([]*KeyShareCard, error) {
	shares, err := ListKeyShares(ctx, KeyShareFilter{})
	if err != nil {
		return nil, err
	}
	me := account(ctx)
	mine := make([]*KeyShareCard, 0, len(shares))
	for _, s := range shares {
		if me.Valid && s.Key.OwnerID == me.UUID {
			mine = append(mine, s)
		}
	}
	return mine, nil
}

// KeyDraft is a new key: its provider, the secret, and a name; without one
// it is named after its provider and today's date.
type KeyDraft struct {
	Name     string
	Platform string
	Secret   string
}

// CreateKey stores a key of the caller's and returns their own use of it.
// It takes {Manage, ApiKey, Own}. Their first key becomes their default.
func CreateKey(ctx context.Context, d KeyDraft) (*KeyShareCard, error) {
	if !mayCreateKey(ctx) {
		return nil, forbidden("you may not add an API key")
	}
	d.Platform, d.Secret, d.Name = strings.TrimSpace(d.Platform), strings.TrimSpace(d.Secret), strings.TrimSpace(d.Name)
	switch {
	case d.Platform == "" || d.Secret == "":
		return nil, refuse("platform and key are required")
	case !acceptsKeys(d.Platform):
		return nil, badRequest(obj.ErrCodeInvalidPlatform, "platform does not take API keys: %s", d.Platform)
	}
	if d.Name == "" {
		d.Name = fmt.Sprintf("%s %s", platformName(d.Platform), time.Now().Format("02.01.06"))
	}
	owner := account(ctx).UUID
	id := uuid.New()
	var self uuid.UUID
	err := store.InTx(ctx, func(q *db.Queries) error {
		count, err := q.V2CountOwnKeys(ctx, owner)
		if err != nil {
			return err
		}
		if err := q.V2CreateKey(ctx, db.V2CreateKeyParams{
			ID: id, Owner: owner, Name: d.Name, Platform: d.Platform, Key: d.Secret, IsDefault: count == 0,
		}); err != nil {
			return err
		}
		self, err = q.V2CreateKeyShare(ctx, db.V2CreateKeyShareParams{
			CreatedBy: owner, ApiKeyID: id, UserID: uuid.NullUUID{UUID: owner, Valid: true},
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return ReadKeyShare(ctx, self)
}

// ownKey loads the key behind share id for its owner: {Manage, ApiKey, Own}.
func ownKey(ctx context.Context, id uuid.UUID) (*KeyShareCard, error) {
	share, err := ReadKeyShare(ctx, id)
	if err != nil {
		return nil, err
	}
	if !mayManageKey(ctx, share.Key.OwnerID) {
		return nil, forbidden("only the key's owner does this")
	}
	return share, nil
}

// RenameKey renames the key behind share id, for its owner.
func RenameKey(ctx context.Context, id uuid.UUID, name string) (*KeyShareCard, error) {
	share, err := ownKey(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := requireField("name", &name, maxOrganisationField); err != nil {
		return nil, err
	}
	if err := store.Queries().V2RenameKey(ctx, db.V2RenameKeyParams{ID: share.Key.ID, Name: name, ModifiedBy: account(ctx).UUID}); err != nil {
		return nil, err
	}
	return ReadKeyShare(ctx, id)
}

// SetDefaultKey makes the key behind share id its owner's default: the one
// they play with themselves.
func SetDefaultKey(ctx context.Context, id uuid.UUID) error {
	share, err := ownKey(ctx, id)
	if err != nil {
		return err
	}
	return store.InTx(ctx, func(q *db.Queries) error {
		if err := q.V2ClearDefaultKey(ctx, share.Key.OwnerID); err != nil {
			return err
		}
		return q.V2SetDefaultKey(ctx, share.Key.ID)
	})
}

// ShareKeyWithOrganisation gives the key behind share id to organisation
// org: its workshops may play with it, its head and staff sponsor games and
// make share links with it. It takes {Share, ApiKey, Own}, for the owner's
// own organisation only: a key never crosses organisations.
func ShareKeyWithOrganisation(ctx context.Context, id, org uuid.UUID) (*KeyShareCard, error) {
	share, err := ReadKeyShare(ctx, id)
	if err != nil {
		return nil, err
	}
	if !share.Self || !share.MayShare() {
		return nil, forbidden("only the key's owner shares it")
	}
	if own := from(ctx).Organisation(); own == nil || *own != org {
		return nil, forbidden("a key is shared with its owner's own organisation only")
	}
	existing, err := ListKeyShares(ctx, KeyShareFilter{Key: &share.Key.ID, Organisation: &org})
	if err != nil {
		return nil, err
	}
	for _, s := range existing {
		if s.GameID == nil {
			return s, nil // given already
		}
	}
	created, err := store.Queries().V2CreateKeyShare(ctx, db.V2CreateKeyShareParams{
		CreatedBy: account(ctx).UUID, ApiKeyID: share.Key.ID, InstitutionID: uuid.NullUUID{UUID: org, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	log.Info("key shared with organisation", "api_key_id", share.Key.ID, "institution_id", org)
	return ReadKeyShare(ctx, created)
}

// RevokeKeyShare takes a share back, and with it everything that draws from
// it: sponsorships and share links, their guests included. Workshops and
// the organisation's free use pointing at it play no more until another key
// is picked. The owner's own use of their key goes only with the key
// (DeleteKey).
func RevokeKeyShare(ctx context.Context, id uuid.UUID) error {
	share, err := ReadKeyShare(ctx, id)
	if err != nil {
		return err
	}
	if share.Self {
		return refuse("your own use of your key goes with the key: delete the key")
	}
	if !share.MayRevoke() {
		return forbidden("you may not take this share back")
	}
	rows, err := store.Queries().V2SharesDrawnFrom(ctx, id)
	if err != nil {
		return err
	}
	if err := endShares(ctx, sharesAndLinks(rows)); err != nil {
		return err
	}
	log.Info("key share taken back", "api_key_share_id", id, "by", account(ctx).UUID)
	return nil
}

// DeleteKey deletes the key behind share id, for its owner: every share of
// it ends, sponsorships and share links included, and the site's free use of
// it. Their oldest other key becomes their default.
func DeleteKey(ctx context.Context, id uuid.UUID) error {
	share, err := ownKey(ctx, id)
	if err != nil {
		return err
	}
	q := store.Queries()
	rows, err := q.V2SharesOfKey(ctx, share.Key.ID)
	if err != nil {
		return err
	}
	shares := make([]sharedLink, len(rows))
	for i, r := range rows {
		shares[i] = sharedLink{share: r.ID, link: ptrUUID(r.LinkID)}
	}
	if err := endShares(ctx, shares); err != nil {
		return err
	}
	if err := q.V2DeleteKey(ctx, share.Key.ID); err != nil {
		return err
	}
	if err := q.V2PromoteOldestKey(ctx, share.Key.OwnerID); err != nil {
		return err
	}
	log.Info("key deleted", "api_key_id", share.Key.ID, "by", account(ctx).UUID)
	return nil
}

// sharedLink is a share and the share link it pays for, if any.
type sharedLink struct {
	share uuid.UUID
	link  *uuid.UUID
}

func sharesAndLinks(rows []db.V2SharesDrawnFromRow) []sharedLink {
	shares := make([]sharedLink, len(rows))
	for i, r := range rows {
		shares[i] = sharedLink{share: r.ID, link: ptrUUID(r.LinkID)}
	}
	return shares
}

// endShares ends shares: the share links they pay for with their guests
// (v1's guest cleanup), then the shares, those drawn from others first.
func endShares(ctx context.Context, shares []sharedLink) error {
	q := store.Queries()
	ids := make([]uuid.UUID, 0, len(shares))
	for _, s := range shares {
		if s.link != nil {
			if err := q.V2DeleteLinkGuests(ctx, uuid.NullUUID{UUID: *s.link, Valid: true}); err != nil {
				return err
			}
			if err := q.V2DeleteGameShare(ctx, *s.link); err != nil {
				return err
			}
		}
		ids = append(ids, s.share)
	}
	// drawn shares point at their source: delete them before it
	for _, drawnFirst := range []bool{true, false} {
		batch := []uuid.UUID{}
		for _, id := range ids {
			drawn, err := isDrawn(ctx, id)
			if err != nil {
				return err
			}
			if drawn == drawnFirst {
				batch = append(batch, id)
			}
		}
		if len(batch) > 0 {
			if err := q.V2DeleteKeyShares(ctx, batch); err != nil {
				return err
			}
		}
	}
	return nil
}

// isDrawn says whether share id draws from another share; a gone share is
// not.
func isDrawn(ctx context.Context, id uuid.UUID) (bool, error) {
	s, err := store.Queries().V2KeyShareFacts(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return s.SourceShareID.Valid, nil
}

// SetWorkshopKey makes share the key workshop id plays with; nil: none, it
// does not play. It takes {Edit, WorkshopSettings}; the share must be one
// of the workshop's organisation's. Changing it ends the public page's
// share links, which pay with the old one.
func SetWorkshopKey(ctx context.Context, id uuid.UUID, share *uuid.UUID) (*WorkshopCard, error) {
	w, err := ReadWorkshop(ctx, id)
	if err != nil {
		return nil, err
	}
	if !w.MayEditSettings() {
		return nil, forbidden("you may not change this workshop's key")
	}
	if share != nil {
		if err := checkOrganisationShare(ctx, *share, w.row.organisationID); err != nil {
			return nil, err
		}
	}
	changed := (share == nil) != (w.row.keyShare == nil) || (share != nil && *share != *w.row.keyShare)
	if !changed {
		return w, nil
	}
	if err := store.Queries().V2SetWorkshopKey(ctx, db.V2SetWorkshopKeyParams{
		ID: id, ShareID: nullUUID(share), ModifiedBy: account(ctx).UUID,
	}); err != nil {
		return nil, err
	}
	// its page's links were paid by the old key
	if err := endPublicPageLinks(ctx, &id, nil); err != nil {
		log.Warn("failed to end a workshop's public page links", "workshop_id", id, "error", err)
	}
	events.GetBroker().PublishWorkshopUpdated(id)
	return ReadWorkshop(ctx, id)
}

// SetOrganisationFreeUseKey makes share the key organisation id's members
// play with for free; nil: none. It takes {Edit, OrganisationSettings}; the
// share must be one of the organisation's.
func SetOrganisationFreeUseKey(ctx context.Context, id uuid.UUID, share *uuid.UUID) (*OrganisationCard, error) {
	org, err := ReadOrganisation(ctx, id)
	if err != nil {
		return nil, err
	}
	if !org.MayEditSettings() {
		return nil, forbidden("you may not change this organisation's free-use key")
	}
	if share != nil {
		if err := checkOrganisationShare(ctx, *share, id); err != nil {
			return nil, err
		}
	}
	if err := store.Queries().V2SetOrganisationFreeUseKey(ctx, db.V2SetOrganisationFreeUseKeyParams{
		ID: id, ShareID: nullUUID(share), ModifiedBy: account(ctx).UUID,
	}); err != nil {
		return nil, err
	}
	return ReadOrganisation(ctx, id)
}

// checkOrganisationShare refuses a share that is not one given to
// organisation org.
func checkOrganisationShare(ctx context.Context, share, org uuid.UUID) error {
	s, err := store.Queries().V2KeyShareFacts(ctx, share)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err != nil || !s.InstitutionID.Valid || s.InstitutionID.UUID != org || s.GameID.Valid {
		return refuse("pick one of the organisation's keys")
	}
	return nil
}

// acceptsKeys says whether a key for the platform may be stored and used:
// one of the engine's that takes users' keys, or the mock in development
// mode, so a dev stack and the tests play without a provider.
func acceptsKeys(platform string) bool {
	if schemafapi.DevMode() && engine.Platform(platform) == engine.PlatformMock {
		return true
	}
	return engine.AcceptsKeys(engine.Platform(platform))
}

// platformName is the provider's name as a new key is named after it.
func platformName(platform string) string {
	if platform == "" {
		return platform
	}
	return strings.ToUpper(platform[:1]) + platform[1:]
}

func nullBool(b sql.NullBool) *bool {
	if !b.Valid {
		return nil
	}
	return &b.Bool
}
