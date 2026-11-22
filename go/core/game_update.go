package core

import (
	"bytes"
	"context"
	"slices"
	"strings"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/events"
	"cgl/log"
	"engine"

	"github.com/google/uuid"
)

// GameChanges is what the caller asks to change in a game. A nil field stays
// as it is, so a client may send only what it changes, or the whole game
// back: only what differs from the stored game counts as a change. The
// settings are not here: who is behind a game changes through actions of
// their own (sponsoring, sharing).
type GameChanges struct {
	// card
	Name        *string
	Description *string
	Icon        []byte  // nil: unchanged; empty: removed
	Genre       *string // "": unchanged
	Theme       *string // "": the engine's default
	Public      *bool

	// content
	Scenario     *string
	GameStart    *string
	ImageStyle   *string
	CSS          *string
	StatusFields *string
	FirstMessage *string // "": none
	FirstStatus  *string // "": none
	FirstImage   []byte  // nil: unchanged; empty: removed
}

// UpdateGame changes a game as the caller asks and returns it as stored.
//
// A game the caller may not read does not exist for them (ErrNotFound). Each
// section the request changes must be one the caller may edit: {Edit,
// GameCard}, {Edit, GameContent}; making the game public takes {Publish,
// GameCard}. One refused change refuses the whole request, so nothing is ever
// half written. A request that changes nothing writes nothing.
func UpdateGame(ctx context.Context, id uuid.UUID, ch GameChanges) (*GameCard, error) {
	game, err := ReadGameCard(ctx, id)
	if err != nil {
		return nil, err
	}
	by := account(ctx).UUID // whoever may edit has an account: a guest owns nothing

	next := game.row
	cardChanged := ch.applyCard(&next)
	contentChanged := ch.applyContent(&next)
	publishing := next.public && !game.row.public

	switch {
	case !cardChanged && !contentChanged:
		return game, nil
	case cardChanged && !game.MayEditCard():
		return nil, forbidden("you may not edit this game's card")
	case contentChanged && !game.MayEditContent():
		return nil, forbidden("you may not edit this game's content")
	case publishing && !game.MayPublish():
		return nil, forbidden("you may not make this game public")
	}
	if err := next.validate(); err != nil {
		return nil, err
	}

	err = store.Queries().V2UpdateGame(ctx, db.V2UpdateGameParams{
		ID:           id,
		ModifiedBy:   by,
		Name:         next.name,
		Description:  next.description,
		Icon:         next.icon,
		Genre:        next.genre,
		Theme:        sqlString(next.theme),
		Public:       next.public,
		Scenario:     next.scenario,
		GameStart:    next.gameStart,
		ImageStyle:   next.imageStyle,
		Css:          next.css,
		StatusFields: next.statusFields,
		FirstMessage: sqlString(next.firstMessage),
		FirstStatus:  sqlString(next.firstStatus),
		FirstImage:   next.firstImage,
	})
	if isDuplicateGameName(err) {
		return nil, errDuplicateGameName
	}
	if err != nil {
		return nil, err
	}

	if game.row.public && !next.public {
		// a private game leaves the workshops' public pages
		if err := endPublicPageLinks(ctx, nil, &id); err != nil {
			log.Warn("failed to end a game's public page links", "game_id", id, "error", err)
		}
	}
	if w := game.row.workshopID; w != nil {
		events.GetBroker().PublishGameUpdated(*w, id, by) // the workshop hears of it
	}
	return ReadGameCard(ctx, id)
}

// applyCard writes the card's changes into row and says whether any differs.
func (ch GameChanges) applyCard(row *gameRow) bool {
	changed := false
	set(&row.name, ch.Name, &changed)
	set(&row.description, ch.Description, &changed)
	setBytes(&row.icon, ch.Icon, &changed)
	if ch.Genre != nil && *ch.Genre != "" {
		set(&row.genre, ch.Genre, &changed)
	}
	setOptional(&row.theme, ch.Theme, &changed)
	set(&row.public, ch.Public, &changed)
	return changed
}

// applyContent writes the content's changes into row and says whether any
// differs.
func (ch GameChanges) applyContent(row *gameRow) bool {
	changed := false
	set(&row.scenario, ch.Scenario, &changed)
	set(&row.gameStart, ch.GameStart, &changed)
	set(&row.imageStyle, ch.ImageStyle, &changed)
	set(&row.css, ch.CSS, &changed)
	set(&row.statusFields, ch.StatusFields, &changed)
	setOptional(&row.firstMessage, ch.FirstMessage, &changed)
	setOptional(&row.firstStatus, ch.FirstStatus, &changed)
	setBytes(&row.firstImage, ch.FirstImage, &changed)
	return changed
}

// maxGameName is the longest a game's name may be, in bytes.
const maxGameName = 70

// validate refuses a game no one may store, whoever asks.
func (r gameRow) validate() error {
	if len(strings.TrimSpace(r.name)) > maxGameName {
		return nameTooLong("a game's name is at most %d characters", maxGameName)
	}
	if !slices.Contains(engine.Genres(), engine.Genre(r.genre)) {
		return refuse("unknown genre %q", r.genre)
	}
	return nil
}

// set writes to into field if it is given and differs.
func set[T comparable](field *T, to *T, changed *bool) {
	if to != nil && *to != *field {
		*field, *changed = *to, true
	}
}

// setOptional writes to into an optional field if it is given and differs;
// "" removes it.
func setOptional(field **string, to *string, changed *bool) {
	if to == nil {
		return
	}
	var next *string
	if *to != "" {
		next = to
	}
	if (next == nil) != (*field == nil) || (next != nil && *next != **field) {
		*field, *changed = next, true
	}
}

// setBytes writes to into field if it is given and differs.
func setBytes(field *[]byte, to []byte, changed *bool) {
	if to != nil && !bytes.Equal(to, *field) {
		*field, *changed = to, true
	}
}
