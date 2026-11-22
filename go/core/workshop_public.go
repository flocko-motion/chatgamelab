package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/functional"
	"cgl/functional/wordtoken"
	"cgl/legal"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"golang.org/x/text/unicode/norm"
)

// A workshop's public page (/w/<slug>) shows its public games to anyone,
// without an account: {Read, WorkshopCard, Public} and {Read, GameCard,
// Public}. While the workshop has a key, each playable game on it gets a
// share link of the page's, paid by a share drawn from the workshop's key;
// the page makes the missing ones as a visitor opens it. A suspended
// organisation's pages are gone.

const (
	// PublicPageSessions is how many sessions each game's share link on the public page allows.
	PublicPageSessions = 50

	PublicSlugMinLength        = 3
	PublicSlugMaxLength        = 60
	PublicDescriptionMaxLength = 2000
	publicSlugNameMaxLength    = 30
	publicSlugWords            = 2

	PublicLinksMax                 = 10
	PublicLinkTitleMaxLength       = 100
	PublicLinkDescriptionMaxLength = 300
	PublicLinkURLMaxLength         = 500
)

var publicSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// publicSlugBase turns a workshop name into the leading part of its slug:
// umlauts transliterated, other accents dropped, lowercase, runs of other
// characters as one "-", cut at a word boundary to publicSlugNameMaxLength.
func publicSlugBase(name string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range norm.NFD.String(wordtoken.Normalize(name)) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingHyphen = false
			b.WriteRune(r)
		} else {
			pendingHyphen = true
		}
	}
	base := b.String()
	if len(base) > publicSlugNameMaxLength {
		base = base[:publicSlugNameMaxLength]
		if i := strings.LastIndexByte(base, '-'); i > 0 {
			base = base[:i]
		}
	}
	return base
}

func joinPublicSlug(base, words string) string {
	if base == "" {
		return words
	}
	return base + "-" + words
}

func publicSlugExists(ctx context.Context, slug string) (bool, error) {
	return store.Queries().V2PublicSlugExists(ctx, sql.NullString{String: slug, Valid: true})
}

// NewPublicSlug returns a free slug of the form <name>-<word>-<word>.
func newPublicSlug(ctx context.Context, name string) (string, error) {
	base := publicSlugBase(name)
	words, err := wordtoken.GenerateUnique(ctx, publicSlugWords, func(ctx context.Context, words string) (bool, error) {
		return publicSlugExists(ctx, joinPublicSlug(base, words))
	})
	if err != nil {
		return "", err
	}
	return joinPublicSlug(base, words), nil
}

// NormalizePublicSlug maps a slug typed by a leader onto the stored form.
func normalizePublicSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

// ValidatePublicSlug checks a normalised slug against the format rules.
func validatePublicSlug(slug string) error {
	if len(slug) < PublicSlugMinLength || len(slug) > PublicSlugMaxLength || !publicSlugPattern.MatchString(slug) {
		return obj.ErrPublicSlugInvalid("The link may contain only a-z, 0-9 and single hyphens, and must be 3 to 60 characters long")
	}
	return nil
}

// ValidatePublicLinks checks the further reading a leader entered. The scheme
// check is the point of this function: the page renders these as links, so a
// "javascript:" address would run a stranger's code in every visitor's browser.
func validatePublicLinks(links []obj.PublicWorkshopLink) ([]obj.PublicWorkshopLink, error) {
	if len(links) > PublicLinksMax {
		return nil, obj.ErrPublicLinksInvalid("A workshop may show at most 10 links")
	}

	cleaned := make([]obj.PublicWorkshopLink, 0, len(links))
	for _, link := range links {
		title := strings.TrimSpace(link.Title)
		description := strings.TrimSpace(link.Description)
		raw := strings.TrimSpace(link.URL)

		// A row the leader left blank is dropped, not rejected.
		if title == "" && description == "" && raw == "" {
			continue
		}
		if title == "" {
			return nil, obj.ErrPublicLinksInvalid("Every link needs a title")
		}
		if utf8.RuneCountInString(title) > PublicLinkTitleMaxLength {
			return nil, obj.ErrPublicLinksInvalid("A link title may be at most 100 characters long")
		}
		if utf8.RuneCountInString(description) > PublicLinkDescriptionMaxLength {
			return nil, obj.ErrPublicLinksInvalid("A link description may be at most 300 characters long")
		}
		if len(raw) > PublicLinkURLMaxLength {
			return nil, obj.ErrPublicLinksInvalid("The address is too long")
		}
		parsed, err := url.Parse(raw)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, obj.ErrPublicLinksInvalid("An address must start with http:// or https://")
		}

		cleaned = append(cleaned, obj.PublicWorkshopLink{
			Title:       title,
			Description: description,
			URL:         raw,
		})
	}

	if len(cleaned) == 0 {
		return nil, nil
	}
	return cleaned, nil
}

