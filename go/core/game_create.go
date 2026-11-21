package core

import (
	"context"
	"strings"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"engine"

	"github.com/google/uuid"
)

// A new game is the caller's own from the start: its rights are the caller's
// Own grants, read straight from the table. No row exists to gate, so these
// checks live here, not in gates.go.

// GameTemplate is a game not yet created: what a new game starts with and
// what the caller may do with it. It is no game: it has no id and nothing is
// stored. The caller fills it in; creating the game is the first submission
// (CreateGame), which checks everything again.
type GameTemplate struct {
	row            gameRow
	mayEditCard    bool
	mayEditContent bool
	mayPublish     bool
}

// NewGame is the template for a game the caller creates: {Create, GameCard}.
func NewGame(ctx context.Context) (*GameTemplate, error) {
	if !granted(ctx, rules.Create, rules.GameCard, rules.Own) {
		return nil, forbidden("you may not create games")
	}
	return &GameTemplate{
		row:            gameRow{genre: string(engine.GenreAdventure)},
		mayEditCard:    granted(ctx, rules.Edit, rules.GameCard, rules.Own),
		mayEditContent: granted(ctx, rules.Edit, rules.GameContent, rules.Own),
		mayPublish:     granted(ctx, rules.Publish, rules.GameCard, rules.Own),
	}, nil
}

// CreateGame creates a game from the template, filled in with ch, and
// returns it as stored. It needs a name; making it public takes {Publish,
// GameCard} on an own game. In a workshop, the game belongs to it.
func CreateGame(ctx context.Context, ch GameChanges) (*GameCard, error) {
	t, err := NewGame(ctx)
	if err != nil {
		return nil, err
	}
	next := t.row
	ch.applyCard(&next)
	ch.applyContent(&next)

	switch {
	case strings.TrimSpace(next.name) == "":
		return nil, refuse("a game needs a name")
	case next.public && !t.mayPublish:
		return nil, forbidden("you may not make this game public")
	}
	if err := next.validate(); err != nil {
		return nil, err
	}
	return insertGame(ctx, next, nil)
}

// CopyGame copies a game into a new private game of the caller's and returns
// the copy. It takes the right to copy the game, {Copy, GameContent}, and to
// create one, {Create, GameCard}. The copy takes the card and content, not
// the images; it names whoever first created the game.
func CopyGame(ctx context.Context, id uuid.UUID) (*GameCard, error) {
	source, err := ReadGameCard(ctx, id)
	if err != nil {
		return nil, err
	}
	if !source.MayCopy() {
		return nil, forbidden("you may not copy this game")
	}
	if _, err := NewGame(ctx); err != nil {
		return nil, err
	}
	content, err := source.AsGameContent()
	if err != nil {
		return nil, err
	}

	from := source.row
	next := gameRow{
		name:         copyName(from.name),
		description:  from.description,
		genre:        from.genre,
		theme:        from.theme,
		scenario:     content.Scenario(),
		gameStart:    content.GameStart(),
		imageStyle:   content.ImageStyle(),
		css:          content.CSS(),
		statusFields: content.StatusFields(),
		firstMessage: content.FirstMessage(),
		firstStatus:  content.FirstStatus(),
	}
	original := from.originallyCreatedBy // carries over from copy to copy
	if original == nil {
		original = from.creatorID
	}

	copied, err := insertGame(ctx, next, original)
	if err != nil {
		return nil, err
	}
	if err := store.Queries().IncrementGameCloneCount(ctx, id); err != nil {
		return nil, err
	}
	return copied, nil
}

// copyName is a copy's name: the original's, marked, within the name's limit.
func copyName(name string) string {
	const mark = " (Copy)"
	runes := []rune(name)
	for len(string(runes))+len(mark) > maxGameName && len(runes) > 0 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + mark
}

// insertGame stores a new game of the caller's, in the workshop they are in,
// and returns it as they see it. The caller must have been checked already.
func insertGame(ctx context.Context, next gameRow, originallyCreatedBy *uuid.UUID) (*GameCard, error) {
	id := uuid.New()
	by := account(ctx).UUID // whoever may create has an account
	workshop := currentWorkshop(ctx)
	err := store.Queries().V2CreateGame(ctx, db.V2CreateGameParams{
		ID:                  id,
		CreatedBy:           by,
		WorkshopID:          workshop,
		OriginallyCreatedBy: nullUUID(originallyCreatedBy),
		Name:                next.name,
		Description:         next.description,
		Icon:                next.icon,
		Genre:               next.genre,
		Theme:               sqlString(next.theme),
		Public:              next.public,
		Scenario:            next.scenario,
		GameStart:           next.gameStart,
		ImageStyle:          next.imageStyle,
		Css:                 next.css,
		StatusFields:        next.statusFields,
		FirstMessage:        sqlString(next.firstMessage),
		FirstStatus:         sqlString(next.firstStatus),
		FirstImage:          next.firstImage,
	})
	if err != nil {
		return nil, err
	}
	db.AfterGameCreated(id, ptrUUID(workshop), by)
	return ReadGameCard(ctx, id)
}

func (t *GameTemplate) Name() string        { return t.row.name }
func (t *GameTemplate) Description() string { return t.row.description }
func (t *GameTemplate) Icon() []byte        { return t.row.icon }
func (t *GameTemplate) Genre() string       { return t.row.genre }
func (t *GameTemplate) Theme() *string      { return t.row.theme }
func (t *GameTemplate) Public() bool        { return t.row.public }

func (t *GameTemplate) Scenario() string      { return t.row.scenario }
func (t *GameTemplate) GameStart() string     { return t.row.gameStart }
func (t *GameTemplate) ImageStyle() string    { return t.row.imageStyle }
func (t *GameTemplate) CSS() string           { return t.row.css }
func (t *GameTemplate) StatusFields() string  { return t.row.statusFields }
func (t *GameTemplate) FirstMessage() *string { return t.row.firstMessage }
func (t *GameTemplate) FirstStatus() *string  { return t.row.firstStatus }
func (t *GameTemplate) FirstImage() []byte    { return t.row.firstImage }

// MayEditCard, MayEditContent and MayPublish are the caller's rights on the
// game once created: their Own grants.
func (t *GameTemplate) MayEditCard() bool    { return t.mayEditCard }
func (t *GameTemplate) MayEditContent() bool { return t.mayEditContent }
func (t *GameTemplate) MayPublish() bool     { return t.mayPublish }
