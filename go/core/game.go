package core

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"time"

	"cgl/core/internal/identity"
	"cgl/core/internal/store"
	"cgl/db"

	"github.com/google/uuid"
)

// A game is split into sections, because who may see them differs: GameCard,
// GameContent and GameSettings. No field is in two sections. The card is the
// base type: whatever passes the gate is a card; the other sections stay
// locked until asked for.

// GameCardQuery is what the caller asks of a game list: wishes, not rights. It
// can only narrow what the grants let through, never widen it.
type GameCardQuery struct {
	Filter  []string // see GameCardFilters; combined, each narrows; none for all
	Search  string   // part of the name, any case
	SortBy  string   // see GameCardSorts; empty for createdAt
	SortDir string   // asc, desc; empty for desc
	Limit   int      // page size; 0 for all
	Offset  int      // matches to skip
}

// GameCardFilters are the filters a game list knows.
var GameCardFilters = []string{"own", "others", "public", "favorites", "sponsored"}

// GameCardSorts are the orders a game list knows. Only card fields: sorting by
// a locked field would reveal it through the order.
var GameCardSorts = []string{"createdAt", "modifiedAt", "name", "playCount", "visibility", "creator"}

// validate refuses what the list does not know, naming what it does: an
// unknown wish is an error, never silently ignored.
func (q GameCardQuery) validate() error {
	for _, f := range q.Filter {
		if !slices.Contains(GameCardFilters, f) {
			return refuse("unknown filter %q; known: %s", f, strings.Join(GameCardFilters, ", "))
		}
	}
	switch {
	case q.SortBy != "" && !slices.Contains(GameCardSorts, q.SortBy):
		return refuse("unknown sortBy %q; known: %s", q.SortBy, strings.Join(GameCardSorts, ", "))
	case q.SortDir != "" && q.SortDir != "asc" && q.SortDir != "desc":
		return refuse("unknown sortDir %q; known: asc, desc", q.SortDir)
	case q.Limit < 0 || q.Offset < 0:
		return refuse("limit and offset cannot be negative")
	}
	return nil
}

// wishes puts the query into the list's parameters.
func (q GameCardQuery) wishes(p *db.V2GameCardsParams) {
	p.OnlyOwn = slices.Contains(q.Filter, "own")
	p.OnlyOthers = slices.Contains(q.Filter, "others")
	p.OnlyPublic = slices.Contains(q.Filter, "public")
	p.OnlyFavorites = slices.Contains(q.Filter, "favorites")
	p.OnlySponsored = slices.Contains(q.Filter, "sponsored")
	p.Search = q.Search
	p.SortBy = q.SortBy
	p.SortDesc = q.SortDir != "asc"
	p.PageOffset = int32(q.Offset)
	if q.Limit > 0 {
		p.PageSize = sql.NullInt32{Int32: int32(q.Limit), Valid: true}
	}
}