// MarshalPublicLinks turns validated links into the jsonb column value.
func marshalPublicLinks(links []obj.PublicWorkshopLink) (pqtype.NullRawMessage, error) {
	if len(links) == 0 {
		return pqtype.NullRawMessage{}, nil
	}
	raw, err := json.Marshal(links)
	if err != nil {
		return pqtype.NullRawMessage{}, obj.ErrServerError("failed to store the workshop links")
	}
	return pqtype.NullRawMessage{RawMessage: raw, Valid: true}, nil
}

// UnmarshalPublicLinks reads the jsonb column. Unreadable content is treated as
// no links rather than breaking every read of the workshop.
func unmarshalPublicLinks(raw pqtype.NullRawMessage) []obj.PublicWorkshopLink {
	if !raw.Valid {
		return nil
	}
	var links []obj.PublicWorkshopLink
	if err := json.Unmarshal(raw.RawMessage, &links); err != nil {
		log.Error("failed to read workshop public links", "error", err)
		return nil
	}
	return links
}

// publicPageLinkMu serialises making the page's links, so two visitors
// opening the page at once make one link per game; the unique index
// game_share_public_page_uniq backs it up.
var publicPageLinkMu sync.Mutex

// PublicWorkshopPage is the public page behind slug; an unknown slug, a page
// switched off and a suspended organisation's are not found.
func PublicWorkshopPage(ctx context.Context, slug string) (*obj.PublicWorkshopPage, error) {
	w, err := publicWorkshop(ctx, slug)
	if err != nil {
		return nil, err
	}
	games, err := store.Queries().V2PublicWorkshopGames(ctx, uuid.NullUUID{UUID: w.ID, Valid: true})
	if err != nil {
		return nil, err
	}
	links, err := publicPageLinks(ctx, w, games)
	if err != nil {
		return nil, err
	}
	page := &obj.PublicWorkshopPage{
		Name: w.Name, Description: w.PublicDescription.String, PlayAvailable: w.DefaultApiKeyShareID.Valid,
		Games: make([]obj.PublicWorkshopGame, 0, len(games)), Links: unmarshalPublicLinks(w.PublicLinks),
	}
	for _, g := range games {
		game := obj.PublicWorkshopGame{ID: g.ID, Name: g.Name, Description: g.Description}
		if l, ok := links[g.ID]; ok && g.Playable {
			game.Play = &obj.PublicWorkshopPlay{Token: l.Token, Remaining: int(l.Remaining.Int32), Limit: PublicPageSessions}
		}
		page.Games = append(page.Games, game)
	}
	return page, nil
}

// PublicWorkshopGame is a game listed on the public page behind slug, as
// anyone reads a public game; one not listed there is not found.
func PublicWorkshopGame(ctx context.Context, slug string, game uuid.UUID) (*GameCard, error) {
	w, err := publicWorkshop(ctx, slug)
	if err != nil {
		return nil, err
	}
	games, err := store.Queries().V2PublicWorkshopGames(ctx, uuid.NullUUID{UUID: w.ID, Valid: true})
	if err != nil {
		return nil, err
	}
	for _, g := range games {
		if g.ID == game {
			return ReadGameCard(ctx, game)
		}
	}
	return nil, ErrNotFound
}

