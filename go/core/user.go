package core

import (
	"context"
	"time"

	"cgl/core/internal/identity"
	"cgl/core/internal/store"
	"cgl/db"

	"github.com/google/uuid"
)

// A user is split into sections, because who may see them differs:
// UserCard, UserAccount, UserContact and UserSecrets. No field is in two
// sections. The card is the base type: whatever passes the gate is a card;
// the other sections stay locked until asked for.

// ListUsers lists the users whose card the caller may list: {List,
// UserCard}. One query; which grants open which of its fragments is in
// gates.go.
func ListUsers(ctx context.Context) ([]*UserCard, error) {
	rows, err := store.Queries().V2Users(ctx, listUserCardsGates(ctx))
	if err != nil {
		return nil, err
	}
	users := make([]*UserCard, len(rows))
	for i, r := range rows {
		users[i] = userCardOf(ctx, r)
	}
	return users, nil
}

// ReadUser opens one user's card, if the caller may read it: {Read,
// UserCard}. Otherwise the user does not exist for them (ErrNotFound).
func ReadUser(ctx context.Context, id uuid.UUID) (*UserCard, error) {
	rows, err := store.Queries().V2Users(ctx, readUserCardGates(ctx, id))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return userCardOf(ctx, rows[0]), nil
}

// The states of the caller's own account, as ReadMe reports them.
const (
	MeGuest       = "guest"       // nobody signed in
	MeRegistering = "registering" // signed in, registration not yet complete
	MeRegistered  = "registered"  // the account is complete
)

// ReadMe is the caller's own user: ReadUser with their id, and the state of
// their account. The user is nil while nobody is signed in or the account
// is still registering.
func ReadMe(ctx context.Context) (string, *UserCard, error) {
	c := from(ctx)
	switch {
	case c.Account() == nil:
		return MeGuest, nil, nil
	case c.Registering():
		return MeRegistering, nil, nil
	}
	user, err := ReadUser(ctx, *c.Account())
	if err != nil {
		return "", nil, err
	}
	return MeRegistered, user, nil
}

// userCardOf is a loaded row as the caller's card, with their rights on it.
func userCardOf(ctx context.Context, r db.V2UsersRow) *UserCard {
	return &UserCard{
		caller: from(ctx),
		row: userRow{
			id:   r.ID,
			name: r.Name.String,

			language:         r.Language,
			ageGroup:         nullString(r.AgeGroup),
			aiQualityTier:    nullString(r.AiQualityTier),
			defaultKeyShare:  ptrUUID(r.DefaultApiKeyShareID),
			role:             r.Role.String,
			organisationID:   ptrUUID(r.OrganisationID),
			organisationName: nullString(r.OrganisationName),
			workshopID:       ptrUUID(r.WorkshopID),
			workshopName:     nullString(r.WorkshopName),
			createdAt:        r.CreatedAt,
			modifiedAt:       r.ModifiedAt,

			email: nullString(r.Email),

			loginToken: nullString(r.GuestToken),
		},
		mayReadAccount: r.MayReadAccount,
		mayReadContact: r.MayReadContact,
		mayReadSecrets: r.MayReadSecrets,
		mayEditCard:    r.MayEditCard,
		mayEditAccount: r.MayEditAccount,
		mayEditContact: r.MayEditContact,
		mayDelete:      r.MayDelete,
	}
}

// userRow is a user's full row, loaded once. It never leaves the core; each
// section's type offers only its own fields.
type userRow struct {
	// card
	id   uuid.UUID
	name string

	// account
	language         string
	ageGroup         *string
	aiQualityTier    *string
	defaultKeyShare  *uuid.UUID
	role             string
	organisationID   *uuid.UUID
	organisationName *string
	workshopID       *uuid.UUID
	workshopName     *string
	createdAt        time.Time
	modifiedAt       time.Time

	// contact
	email *string

	// secrets
	loginToken *string
}

// UserCard is who a user is: what anyone who may see the user gets. It holds
// the full row, the caller it was loaded for and that caller's rights on
// it, and shows only the card.
type UserCard struct {
	caller *identity.Caller
	row    userRow

	mayReadAccount bool
	mayReadContact bool
	mayReadSecrets bool
	mayEditCard    bool
	mayEditAccount bool
	mayEditContact bool
	mayDelete      bool
}

func (u *UserCard) ID() uuid.UUID { return u.row.id }

// Name is the user's public profile name; empty while they register.
func (u *UserCard) Name() string { return u.row.name }

// MayEditCard says whether the caller may change the user's name:
// {Edit, UserCard}.
func (u *UserCard) MayEditCard() bool { return u.mayEditCard }

// MayEditAccount says whether the caller may change how the user's account
// is set up: {Edit, UserAccount}.
func (u *UserCard) MayEditAccount() bool { return u.mayEditAccount }

// MayEditContact says whether the caller may change the user's email:
// {Edit, UserContact}.
func (u *UserCard) MayEditContact() bool { return u.mayEditContact }

