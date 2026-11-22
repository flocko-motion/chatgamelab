// package: wire / the API's output formats
// type:    logic
// job:     converts the secure core's entities into the JSON the API sends
// limits:  decides nothing: it reads only what the core's types let through (getters, unlocked sections); every rights decision is the core's
package wire

import (
	"time"

	"cgl/core"
	"cgl/obj"

	"github.com/google/uuid"
)

// GameView is a game as the caller sees it: the card always; content and
// settings only when the caller may see them, which hasContent and hasSettings
// say. Field names follow v1's game, so clients read the same keys.
type GameView struct {
	ID          uuid.UUID `json:"id"`
	Meta        GameMeta  `json:"meta"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatorName *string   `json:"creatorName,omitempty"`
	Icon        []byte    `json:"icon"`
	Genre       string    `json:"genre"`
	Theme       *string   `json:"theme,omitempty"`
	Public      bool      `json:"public"`
	PlayCount   int       `json:"playCount"`
	CloneCount  int       `json:"cloneCount"`
	// CoverAt is when the game's cover was made; absent while it has none.
	// The picture is GET /api/games/{id}/cover, with ?v= this time to bust
	// caches when it changes.
	CoverAt *time.Time `json:"coverAt,omitempty"`

	// The caller's rights on the game, by section, and on the whole game.
	MayEditCard     bool `json:"mayEditCard"`
	MayEditContent  bool `json:"mayEditContent"`
	MayEditSettings bool `json:"mayEditSettings"`
	MayDelete       bool `json:"mayDelete"`
	MayCopy         bool `json:"mayCopy"`
	MayShare        bool `json:"mayShare"`   // in the caller's situation, if sharing is possible at all
	MayPublish      bool `json:"mayPublish"` // making it private again is editing the card

	IsFavorite bool `json:"isFavorite"` // the caller's own mark

	// HasContent: the caller sees how the game works; the fields below are set.
	HasContent             bool    `json:"hasContent"`
	SystemMessageScenario  *string `json:"systemMessageScenario,omitempty"`
	SystemMessageGameStart *string `json:"systemMessageGameStart,omitempty"`
	ImageStyle             *string `json:"imageStyle,omitempty"`
	CSS                    *string `json:"css,omitempty"`
	StatusFields           *string `json:"statusFields,omitempty"`
	FirstMessage           *string `json:"firstMessage,omitempty"`
	FirstStatus            *string `json:"firstStatus,omitempty"`
	FirstImage             []byte  `json:"firstImage,omitempty"`

	// HasSettings: the caller sees who is behind the game; the fields below are set.
	HasSettings                  bool       `json:"hasSettings"`
	WorkshopID                   *uuid.UUID `json:"workshopId,omitempty"`
	PublicSponsoredApiKeyShareID *uuid.UUID `json:"publicSponsoredApiKeyShareId,omitempty"`
	CreatorID                    *uuid.UUID `json:"creatorId,omitempty"`
	OriginallyCreatedBy          *uuid.UUID `json:"originallyCreatedBy,omitempty"`
}

// GameMeta is when a game was made and changed, and by whom when the caller
// sees its settings.
type GameMeta struct {
	CreatedAt  time.Time  `json:"createdAt"`
	ModifiedAt time.Time  `json:"modifiedAt"`
	CreatedBy  *uuid.UUID `json:"createdBy,omitempty"`
}

// GameViewOf converts one game card, passing a loading error through:
//
//	return wire.GameViewOf(core.ReadGameCard(ctx, id))
func GameViewOf(card *core.GameCard, err error) (GameView, error) {
	if err != nil {
		return GameView{}, err
	}
	return gameView(card), nil
}

// gameView converts a game card, with every section the caller may see. It
// unlocks the sections on card itself, so a view never mixes two games.
func gameView(card *core.GameCard) GameView {
	v := GameView{
		ID:              card.ID(),
		Meta:            GameMeta{CreatedAt: card.CreatedAt(), ModifiedAt: card.ModifiedAt()},
		Name:            card.Name(),
		Description:     card.Description(),
		CreatorName:     card.CreatorName(),
		Icon:            card.Icon(),
		Genre:           card.Genre(),
		Theme:           card.Theme(),
		Public:          card.Public(),
		PlayCount:       card.PlayCount(),
		CloneCount:      card.CloneCount(),
		CoverAt:         card.CoverAt(),
		MayEditCard:     card.MayEditCard(),
		MayEditContent:  card.MayEditContent(),
		MayEditSettings: card.MayEditSettings(),
		MayDelete:       card.MayDelete(),
		MayCopy:         card.MayCopy(),
		MayShare:        card.MayShare(),
		MayPublish:      card.MayPublish(),
		IsFavorite:      card.IsFavorite(),
	}
	if c, err := card.AsGameContent(); err == nil {
		v.HasContent = true
		v.SystemMessageScenario = ptr(c.Scenario())
		v.SystemMessageGameStart = ptr(c.GameStart())
		v.ImageStyle = ptr(c.ImageStyle())
		v.CSS = ptr(c.CSS())
		v.StatusFields = ptr(c.StatusFields())
		v.FirstMessage = c.FirstMessage()
		v.FirstStatus = c.FirstStatus()
		v.FirstImage = c.FirstImage()
	}
	if s, err := card.AsGameSettings(); err == nil {
		v.HasSettings = true
		v.WorkshopID = s.WorkshopID()
		v.PublicSponsoredApiKeyShareID = s.SponsorID()
		v.CreatorID = s.CreatorID()
		v.OriginallyCreatedBy = s.OriginallyCreatedBy()
		v.Meta.CreatedBy = s.CreatorID()
	}
	return v
}

// GameViewPageOf converts a page of game cards, passing a loading error
// through:
//
//	return wire.GameViewPageOf(core.List(ctx, core.GameCards, q))
func GameViewPageOf(p core.Page[*core.GameCard], err error) (Page[GameView], error) {
	return pageOf(p, err, gameView)
}

func ptr[T any](v T) *T { return &v }

// GameTemplate is a game not yet created: what a new game starts with and
// what the caller may do with it. It is a strict subset of GameView, same
// names and types, so one form fills from either; it has no id, no meta, no
// counts and no settings, because no game exists yet.
type GameTemplate struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Icon        []byte  `json:"icon"`
	Genre       string  `json:"genre"`
	Theme       *string `json:"theme,omitempty"`
	Public      bool    `json:"public"`

	MayEditCard    bool `json:"mayEditCard"`
	MayEditContent bool `json:"mayEditContent"`
	MayPublish     bool `json:"mayPublish"`

	HasContent             bool    `json:"hasContent"` // always: the creator sees their own game's content
	SystemMessageScenario  *string `json:"systemMessageScenario,omitempty"`
	SystemMessageGameStart *string `json:"systemMessageGameStart,omitempty"`
	ImageStyle             *string `json:"imageStyle,omitempty"`
	CSS                    *string `json:"css,omitempty"`
	StatusFields           *string `json:"statusFields,omitempty"`
	FirstMessage           *string `json:"firstMessage,omitempty"`
	FirstStatus            *string `json:"firstStatus,omitempty"`
	FirstImage             []byte  `json:"firstImage,omitempty"`
}

// GameTemplateOf converts a game template, passing a loading error through:
//
//	return wire.GameTemplateOf(core.NewGame(ctx))
func GameTemplateOf(t *core.GameTemplate, err error) (GameTemplate, error) {
	if err != nil {
		return GameTemplate{}, err
	}
	return GameTemplate{
		Name:                   t.Name(),
		Description:            t.Description(),
		Icon:                   t.Icon(),
		Genre:                  t.Genre(),
		Theme:                  t.Theme(),
		Public:                 t.Public(),
		MayEditCard:            t.MayEditCard(),
		MayEditContent:         t.MayEditContent(),
		MayPublish:             t.MayPublish(),
		HasContent:             true,
		SystemMessageScenario:  ptr(t.Scenario()),
		SystemMessageGameStart: ptr(t.GameStart()),
		ImageStyle:             ptr(t.ImageStyle()),
		CSS:                    ptr(t.CSS()),
		StatusFields:           ptr(t.StatusFields()),
		FirstMessage:           t.FirstMessage(),
		FirstStatus:            t.FirstStatus(),
		FirstImage:             t.FirstImage(),
	}, nil
}

// GameFileOf converts a game card into a game file: its card and content.
// A game whose content the caller may not see is refused (core.ErrLocked):
// a file without content is no game.
//
//	return wire.GameFileOf(core.ReadGameCard(ctx, id))
func GameFileOf(card *core.GameCard, err error) (obj.GameFile, error) {
	if err != nil {
		return obj.GameFile{}, err
	}
	content, err := card.AsGameContent()
	if err != nil {
		return obj.GameFile{}, err
	}
	return obj.GameFile{
		ID:                     card.ID(),
		Name:                   card.Name(),
		Description:            card.Description(),
		Genre:                  card.Genre(),
		Theme:                  card.Theme(),
		SystemMessageScenario:  content.Scenario(),
		SystemMessageGameStart: content.GameStart(),
		ImageStyle:             content.ImageStyle(),
		CSS:                    content.CSS(),
		StatusFields:           content.StatusFields(),
	}, nil
}
