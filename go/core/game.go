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

// GameCards is the game list: {List, GameCard}, the games whose card the
// caller may see. Its orders are card fields only: sorting by a locked
// field would reveal it through the order.
var GameCards Lister[*GameCard] = gameCards{}

type gameCards struct{}

// genreFamily names the filters that keep one genre's games, genre:<id>.
// Several of them are one filter, met by a game of any of the genres.
const genreFamily = "genre:"

// Spec: the filters own, others, organisation, public, favorites and
// sponsored, and genre:<id>; the orders createdAt (default, newest first),
// modifiedAt, name, playCount, visibility and creator.
func (gameCards) Spec() ListSpec {
	return ListSpec{
		Filters:        []string{"own", "others", "organisation", "public", "favorites", "sponsored"},
		FilterFamilies: []string{genreFamily},
		Sorts:          []string{"createdAt", "modifiedAt", "name", "playCount", "visibility", "creator"},
		DefaultSort:    "createdAt",
		DefaultDir:     "desc",
	}
}

// Query lists a page in one query, with the number of all matches; which
// grants open which of its fragments is in gates.go.
func (gameCards) Query(ctx context.Context, q ListQuery) (Page[*GameCard], error) {
	params := listGameCardsGates(ctx)
	p := &params
	p.OnlyOwn = slices.Contains(q.Filter, "own")
	p.OnlyOthers = slices.Contains(q.Filter, "others")
	p.OnlyOrganisation = slices.Contains(q.Filter, "organisation")
	p.OnlyPublic = slices.Contains(q.Filter, "public")
	p.OnlyFavorites = slices.Contains(q.Filter, "favorites")
	p.OnlySponsored = slices.Contains(q.Filter, "sponsored")
	for _, f := range q.Filter {
		if genre, ok := strings.CutPrefix(f, genreFamily); ok {
			p.Genres = append(p.Genres, genre)
		}
	}
	p.Search = q.Search
	p.SortBy = q.SortBy
	p.SortDesc = q.SortDir == "desc"
	p.PageOffset = int32(q.Offset)
	if q.Limit > 0 {
		p.PageSize = sql.NullInt32{Int32: int32(q.Limit), Valid: true}
	}
	rows, err := store.Queries().V2GameCards(ctx, params)
	if err != nil {
		return Page[*GameCard]{}, err
	}
	if len(rows) == 0 && params.PageOffset > 0 {
		total, err := countGameCards(ctx, params)
		return Page[*GameCard]{Total: total}, err
	}
	page := Page[*GameCard]{Items: make([]*GameCard, 0, len(rows))}
	for _, r := range rows {
		page.Total = int(r.Total)
		page.Items = append(page.Items, gameCardOf(ctx, r))
	}
	return page, nil
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

// ReadGameCardByLink opens the game behind a share link's token, for
// whoever holds it: {Read, GameCard, ViaShareLink}, and its content and
// play by the link's grants. Otherwise it does not exist for them
// (ErrNotFound).
func ReadGameCardByLink(ctx context.Context, id uuid.UUID, token string) (*GameCard, error) {
	p := readGameCardGates(ctx, id)
	p.ShareToken = sql.NullString{String: token, Valid: true}
	rows, err := store.Queries().V2GameCards(ctx, p)
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
		row:             gameRowFrom(ctx, r),
		mayReadContent:  r.MayReadContent,
		mayReadSettings: r.MayReadSettings,
		mayEditCard:     r.MayEditCard,
		mayEditContent:  r.MayEditContent,
		mayEditSettings: r.MayEditSettings,
		mayCopy:         r.MayCopy,
		mayShare:        r.MayShare,
		mayPublish:      r.MayPublish,
		mayPlay:         r.MayPlay,
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
	coverAt     *time.Time // when its cover was made; nil: none yet

	// content
	scenario     string
	gameStart    string
	imageStyle   string
	voiceStyle   string
	voices       map[string]string // voice slot: choice; a slot left out is any
	css          string
	statusFields string
	firstMessage *string
	firstStatus  *string
	firstImage   []byte

	// settings
	workshopID          *uuid.UUID
	organisationID      *uuid.UUID // the organisation it belongs to, seen from the caller
	sponsorID           *uuid.UUID
	creatorID           *uuid.UUID
	originallyCreatedBy *uuid.UUID
}

func gameRowFrom(ctx context.Context, r db.V2GameCardsRow) gameRow {
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
		coverAt:     nullTime(r.CoverAt),

		scenario:     r.SystemMessageScenario,
		gameStart:    r.SystemMessageGameStart,
		imageStyle:   r.ImageStyle,
		voiceStyle:   r.VoiceStyle,
		voices:       voicesFrom(r.Voices),
		css:          r.Css,
		statusFields: r.StatusFields,
		firstMessage: nullString(r.FirstMessage),
		firstStatus:  nullString(r.FirstStatus),
		firstImage:   r.FirstImage,

		workshopID:          ptrUUID(r.WorkshopID),
		organisationID:      gameOrganisationOf(ctx, r),
		sponsorID:           ptrUUID(r.PublicSponsoredApiKeyShareID),
		creatorID:           ptrUUID(r.CreatedBy),
		originallyCreatedBy: ptrUUID(r.OriginallyCreatedBy),
	}
}

