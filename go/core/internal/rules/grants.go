// Package rules is the secure core's access rules: who may do what. It holds
// only vocabulary and the grant table, no code. DRAFT.
package rules

// Role is what a caller is in the system. A role is nothing but a list of
// grants (see rules.go).
type Role string

const (
	Admin      Role = "admin"
	Head       Role = "head"
	Staff      Role = "staff"
	Individual Role = "individual"

	// Guest plays a shared game through a share link, or takes part in a
	// workshop through its invite link under an animal name. A guest through
	// a share link never signs in and is never in a workshop; a guest in a
	// workshop signs in (so SignedIn holds too) and is always in its
	// workshop: no grant lets a guest enter or leave one. A caller nobody
	// signed in is a guest too.
	Guest Role = "guest"

	// Everyone is every caller, signed in or not, on top of their role.
	Everyone Role = "everyone"

	// SignedIn is every caller with an account that completed registration,
	// on top of Everyone. While registering, an account holds only what
	// Everyone may in the Registering context.
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
	// Edit changes an entity. Deleting it is Delete: whoever may change an
	// entity may not always remove it (everyone edits their own account,
	// nobody deletes it).
	Edit   Action = "edit"
	Delete Action = "delete"
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

	// Suspend stops an entity's use, and lifts that again; its data stays.
	// A suspended organisation's members are refused (see Suspended).
	Suspend Action = "suspend"
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

	// UserCard is who a user is: their id and public profile name, the name
	// games show as their creator's.
	UserCard Entity = "user card"
	// UserAccount is how a user's account is set up: language, age group,
	// AI quality tier, default key, role (with its organisation and
	// workshop), timestamps and statistics.
	UserAccount Entity = "user account"
	// UserContact is how to reach a user: their email. Seen like the account,
	// changed by its owner; a guest has none.
	UserContact Entity = "user contact"
	// UserSecrets is how a user logs in without signing up: a guest's login
	// code. Whoever holds it is that guest.
	UserSecrets Entity = "user secrets"

	Workshop Entity = "workshop"
	// Organisation is an organisation: its name, operator, address and
	// youth-protection restriction, and its consent to the declaration, which
	// a head gives or withdraws by editing it.
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
	OfOwnOrganisation Relation = "of own organisation" // belongs to the caller's organisation, e.g. a game of one of its workshops, a member, a guest in one of its workshops

	MemberOfOwnOrganisation Relation = "member of own organisation" // a user holding a role in the caller's organisation (individual, staff, head)
	GuestOfOwnOrganisation  Relation = "guest of own organisation"  // a guest in one of the caller's organisation's workshops
	OfOwnGame               Relation = "of own game"                // belongs to a game the caller created
	PaidByOwnKey            Relation = "paid by own key"            // spends from an API key the caller owns

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
	Always          Context = "always"
	InWorkshop      Context = "in a workshop"      // the caller is in a workshop
	OutsideWorkshop Context = "outside a workshop" // the caller is in none
	PublicShown     Context = "public shown"       // the caller's workshop shows public games
	OthersShown     Context = "others shown"       // the caller's workshop shows the games of the others in it
	SharingAllowed  Context = "sharing allowed"    // the caller's workshop lets those in it share games through its key
	Registering     Context = "registering"        // the caller signed in, but has not completed registration
	// Suspended: the caller's organisation is suspended, by an admin or for
	// want of a consent to the current declaration. It is the caller's only
	// context, so of all their grants only Suspended ones hold.
	Suspended Context = "organisation suspended"
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