// ListGameCards lists a page of the games whose card the caller may see:
// {List, GameCard}, narrowed by q. It returns the page and the number of all
// matches across pages. One query; which grants open which of its fragments
// is in gates.go.
func ListGameCards(ctx context.Context, q GameCardQuery) ([]*GameCard, int, error) {
	if err := q.validate(); err != nil {
		return nil, 0, err
	}
	params := listGameCardsGates(ctx)
	q.wishes(&params)
	rows, err := store.Queries().V2GameCards(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 && params.PageOffset > 0 {
		total, err := countGameCards(ctx, params)
		return nil, total, err
	}
	total := 0
	cards := make([]*GameCard, 0, len(rows))
	for _, r := range rows {
		total = int(r.Total)
		cards = append(cards, gameCardOf(ctx, r))
	}
	return cards, total, nil
}

// ReadGameCard opens one game's card, if the caller may read it:
// {Read, GameCard}. Otherwise the game does not exist for them (ErrNotFound).
func ReadGameCard(ctx context.Context, id uuid.UUID) (*GameCard, error) {
	rows, err := store.Queries().V2GameCards(ctx, readGameCardGates(ctx, id))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return gameCardOf(ctx, rows[0]), nil
}

// gameCardOf is a loaded row as the caller's card, with their rights on it.
func gameCardOf(ctx context.Context, r db.V2GameCardsRow) *GameCard {
	return &GameCard{
		caller:          from(ctx),
		row:             gameRowFrom(r),
		mayReadContent:  r.MayReadContent,
		mayReadSettings: r.MayReadSettings,
		mayEditCard:     r.MayEditCard,
		mayEditContent:  r.MayEditContent,
		mayEditSettings: r.MayEditSettings,
		mayCopy:         r.MayCopy,
		mayShare:        r.MayShare,
		mayPublish:      r.MayPublish,
		mayDelete:       r.MayDelete,
		isFavorite:      r.IsFavorite,
	}
}

// countGameCards is the total of a list whose page lies past its end: the count
// comes with every row, so it reads the first row of the same list.
func countGameCards(ctx context.Context, params db.V2GameCardsParams) (int, error) {
	params.PageOffset = 0
	params.PageSize = sql.NullInt32{Int32: 1, Valid: true}
	rows, err := store.Queries().V2GameCards(ctx, params)
	if err != nil || len(rows) == 0 {
		return 0, err
	}
	return int(rows[0].Total), nil
}

// gameRow is a game's full row, loaded once. It never leaves the core; each
// section's type offers only its own fields.
type gameRow struct {
	// card
	id          uuid.UUID
	name        string
	description string
	creatorName *string // a public profile name
	icon        []byte
	genre       string
	theme       *string
	public      bool
	playCount   int
	cloneCount  int
	createdAt   time.Time
	modifiedAt  time.Time

	// content
	scenario     string
	gameStart    string
	imageStyle   string
	css          string
	statusFields string
	firstMessage *string
	firstStatus  *string
	firstImage   []byte

	// settings
	workshopID          *uuid.UUID
	sponsorID           *uuid.UUID
	creatorID           *uuid.UUID
	originallyCreatedBy *uuid.UUID
}

func gameRowFrom(r db.V2GameCardsRow) gameRow {
	return gameRow{
		id:          r.ID,
		name:        r.Name,
		description: r.Description,
		creatorName: nullString(r.CreatorName),
		icon:        r.Icon,
		genre:       r.Genre,
		theme:       nullString(r.Theme),
		public:      r.Public,
		playCount:   int(r.PlayCount),
		cloneCount:  int(r.CloneCount),
		createdAt:   r.CreatedAt,
		modifiedAt:  r.ModifiedAt,

		scenario:     r.SystemMessageScenario,
		gameStart:    r.SystemMessageGameStart,
		imageStyle:   r.ImageStyle,
		css:          r.Css,
		statusFields: r.StatusFields,
		firstMessage: nullString(r.FirstMessage),
		firstStatus:  nullString(r.FirstStatus),
		firstImage:   r.FirstImage,

		workshopID:          ptrUUID(r.WorkshopID),
		sponsorID:           ptrUUID(r.PublicSponsoredApiKeyShareID),
		creatorID:           ptrUUID(r.CreatedBy),
		originallyCreatedBy: ptrUUID(r.OriginallyCreatedBy),
	}
}

// GameCard is a game's card: what lists show and what is played. It holds the
// full row, the caller it was loaded for and that caller's rights on it, and
// shows only the card.
type GameCard struct {
	caller *identity.Caller
	row    gameRow

	mayReadContent  bool
	mayReadSettings bool
	mayEditCard     bool
	mayEditContent  bool
	mayEditSettings bool
	mayCopy         bool
	mayShare        bool
	mayPublish      bool
	mayDelete       bool

	isFavorite bool // the caller's own mark
}

func (g *GameCard) ID() uuid.UUID         { return g.row.id }
func (g *GameCard) Name() string          { return g.row.name }
func (g *GameCard) Description() string   { return g.row.description }
func (g *GameCard) CreatorName() *string  { return g.row.creatorName }
func (g *GameCard) Icon() []byte          { return g.row.icon }
func (g *GameCard) Genre() string         { return g.row.genre }
func (g *GameCard) Theme() *string        { return g.row.theme }
func (g *GameCard) Public() bool          { return g.row.public }
func (g *GameCard) PlayCount() int        { return g.row.playCount }
func (g *GameCard) CloneCount() int       { return g.row.cloneCount }
func (g *GameCard) CreatedAt() time.Time  { return g.row.createdAt }
func (g *GameCard) ModifiedAt() time.Time { return g.row.modifiedAt }

// MayEditCard says whether the caller may change the game's card:
// {Edit, GameCard}.
func (g *GameCard) MayEditCard() bool { return g.mayEditCard }

// MayEditContent says whether the caller may change how the game works:
// {Edit, GameContent}.
func (g *GameCard) MayEditContent() bool { return g.mayEditContent }

// MayEditSettings says whether the caller may change who is behind the game:
// {Edit, GameSettings}.
func (g *GameCard) MayEditSettings() bool { return g.mayEditSettings }

// MayDelete says whether the caller may delete the game: {Delete, GameCard}.
func (g *GameCard) MayDelete() bool { return g.mayDelete }

// MayCopy says whether the caller may copy the game into a new game of their
// own: {Copy, GameContent}. It answers what to offer; the copy itself checks
// again.
func (g *GameCard) MayCopy() bool { return g.mayCopy }

// MayShare says whether the caller, in their situation, may share the game,
// if sharing is possible for them at all: {Share, GameCard}. Who pays is the
// key share's question.
func (g *GameCard) MayShare() bool { return g.mayShare }

// MayPublish says whether the caller may make the game public:
// {Publish, GameCard}. Making it private again is editing the card.
func (g *GameCard) MayPublish() bool { return g.mayPublish }

// IsFavorite says whether the caller marked the game as a favorite: their own
// mark, no right.
func (g *GameCard) IsFavorite() bool { return g.isFavorite }

// AsGameContent unlocks how the game works: a flag check, no query.
func (g *GameCard) AsGameContent() (*GameContent, error) {
	if !g.mayReadContent {
		return nil, ErrLocked
	}
	return &GameContent{card: g}, nil
}

// AsGameSettings unlocks who is behind the game: a flag check, no query.
func (g *GameCard) AsGameSettings() (*GameSettings, error) {
	if !g.mayReadSettings {
		return nil, ErrLocked
	}
	return &GameSettings{card: g}, nil
}

// GameContent is how a game works: open in the spirit of ChatGameLab.
type GameContent struct{ card *GameCard }

func (c *GameContent) Scenario() string      { return c.card.row.scenario }
func (c *GameContent) GameStart() string     { return c.card.row.gameStart }
func (c *GameContent) ImageStyle() string    { return c.card.row.imageStyle }
func (c *GameContent) CSS() string           { return c.card.row.css }
func (c *GameContent) StatusFields() string  { return c.card.row.statusFields }
func (c *GameContent) FirstMessage() *string { return c.card.row.firstMessage }
func (c *GameContent) FirstStatus() *string  { return c.card.row.firstStatus }
func (c *GameContent) FirstImage() []byte    { return c.card.row.firstImage }

// GameSettings is who is behind a game.
type GameSettings struct{ card *GameCard }

func (s *GameSettings) WorkshopID() *uuid.UUID          { return s.card.row.workshopID }
func (s *GameSettings) SponsorID() *uuid.UUID           { return s.card.row.sponsorID }
func (s *GameSettings) CreatorID() *uuid.UUID           { return s.card.row.creatorID }
func (s *GameSettings) OriginallyCreatedBy() *uuid.UUID { return s.card.row.originallyCreatedBy }
