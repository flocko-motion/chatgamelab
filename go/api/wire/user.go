package wire

import (
	"time"

	"cgl/core"

	"github.com/google/uuid"
)

// UserView is a user as the caller sees them: the card always; account and
// contact only when the caller may see them, which hasAccount and
// hasContact say. Field names follow v1's user, so clients read the same
// keys. A user's login code is never part of it: it has its own view
// (UserSecretsView), fetched on purpose.
type UserView struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`

	// The caller's rights on the user, by section, and on the whole user.
	MayEditCard    bool `json:"mayEditCard"`
	MayEditAccount bool `json:"mayEditAccount"`
	MayEditContact bool `json:"mayEditContact"`
	MayDelete      bool `json:"mayDelete"`

	// HasAccount: the caller sees how the account is set up; the fields
	// below are set.
	HasAccount           bool         `json:"hasAccount"`
	Meta                 *UserMeta    `json:"meta,omitempty"`
	Language             *string      `json:"language,omitempty"`
	AgeGroup             *string      `json:"ageGroup,omitempty"`
	AiQualityTier        *string      `json:"aiQualityTier,omitempty"`
	DefaultApiKeyShareID *uuid.UUID   `json:"defaultApiKeyShareId,omitempty"`
	Role                 *UserRoleRef `json:"role,omitempty"`

	// HasContact: the caller sees how to reach the user; email is set when
	// they have one.
	HasContact bool    `json:"hasContact"`
	Email      *string `json:"email,omitempty"`
}

// UserMeta is when an account was made and last changed.
type UserMeta struct {
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// UserRoleRef is a user's role, with the organisation it belongs to and the
// workshop they are in. Only names and ids: what an organisation or a
// workshop is otherwise is theirs to show.
type UserRoleRef struct {
	Role        string    `json:"role"`
	Institution *NamedRef `json:"institution,omitempty"`
	Workshop    *NamedRef `json:"workshop,omitempty"`
}

// NamedRef names another entity: its id and name.
type NamedRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// UserViewOf converts one user card, passing a loading error through:
//
//	return wire.UserViewOf(core.ReadUser(ctx, id))
func UserViewOf(user *core.UserCard, err error) (UserView, error) {
	if err != nil {
		return UserView{}, err
	}
	return userView(user), nil
}

// UserViewsOf converts a list of user cards, passing a loading error through.
func UserViewsOf(users []*core.UserCard, err error) ([]UserView, error) {
	if err != nil {
		return nil, err
	}
	views := make([]UserView, len(users))
	for i, u := range users {
		views[i] = userView(u)
	}
	return views, nil
}

// userView converts a user card, with every section the caller may see.
func userView(user *core.UserCard) UserView {
	v := UserView{
		ID:             user.ID(),
		Name:           user.Name(),
		MayEditCard:    user.MayEditCard(),
		MayEditAccount: user.MayEditAccount(),
		MayEditContact: user.MayEditContact(),
		MayDelete:      user.MayDelete(),
	}
	if a, err := user.AsUserAccount(); err == nil {
		v.HasAccount = true
		v.Meta = &UserMeta{CreatedAt: a.CreatedAt(), ModifiedAt: a.ModifiedAt()}
		v.Language = ptr(a.Language())
		v.AgeGroup = a.AgeGroup()
		v.AiQualityTier = a.AiQualityTier()
		v.DefaultApiKeyShareID = a.DefaultApiKeyShareID()
		if a.Role() != "" {
			v.Role = &UserRoleRef{
				Role:        a.Role(),
				Institution: namedRef(a.OrganisationID(), a.OrganisationName()),
				Workshop:    namedRef(a.WorkshopID(), a.WorkshopName()),
			}
		}
	}
	if c, err := user.AsUserContact(); err == nil {
		v.HasContact = true
		v.Email = c.Email()
	}
	return v
}

func namedRef(id *uuid.UUID, name *string) *NamedRef {
	if id == nil {
		return nil
	}
	ref := &NamedRef{ID: *id}
	if name != nil {
		ref.Name = *name
	}
	return ref
}

// UserSecretsView is how a user logs in without signing up: a guest's login
// code. Both fields are null for an account without one; words is null for
// the long codes issued before word codes.
type UserSecretsView struct {
	Token *string `json:"token"`
	Words *string `json:"words"`
}

// UserSecretsViewOf converts a user's secrets, refused (core.ErrLocked) when
// the caller may not see them.
//
//	return wire.UserSecretsViewOf(core.ReadUser(ctx, id))
func UserSecretsViewOf(user *core.UserCard, err error) (UserSecretsView, error) {
	if err != nil {
		return UserSecretsView{}, err
	}
	s, err := user.AsUserSecrets()
	if err != nil {
		return UserSecretsView{}, err
	}
	return UserSecretsView{Token: s.LoginToken(), Words: s.LoginWords()}, nil
}

// UserStatsView is what a user did on the platform, counted.
type UserStatsView struct {
	GamesPlayed       int `json:"gamesPlayed"`
	GamesCreated      int `json:"gamesCreated"`
	MessagesSent      int `json:"messagesSent"`
	TotalPlaysOnGames int `json:"totalPlaysOnGames"`
}

// UserStatsViewOf converts a user's statistics, passing a loading error
// through:
//
//	return wire.UserStatsViewOf(core.ReadUserStats(ctx, id))
func UserStatsViewOf(stats core.UserStats, err error) (UserStatsView, error) {
	if err != nil {
		return UserStatsView{}, err
	}
	return UserStatsView(stats), nil
}