// MayDelete says whether the caller may delete the user: {Delete,
// UserCard}. An organisation's last head is refused even so.
func (u *UserCard) MayDelete() bool { return u.mayDelete }

// AsUserAccount unlocks how the account is set up: a flag check, no query.
func (u *UserCard) AsUserAccount() (*UserAccount, error) {
	if !u.mayReadAccount {
		return nil, ErrLocked
	}
	return &UserAccount{card: u}, nil
}

// AsUserContact unlocks how to reach the user: a flag check, no query.
func (u *UserCard) AsUserContact() (*UserContact, error) {
	if !u.mayReadContact {
		return nil, ErrLocked
	}
	return &UserContact{card: u}, nil
}

// AsUserSecrets unlocks how the user logs in without signing up: a flag
// check, no query.
func (u *UserCard) AsUserSecrets() (*UserSecrets, error) {
	if !u.mayReadSecrets {
		return nil, ErrLocked
	}
	return &UserSecrets{card: u}, nil
}

// UserAccount is how a user's account is set up.
type UserAccount struct{ card *UserCard }

func (a *UserAccount) Language() string { return a.card.row.language }

// AgeGroup is "u13" (13-17, no parental consent), "u13p" (13-17, with
// consent), "u18" (18+), or nil when unknown.
func (a *UserAccount) AgeGroup() *string { return a.card.row.ageGroup }

// AiQualityTier is high, medium or low; nil for the server's default.
func (a *UserAccount) AiQualityTier() *string           { return a.card.row.aiQualityTier }
func (a *UserAccount) DefaultApiKeyShareID() *uuid.UUID { return a.card.row.defaultKeyShare }

// Role is the user's role as the database stores it; empty for none.
func (a *UserAccount) Role() string { return a.card.row.role }

// Organisation is the user's organisation: their role's, or their
// workshop's. Nil for none.
func (a *UserAccount) OrganisationID() *uuid.UUID { return a.card.row.organisationID }
func (a *UserAccount) OrganisationName() *string  { return a.card.row.organisationName }

// Workshop is the one the user is in: a guest's own, or the one entered in
// workshop mode. Nil for none.
func (a *UserAccount) WorkshopID() *uuid.UUID { return a.card.row.workshopID }
func (a *UserAccount) WorkshopName() *string  { return a.card.row.workshopName }

func (a *UserAccount) CreatedAt() time.Time  { return a.card.row.createdAt }
func (a *UserAccount) ModifiedAt() time.Time { return a.card.row.modifiedAt }

// UserContact is how to reach a user.
type UserContact struct{ card *UserCard }

// Email is nil for none; a guest has none.
func (c *UserContact) Email() *string { return c.card.row.email }

// UserSecrets is how a user logs in without signing up.
type UserSecrets struct{ card *UserCard }

// LoginToken is a guest's whole login code, the one their link and QR code
// carry; nil for an account without one.
func (s *UserSecrets) LoginToken() *string { return s.card.row.loginToken }

// LoginWords is the login code as words to type; nil for none, and for the
// long codes issued before word codes.
func (s *UserSecrets) LoginWords() *string {
	if s.card.row.loginToken == nil {
		return nil
	}
	words := db.GuestTokenWords(*s.card.row.loginToken)
	if words == "" {
		return nil
	}
	return &words
}

// ReadUserStats counts what a user did: their account's statistics. A user
// the caller may not read does not exist for them (ErrNotFound); one whose
// account they may not see is refused (ErrLocked).
func ReadUserStats(ctx context.Context, id uuid.UUID) (UserStats, error) {
	user, err := ReadUser(ctx, id)
	if err != nil {
		return UserStats{}, err
	}
	account, err := user.AsUserAccount()
	if err != nil {
		return UserStats{}, err
	}
	return account.Stats(ctx)
}

// UserStats is what a user did on the platform, counted.
type UserStats struct {
	GamesPlayed       int // sessions they played
	GamesCreated      int
	MessagesSent      int
	TotalPlaysOnGames int // plays of the games they created
}

// Stats counts what the user did. It belongs to the account: whoever may see
// the account may see it. Loaded on demand, four queries.
func (a *UserAccount) Stats(ctx context.Context) (UserStats, error) {
	id := a.card.row.id
	q := store.Queries()
	created, err := q.CountUserGames(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return UserStats{}, err
	}
	plays, err := q.SumPlayCountOfUserGames(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return UserStats{}, err
	}
	played, err := q.CountUserGameSessions(ctx, id)
	if err != nil {
		return UserStats{}, err
	}
	messages, err := q.SumUserGameSessionMessages(ctx, id)
	if err != nil {
		return UserStats{}, err
	}
	return UserStats{
		GamesPlayed:       int(played),
		GamesCreated:      int(created),
		MessagesSent:      int(messages),
		TotalPlaysOnGames: int(plays),
	}, nil
}
