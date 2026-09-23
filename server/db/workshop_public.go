// package: db / database access and repository layer
// type:    data
// job:     the public workshop page: its slug, its list of public games, and the share links visitors play through.
// limits:  does not store the other workshop settings (-> workshop.go) or manage hand-made share links (-> game_shares.go).
package db

import (
	db "cgl/db/sqlc"
	"cgl/functional/wordtoken"
	"cgl/log"
	"cgl/obj"
	"context"
	"database/sql"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

const (
	// PublicPageSessions is how many sessions each game's share link on the public page allows.
	PublicPageSessions = 50

	PublicSlugMinLength        = 3
	PublicSlugMaxLength        = 60
	PublicDescriptionMaxLength = 2000
	publicSlugNameMaxLength    = 30
	publicSlugWords            = 2
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
	return queries().WorkshopPublicSlugExists(ctx, sql.NullString{String: slug, Valid: true})
}

// newPublicSlug returns a free slug of the form <name>-<word>-<word>.
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
func NormalizePublicSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

// validatePublicSlug checks a normalised slug against the format rules.
func validatePublicSlug(slug string) error {
	if len(slug) < PublicSlugMinLength || len(slug) > PublicSlugMaxLength || !publicSlugPattern.MatchString(slug) {
		return obj.ErrPublicSlugInvalid("The link may contain only a-z, 0-9 and single hyphens, and must be 3 to 60 characters long")
	}
	return nil
}

// BackfillWorkshopPublicSlugs gives every workshop created before migration 033 its slug.
func BackfillWorkshopPublicSlugs(ctx context.Context) error {
	rows, err := queries().ListWorkshopsWithoutPublicSlug(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		slug, err := newPublicSlug(ctx, row.Name)
		if err != nil {
			return err
		}
		if err := queries().SetWorkshopPublicSlug(ctx, db.SetWorkshopPublicSlugParams{
			ID:         row.ID,
			PublicSlug: sql.NullString{String: slug, Valid: true},
		}); err != nil {
			return err
		}
	}
	if len(rows) > 0 {
		log.Info("gave workshops their public page slug", "count", len(rows))
	}
	return nil
}

// loadPublicWorkshop returns the workshop behind a slug while its page is switched on.
func loadPublicWorkshop(ctx context.Context, slug string) (*db.Workshop, error) {
	ws, err := queries().GetWorkshopByPublicSlug(ctx, sql.NullString{String: NormalizePublicSlug(slug), Valid: true})
	if err != nil || !ws.Public {
		return nil, obj.ErrNotFound("workshop page not found")
	}
	return &ws, nil
}

// publicPageShareMu serialises the creation of public-page share links, so two
// visitors opening the page at once don't create two links for one game. The
// unique index game_share_public_page_uniq backs it up.
var publicPageShareMu sync.Mutex

// GetPublicWorkshopPage returns the public page behind a slug. It creates the
// share link of every listed game that has none yet, paid by the workshop's key.
func GetPublicWorkshopPage(ctx context.Context, slug string) (*obj.PublicWorkshopPage, error) {
	ws, err := loadPublicWorkshop(ctx, slug)
	if err != nil {
		return nil, err
	}
	workshopID := uuid.NullUUID{UUID: ws.ID, Valid: true}

	games, err := queries().GetPublicGamesByWorkshop(ctx, workshopID)
	if err != nil {
		return nil, obj.ErrServerError("failed to load workshop games")
	}

	shares, err := ensurePublicPageShares(ctx, ws, games)
	if err != nil {
		return nil, err
	}

	page := &obj.PublicWorkshopPage{
		Name:        ws.Name,
		Description: ws.PublicDescription.String,
		Games:       make([]obj.PublicWorkshopGame, 0, len(games)),
	}
	for _, g := range games {
		game := obj.PublicWorkshopGame{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
		}
		if share, ok := shares[g.ID]; ok && isPlayable(g) {
			game.Play = &obj.PublicWorkshopPlay{
				Token:     share.Token,
				Remaining: int(share.Remaining.Int32),
				Limit:     PublicPageSessions,
			}
		}
		page.Games = append(page.Games, game)
	}
	return page, nil
}

// ensurePublicPageShares returns the public-page share link of each game, keyed
// by game ID, and creates the missing ones when the workshop has a key.
func ensurePublicPageShares(ctx context.Context, ws *db.Workshop, games []db.Game) (map[uuid.UUID]db.GameShare, error) {
	workshopID := uuid.NullUUID{UUID: ws.ID, Valid: true}
	load := func() (map[uuid.UUID]db.GameShare, error) {
		rows, err := queries().GetPublicPageGameSharesByWorkshop(ctx, workshopID)
		if err != nil {
			return nil, obj.ErrServerError("failed to load share links")
		}
		byGame := make(map[uuid.UUID]db.GameShare, len(rows))
		for _, row := range rows {
			byGame[row.GameID] = row
		}
		return byGame, nil
	}

	shares, err := load()
	if err != nil || !ws.DefaultApiKeyShareID.Valid {
		return shares, err
	}
	missing := func() bool {
		for _, g := range games {
			if _, ok := shares[g.ID]; !ok && isPlayable(g) {
				return true
			}
		}
		return false
	}
	if !missing() {
		return shares, nil
	}

	publicPageShareMu.Lock()
	defer publicPageShareMu.Unlock()
	if shares, err = load(); err != nil {
		return nil, err
	}

	author, err := publicPageShareAuthor(ctx, ws)
	if err != nil {
		log.Warn("public page: no author for share links", "workshop_id", ws.ID, "error", err)
		return shares, nil
	}
	sessions := PublicPageSessions
	for _, g := range games {
		if _, ok := shares[g.ID]; ok || !isPlayable(g) {
			continue
		}
		gs, err := createGameShare(ctx, author, g.ID, ws.DefaultApiKeyShareID.UUID, &ws.InstitutionID, &ws.ID, &sessions, nil, true)
		if err != nil {
			// A broken workshop key stops every game, so there is no point trying the rest.
			log.Warn("public page: failed to create share link", "workshop_id", ws.ID, "game_id", g.ID, "error", err)
			return shares, nil
		}
		row, err := queries().GetGameShareByID(ctx, gs.ID)
		if err != nil {
			return nil, obj.ErrServerError("failed to load share link")
		}
		shares[g.ID] = row
	}
	return shares, nil
}

// isPlayable mirrors the check in game.ValidatePrivateShareToken.
func isPlayable(g db.Game) bool {
	return g.SystemMessageScenario != "" && g.SystemMessageGameStart != ""
}

// publicPageShareAuthor is the account recorded as creator of the page's share
// links. Guests fall back to the author's youth-protection constraint once the
// workshop and the organisation set none (JUGENDSCHUTZ.md, "Gäste"), so this is
// the workshop's creator, or else whoever assigned the workshop its key.
func publicPageShareAuthor(ctx context.Context, ws *db.Workshop) (uuid.UUID, error) {
	if ws.CreatedBy.Valid {
		if _, err := queries().GetUserByID(ctx, ws.CreatedBy.UUID); err == nil {
			return ws.CreatedBy.UUID, nil
		}
	}
	keyShare, err := queries().GetApiKeyShareByID(ctx, ws.DefaultApiKeyShareID.UUID)
	if err != nil {
		return uuid.Nil, err
	}
	if !keyShare.CreatedBy.Valid {
		return uuid.Nil, obj.ErrNotFound("workshop key has no creator")
	}
	return keyShare.CreatedBy.UUID, nil
}

// GetPublicWorkshopGame returns a game listed on the public page behind a slug.
func GetPublicWorkshopGame(ctx context.Context, slug string, gameID uuid.UUID) (*obj.Game, error) {
	ws, err := loadPublicWorkshop(ctx, slug)
	if err != nil {
		return nil, err
	}
	game, err := loadGameByID(ctx, gameID)
	if err != nil || !game.Public || game.WorkshopID == nil || *game.WorkshopID != ws.ID {
		return nil, obj.ErrNotFound("game not found")
	}
	return game, nil
}

func deletePublicPageShares(ctx context.Context, shares []db.GameShare) {
	for _, gs := range shares {
		if err := DeleteGameShare(ctx, gs.ID); err != nil {
			log.Warn("failed to delete public page share link", "game_share_id", gs.ID, "error", err)
		}
	}
}

// deleteWorkshopPublicPageShares revokes every share link of a workshop's public page.
func deleteWorkshopPublicPageShares(ctx context.Context, workshopID uuid.UUID) {
	shares, err := queries().GetPublicPageGameSharesByWorkshop(ctx, uuid.NullUUID{UUID: workshopID, Valid: true})
	if err != nil {
		log.Warn("failed to load public page share links", "workshop_id", workshopID, "error", err)
		return
	}
	deletePublicPageShares(ctx, shares)
}

// deleteGamePublicPageShares revokes the public-page share links of one game.
func deleteGamePublicPageShares(ctx context.Context, gameID uuid.UUID) {
	shares, err := queries().GetPublicPageGameSharesByGame(ctx, gameID)
	if err != nil {
		log.Warn("failed to load public page share links", "game_id", gameID, "error", err)
		return
	}
	deletePublicPageShares(ctx, shares)
}
