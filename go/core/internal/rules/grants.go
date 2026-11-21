// Package rules is the secure core's access rules: who may do what. It holds
// only vocabulary and the grant table, no code. DRAFT.
package rules

// Role is what a caller is in the system. A role is nothing but a list of
// grants (see rules.go).
type Role string

const (
	Admin       Role = "admin"
	Head        Role = "head"
	Staff       Role = "staff"
	Participant Role = "participant"
	Individual  Role = "individual"

	// Everyone is every caller, signed in or not, on top of their role. It is
	// the baseline: a guest, who plays without an account, has exactly this.
	Everyone Role = "everyone"

	// SignedIn is every caller with an account, on top of Everyone.
	SignedIn Role = "signed in"
)

// Action is what is done with an entity.
//
// Read opens one entity whose address the caller has; List finds entities.
// Whoever may list something may also read it, not the other way round.
// Listing a secret, such as share links, hands out the secrets themselves.
type Action string

const (
	Read Action = "read"
	List Action = "list"
	Play Action = "play"
	// Edit changes an entity, deleting it included.
	Edit   Action = "edit"
	Manage Action = "manage"
	// Create makes a new entity. It is the caller's own from the start, so a
	// create grant's relation is Own, and the new entity's rights are the
	// role's Own grants.
	Create Action = "create"
	// Copy makes a new entity of the caller's from this one.
	Copy Action = "copy"
	// Publish makes an entity public: from then on, everyone finds it.
	// Making it private again only narrows who sees it, so it is an Edit.
	Publish Action = "publish"

	// Use spends from a key for the caller's own plays; Share passes it on
	// (a key share, a share link, a workshop's key, sponsoring a game).
	Use   Action = "use"
	Share Action = "share"

	// Enter puts the caller into a workshop: workshop mode.
	Enter Action = "enter"
)

// Entity is a kind of thing in the system, named in the singular. Where
// sections of a thing differ in who may see them, the thing is split and each
// section is an entity of its own (GameCard, GameContent, GameSettings); the
// whole thing is then no entity. Otherwise a thing is one entity.
type Entity string

const (
	// GameCard is a game's card: name, description, image, genre, tags, play
	// count. It stands for the game in lists and is what is played; it
	// reveals neither content nor settings.
	GameCard Entity = "game card"
	// GameContent is how a game works: prompts, style, status fields, first
	// scene. Open in the spirit of ChatGameLab.
	GameContent Entity = "game content"
	// GameSettings is who is behind a game: its workshop and organisation,
	// the key that pays, its creator.
	GameSettings Entity = "game settings"

	Session Entity = "session"
	// ShareLink is a voucher: someone's money, for plays of one game, for
	// whoever holds the link. It belongs to neither the game nor the key; the
	// payer is the weightier side, so whoever pays for it may see it.
	ShareLink Entity = "share link"
	// ApiKey is a key at an AI provider. Its secret is write-only: no grant
	// reads it, not even the owner's.
	ApiKey Entity = "api key"
	// ApiKeyShare is the right to spend from a key, given to a user, a
	// workshop or an organisation. Its terms (tiers, features, passing on)
	// are data on the share, not rules here.
	ApiKeyShare Entity = "api key share"

	Workshop     Entity = "workshop"
	Organisation Entity = "organisation"
)

// Relation says which entities a grant covers, seen from the caller. A
// relation means the same for every entity it is used with. The caller's
// situation, such as being in a workshop or what it shows, is the grant's
// context (Grants), never part of a relation.
type Relation string

const (
	Everywhere Relation = "everywhere"
	Public     Relation = "public"
	Active     Relation = "active"  // switched on, e.g. a workshop that is running
	Current    Relation = "current" // the one the caller is in, e.g. their current workshop
	Held       Relation = "held"    // held by the caller, directly or through their organisation or workshop
	Own        Relation = "own"     // created by the caller

	Assigned          Relation = "assigned"            // the caller is assigned to it
	OfCurrentWorkshop Relation = "of current workshop" // the workshop the caller is in
	OfOwnOrganisation Relation = "of own organisation" // belongs to the caller's organisation, e.g. a game of one of its workshops
	OfOwnGame         Relation = "of own game"         // belongs to a game the caller created
	PaidByOwnKey      Relation = "paid by own key"     // spends from an API key the caller owns

	OwnOfCurrentWorkshop Relation = "own, of the current workshop" // created by the caller, in the workshop they are in
	PublicElsewhere      Relation = "public, elsewhere"            // public, not of the workshop the caller is in

	SharedWithMe              Relation = "shared with me"               // given to the caller
	SharedWithCurrentWorkshop Relation = "shared with current workshop" // given to the workshop the caller is in
	SharedWithOwnOrganisation Relation = "shared with own organisation" // given to the caller's organisation

	// Via: the caller holds the secret that opens this one entity. Like a
	// wallet key, it grants access but can never be listed.
	ViaShareLink      Relation = "via share link"
	ViaGuestSessionID Relation = "via guest session id"
)

// Context is a situation the caller is in. A grant holds in one context; a
// caller can be in several at once, and holds the grants of all of them.
type Context string

const (
	Always            Context = "always"
	InWorkshop        Context = "in a workshop"      // the caller is in a workshop
	OutsideWorkshop   Context = "outside a workshop" // the caller is in none
	PublicShown       Context = "public shown"       // the caller's workshop shows public games
	ParticipantsShown Context = "participants shown" // the caller's workshop shows participants' games
	SharingAllowed    Context = "sharing allowed"    // the caller's workshop lets participants share games
)

// Grants are a role's grants by context.
type Grants map[Context][]Grant

// For are the grants that hold in the given contexts.
func (g Grants) For(contexts []Context) []Grant {
	var grants []Grant
	for _, c := range contexts {
		grants = append(grants, g[c]...)
	}
	return grants
}

// Grant gives a role an action on the entities of a relation: one line of
// the rules.
type Grant struct {
	Action   Action
	Entity   Entity
	Relation Relation
}