// gameOrganisationOf is the organisation a game belongs to, seen from the
// caller: their own when it is theirs (it may be two organisations', its
// workshop's and its creator's), otherwise its workshop's, otherwise the one
// its creator is head or staff of; nil for an individual's game.
func gameOrganisationOf(ctx context.Context, r db.V2GameCardsRow) *uuid.UUID {
	switch {
	case r.OfOwnOrganisation:
		return from(ctx).Organisation()
	case r.WorkshopOrganisationID.Valid:
		return ptrUUID(r.WorkshopOrganisationID)
	default:
		return ptrUUID(r.CreatorOrganisationID)
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
	mayPlay         bool
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

// CoverAt is when the game's cover was made, nil while it has none. The
// picture itself is ReadGameCover's.
func (g *GameCard) CoverAt() *time.Time { return g.row.coverAt }

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

// MayPlay says whether the caller may start a session of it: {Play,
// GameCard}.
func (g *GameCard) MayPlay() bool { return g.mayPlay }

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

func (c *GameContent) Scenario() string   { return c.card.row.scenario }
func (c *GameContent) GameStart() string  { return c.card.row.gameStart }
func (c *GameContent) ImageStyle() string { return c.card.row.imageStyle }

// VoiceStyle is how the game's voice sounds, in the designer's words; empty
// leaves the voice to the engine.
func (c *GameContent) VoiceStyle() string { return c.card.row.voiceStyle }

// Voices is the game's choice for each voice its genre speaks in, by the
// voice's name; a voice left out is any.
func (c *GameContent) Voices() map[string]string { return cloneVoices(c.card.row.voices) }

func (c *GameContent) CSS() string           { return c.card.row.css }
func (c *GameContent) StatusFields() string  { return c.card.row.statusFields }
func (c *GameContent) FirstMessage() *string { return c.card.row.firstMessage }
func (c *GameContent) FirstStatus() *string  { return c.card.row.firstStatus }
func (c *GameContent) FirstImage() []byte    { return c.card.row.firstImage }

// GameSettings is who is behind a game.
type GameSettings struct{ card *GameCard }

func (s *GameSettings) WorkshopID() *uuid.UUID          { return s.card.row.workshopID }
func (s *GameSettings) OrganisationID() *uuid.UUID      { return s.card.row.organisationID }
func (s *GameSettings) SponsorID() *uuid.UUID           { return s.card.row.sponsorID }
func (s *GameSettings) CreatorID() *uuid.UUID           { return s.card.row.creatorID }
func (s *GameSettings) OriginallyCreatedBy() *uuid.UUID { return s.card.row.originallyCreatedBy }
