package wire

import (
	"time"

	"cgl/core"

	"github.com/google/uuid"
)

// ShareLinkView is a share link as the caller sees it: with its token and
// address, the game, where its money comes from, and whether the caller may
// change or stop it. Field names follow v1's enriched game share.
type ShareLinkView struct {
	ID            uuid.UUID  `json:"id"`
	GameID        uuid.UUID  `json:"gameId"`
	GameName      string     `json:"gameName,omitempty"`
	Token         string     `json:"token"`
	ShareURL      string     `json:"shareUrl"`
	ApiKeyShareID uuid.UUID  `json:"apiKeyShareId"`
	InstitutionID *uuid.UUID `json:"institutionId"`
	WorkshopID    *uuid.UUID `json:"workshopId"`
	WorkshopName  string     `json:"workshopName,omitempty"`
	// Remaining plays; null: unlimited.
	Remaining     *int32     `json:"remaining"`
	AiQualityTier *string    `json:"aiQualityTier,omitempty"`
	CreatedBy     *uuid.UUID `json:"createdBy"`
	CreatedAt     time.Time  `json:"createdAt"`
	// Source is where its money comes from: "workshop" (a workshop's key),
	// "organization" (an organisation's key) or "personal" (someone's own).
	Source    string `json:"source"`
	KeyName   string `json:"keyName"`
	MayManage bool   `json:"mayManage"`
	// Embed marks a game's embed link, which plays framed on another
	// website; Enabled is false while embedding is switched off.
	Embed   bool `json:"embed"`
	Enabled bool `json:"enabled"`
}

// ShareLinkViewOf converts one share link, passing a loading error through.
func ShareLinkViewOf(l *core.ShareLinkCard, err error) (ShareLinkView, error) {
	if err != nil {
		return ShareLinkView{}, err
	}
	return shareLinkView(l), nil
}

// ShareLinkViewsOf converts a list of share links, passing a loading error
// through.
func ShareLinkViewsOf(links []*core.ShareLinkCard, err error) ([]ShareLinkView, error) {
	if err != nil {
		return nil, err
	}
	views := make([]ShareLinkView, len(links))
	for i, l := range links {
		views[i] = shareLinkView(l)
	}
	return views, nil
}

func shareLinkView(l *core.ShareLinkCard) ShareLinkView {
	source := "personal"
	switch {
	case l.WorkshopID != nil:
		source = "workshop"
	case l.PayingOrganisation != nil:
		source = "organization"
	}
	v := ShareLinkView{
		ID: l.ID, GameID: l.GameID, GameName: l.GameName, Token: l.Token, ShareURL: "/play/" + l.Token,
		ApiKeyShareID: l.ShareID, InstitutionID: l.PayingOrganisation, WorkshopID: l.WorkshopID,
		Remaining: l.Remaining, AiQualityTier: l.Tier, CreatedBy: l.CreatedBy, CreatedAt: l.CreatedAt,
		Source: source, KeyName: l.KeyName, MayManage: l.MayManage(),
		Embed: l.Embed, Enabled: l.Enabled,
	}
	if l.Embed {
		v.ShareURL = ""
	}
	if l.WorkshopName != nil {
		v.WorkshopName = *l.WorkshopName
	}
	return v
}
