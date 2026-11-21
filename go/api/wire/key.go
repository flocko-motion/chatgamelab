package wire

import (
	"time"

	"cgl/core"

	"github.com/google/uuid"
)

// ApiKeyView is a key as its shares show it: never its secret, only its
// last four characters. Field names follow v1's key.
type ApiKeyView struct {
	ID               uuid.UUID `json:"id"`
	Meta             KeyMeta   `json:"meta"`
	Name             string    `json:"name"`
	UserID           uuid.UUID `json:"userId"`
	UserName         string    `json:"userName"`
	Platform         string    `json:"platform"`
	KeyShortened     string    `json:"keyShortened"`
	IsDefault        bool      `json:"isDefault"`
	LastUsageSuccess *bool     `json:"lastUsageSuccess"`
	LastErrorCode    *string   `json:"lastErrorCode,omitempty"`
}

// KeyMeta is when a key was added and last changed.
type KeyMeta struct {
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// KeyShareView is a share of a key as the caller sees it: the owner's own
// use of it (self), given to an organisation, or drawn for a sponsored game
// or a share link. Field names follow v1's share.
type KeyShareView struct {
	ID            uuid.UUID    `json:"id"`
	Meta          KeyShareMeta `json:"meta"`
	ApiKeyID      uuid.UUID    `json:"apiKeyId"`
	ApiKey        ApiKeyView   `json:"apiKey"`
	SourceShareID *uuid.UUID   `json:"sourceShareId,omitempty"`
	Self          bool         `json:"self"`
	Institution   *NamedRef    `json:"institution,omitempty"`
	Game          *NamedRef    `json:"game,omitempty"`
	// IsPrivateShare: it pays for a share link (gameShareId), with
	// remaining plays left; none: unlimited.
	IsPrivateShare bool       `json:"isPrivateShare,omitempty"`
	GameShareID    *uuid.UUID `json:"gameShareId,omitempty"`
	Remaining      *int32     `json:"remaining,omitempty"`

	// The caller's rights on it: drawing sponsorships and share links from
	// it, and taking it back.
	MayShare  bool `json:"mayShare"`
	MayRevoke bool `json:"mayRevoke"`
}

// KeyShareMeta is when a share was made, and by whom.
type KeyShareMeta struct {
	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy *uuid.UUID `json:"createdBy,omitempty"`
}

// KeyShareViewOf converts one share, passing a loading error through.
func KeyShareViewOf(s *core.KeyShareCard, err error) (KeyShareView, error) {
	if err != nil {
		return KeyShareView{}, err
	}
	return keyShareView(s), nil
}

// KeyShareViewsOf converts a list of shares, passing a loading error
// through.
func KeyShareViewsOf(shares []*core.KeyShareCard, err error) ([]KeyShareView, error) {
	if err != nil {
		return nil, err
	}
	views := make([]KeyShareView, len(shares))
	for i, s := range shares {
		views[i] = keyShareView(s)
	}
	return views, nil
}

// ApiKeyViewOf is the key behind a share.
func ApiKeyViewOf(s *core.KeyShareCard) ApiKeyView {
	k := s.Key
	return ApiKeyView{
		ID: k.ID, Meta: KeyMeta{CreatedAt: k.CreatedAt, ModifiedAt: k.ModifiedAt}, Name: k.Name, UserID: k.OwnerID, UserName: k.OwnerName, Platform: k.Platform,
		KeyShortened: k.Shortened(), IsDefault: k.Default,
		LastUsageSuccess: k.LastUsageSuccess, LastErrorCode: k.LastErrorCode,
	}
}

func keyShareView(s *core.KeyShareCard) KeyShareView {
	v := KeyShareView{
		ID:             s.ID,
		Meta:           KeyShareMeta{CreatedAt: s.CreatedAt, CreatedBy: s.CreatedBy},
		ApiKeyID:       s.Key.ID,
		ApiKey:         ApiKeyViewOf(s),
		SourceShareID:  s.Source,
		Self:           s.Self,
		Institution:    namedRef(s.OrganisationID, s.OrganisationName),
		Game:           namedRef(s.GameID, s.GameName),
		IsPrivateShare: s.LinkID != nil,
		GameShareID:    s.LinkID,
		Remaining:      s.LinkRemaining,
		MayShare:       s.MayShare(),
		MayRevoke:      s.MayRevoke(),
	}
	return v
}