// publicWorkshop is the workshop behind a public page's slug.
func publicWorkshop(ctx context.Context, slug string) (*db.V2PublicWorkshopRow, error) {
	declaration := legal.CurrentDeclaration()
	rows, err := store.Queries().V2PublicWorkshop(ctx, db.V2PublicWorkshopParams{
		Slug:               sql.NullString{String: normalizePublicSlug(slug), Valid: true},
		DeclarationVersion: declaration.Version, DeclarationHash: declaration.Hash,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}

// publicPageLinks are the page's share links by game, making the missing
// ones of playable games while the workshop has a key.
func publicPageLinks(ctx context.Context, w *db.V2PublicWorkshopRow, games []db.V2PublicWorkshopGamesRow) (map[uuid.UUID]db.V2PublicPageLinksRow, error) {
	load := func() (map[uuid.UUID]db.V2PublicPageLinksRow, error) {
		rows, err := store.Queries().V2PublicPageLinks(ctx, db.V2PublicPageLinksParams{WorkshopID: uuid.NullUUID{UUID: w.ID, Valid: true}})
		if err != nil {
			return nil, err
		}
		byGame := make(map[uuid.UUID]db.V2PublicPageLinksRow, len(rows))
		for _, r := range rows {
			byGame[r.GameID] = r
		}
		return byGame, nil
	}
	links, err := load()
	if err != nil || !w.DefaultApiKeyShareID.Valid {
		return links, err
	}
	missing := func() bool {
		for _, g := range games {
			if _, ok := links[g.ID]; !ok && g.Playable {
				return true
			}
		}
		return false
	}
	if !missing() {
		return links, nil
	}
	publicPageLinkMu.Lock()
	defer publicPageLinkMu.Unlock()
	if links, err = load(); err != nil {
		return nil, err
	}
	// the links are the workshop's: recorded as made by its creator, else
	// by whoever gave it its key; a guest without a rule of the workshop's
	// or the organisation's falls back to theirs (spec, Jugendschutz, Gäste)
	author := w.CreatedBy
	if !author.Valid {
		author = w.KeyShareCreatedBy
	}
	source, err := store.Queries().V2PayerShare(ctx, w.DefaultApiKeyShareID.UUID)
	if err != nil || !author.Valid || (source.LastUsageSuccess.Valid && !source.LastUsageSuccess.Bool) {
		log.Warn("public page: no links made", "workshop_id", w.ID, "error", err)
		return links, nil
	}
	for _, g := range games {
		if _, ok := links[g.ID]; ok || !g.Playable {
			continue
		}
		link, err := makePublicPageLink(ctx, w, source, author.UUID, g.ID)
		if err != nil {
			// a broken workshop key stops every game: no point trying the rest
			log.Warn("public page: failed to make a share link", "workshop_id", w.ID, "game_id", g.ID, "error", err)
			return links, nil
		}
		links[g.ID] = link
	}
	return links, nil
}

// makePublicPageLink makes the page's link for a game: a share drawn from
// the workshop's key, and the link it pays for.
func makePublicPageLink(ctx context.Context, w *db.V2PublicWorkshopRow, source db.V2PayerShareRow, author, game uuid.UUID) (db.V2PublicPageLinksRow, error) {
	token, err := functional.GenerateSecureToken(20)
	if err != nil {
		return db.V2PublicPageLinksRow{}, err
	}
	link := db.V2PublicPageLinksRow{GameID: game, Token: token, Remaining: sql.NullInt32{Int32: PublicPageSessions, Valid: true}}
	err = store.InTx(ctx, func(q *db.Queries) error {
		drawn, err := q.V2CreateKeyShare(ctx, db.V2CreateKeyShareParams{
			CreatedBy: author, ApiKeyID: source.KeyID,
			GameID: uuid.NullUUID{UUID: game, Valid: true}, SourceShareID: uuid.NullUUID{UUID: source.ID, Valid: true},
		})
		if err != nil {
			return err
		}
		link.ApiKeyShareID = drawn
		link.ID, err = q.V2CreatePublicPageLink(ctx, db.V2CreatePublicPageLinkParams{
			GameID: game, Token: token, ApiKeyShareID: drawn,
			InstitutionID: uuid.NullUUID{UUID: w.InstitutionID, Valid: true}, WorkshopID: uuid.NullUUID{UUID: w.ID, Valid: true},
			Remaining: link.Remaining, CreatedBy: author,
		})
		return err
	})
	return link, err
}

// endPublicPageLinks ends the public-page links of a workshop, or of a game
// on every page, as taking their shares back does: with their guests.
func endPublicPageLinks(ctx context.Context, workshop, game *uuid.UUID) error {
	rows, err := store.Queries().V2PublicPageLinks(ctx, db.V2PublicPageLinksParams{WorkshopID: nullUUID(workshop), GameID: nullUUID(game)})
	if err != nil {
		return err
	}
	shares := make([]sharedLink, len(rows))
	for i, r := range rows {
		shares[i] = sharedLink{share: r.ApiKeyShareID, link: &r.ID}
	}
	return endShares(ctx, shares)
}
