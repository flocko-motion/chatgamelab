// package: core/internal/rules / the access vocabulary
// type:    data
// job:     the roles, actions, entities, relations and contexts the grant table speaks
// limits:  vocabulary only, no code; the table itself (-> rules.go)
package rules

// Role is what a caller is: nothing but a list of grants (rules.go).
type Role string

// The role says where someone is (spec, Roles): head and staff in an
// organisation, an individual in none, again on leaving it.
const (
	// Admin runs the site.
	Admin Role = "admin"
	// Head leads an organisation; Staff runs its workshops.
	Head  Role = "head"
	Staff Role = "staff"
	// Individual uses the platform in no organisation and no workshop.
	Individual Role = "individual"

	// Guest plays through a share link, or is fixed in one workshop (signed in,
	// never entering or leaving one); a caller nobody signed in is one too.
	Guest Role = "guest"

	// Everyone is every caller, signed in or not, on top of their role.
	Everyone Role = "everyone"

	// SignedIn is every registered account, on top of Everyone; one still
	// registering holds only Everyone's Registering grants.
	SignedIn Role = "signed in"
)

// Action is what is done with an entity. Read opens one by its address; List
// finds them, and listing a secret hands it out.
type Action string

const (
	Read Action = "read"
	List Action = "list"
	Play Action = "play"
	// Edit changes an entity; Delete removes it, a right of its own (everyone
	// edits their account, nobody deletes it).
	Edit   Action = "edit"
	Delete Action = "delete"
	Manage Action = "manage"
	// Create makes an entity: the caller's (Own), their organisation's
	// (OfOwnOrganisation), or for others (Everywhere).
	Create Action = "create"
	// Copy makes a new entity of the caller's from this one.
	Copy Action = "copy"
	// Publish makes an entity public; making it private again is an Edit.
	Publish Action = "publish"

	// Use spends from a key for the caller's own plays; Share passes it on
	// (a key share, a share link, a workshop's key, sponsoring a game).
	Use   Action = "use"
	Share Action = "share"

	// Enter and Leave move the caller into a workshop and out; a guest never
	// leaves, as their workshop is their place.
	Enter Action = "enter"
	Leave Action = "leave"

	// Suspend stops an entity's use and lifts it again; its data stays.
	Suspend Action = "suspend"
)

// Entity is a kind of thing; one whose sections differ in who sees them is
// split, each section an entity (GameCard, GameContent, …).
type Entity string

const (
	// GameCard is a game in lists and what is played: name, description,
	// image, genre, tags, plays; no content or settings.
	GameCard Entity = "game card"
	// GameContent is how a game works: prompts, style, status, first scene.
	GameContent Entity = "game content"
	// GameSettings is who is behind a game: workshop, organisation, payer,
	// creator.
	GameSettings Entity = "game settings"

	// Session is a game being played: its story, state and insights.
	Session Entity = "session"
	// SessionRule is how often each block of a session repeats its rule.
	SessionRule Entity = "session rule"
	// ShareLink is a voucher: someone's money for plays of one game; whoever
	// pays for it may see it.
	ShareLink Entity = "share link"
	// ApiKey is a key at an AI provider; no grant reads its secret.
	ApiKey Entity = "api key"
	// ApiKeyShare is the right to spend from a key; its terms are data on the
	// share, not rules.
	ApiKeyShare Entity = "api key share"

	// UserCard is who a user is: id and the name games show as creator.
	UserCard Entity = "user card"
	// UserAccount is how an account is set up: language, age group, tier,
	// default key, role, timestamps, statistics.
	UserAccount Entity = "user account"
	// UserContact is a user's email; a guest has none.
	UserContact Entity = "user contact"
	// UserSecrets is a guest's login code: whoever holds it is that guest.
	UserSecrets Entity = "user secrets"

	// WorkshopCard is a workshop to those in it: name, organisation, running,
	// what it shows and allows, tier, public page.
	WorkshopCard Entity = "workshop card"
	// WorkshopSettings is a workshop's rule, key share and public page text.
	WorkshopSettings Entity = "workshop settings"
	// WorkshopParticipants is who is in a workshop; managing it takes someone
	// out of it.
	WorkshopParticipants Entity = "workshop participants"
	// Invite brings someone into an organisation or a workshop; using it is
	// answering it, listing it hands out its link.
	Invite Entity = "invite"
	// HeadInvite invites a head, apart as fewer may make, see or revoke it.
	HeadInvite Entity = "head invite"

	// OrganisationCard is an organisation's id and name.
	OrganisationCard Entity = "organisation card"
	// OrganisationSettings is an organisation's rule and free-use key.
	OrganisationSettings Entity = "organisation settings"
	// OrganisationDetails is its Träger and address; consenting to the
	// declaration is a head's edit of it.
	OrganisationDetails Entity = "organisation details"
	// OrganisationRecord is its standing: consent, registration, suspension.
	OrganisationRecord Entity = "organisation record"
	// VerifiedPage is an organisation's one page that frames its embedded
	// games once approved; managing it is approving it.
	VerifiedPage Entity = "verified page"
	// Membership is a user's role in an organisation; deleting it leaves them
	// an individual.
	Membership Entity = "membership"
	// Site is the running platform itself, such as restarting the server.
	Site Entity = "site"
	// SiteSettings are the site's settings: its rules, tiers and free-use key.
	SiteSettings Entity = "site settings"
	// SiteRules are the site's youth-protection rules per age group.
	SiteRules Entity = "site rules"
)

// Relation says which entities a grant covers, seen from the caller; the
// caller's situation is a context, never a relation.
type Relation string

const (
	Everywhere Relation = "everywhere"
	Public     Relation = "public"
	Active     Relation = "active"  // switched on, e.g. a workshop that is running
	Current    Relation = "current" // the one the caller is in, e.g. their current workshop
	Own        Relation = "own"     // created by the caller

	OfCurrentWorkshop Relation = "of current workshop" // the workshop the caller is in
	OfOwnOrganisation Relation = "of own organisation" // belongs to the caller's organisation, e.g. a member, a guest in one of its workshops, a game of one of its workshops or by its head or staff
	// OfOwnOrganisationsWorkshop is narrower for games: made in one of the
	// caller's organisation's workshops, not only by its head or staff.
	OfOwnOrganisationsWorkshop Relation = "of own organisation's workshop"

	MemberOfOwnOrganisation Relation = "member of own organisation" // head or staff of the caller's organisation; an individual is in none
	GuestOfOwnOrganisation  Relation = "guest of own organisation"  // a guest in one of the caller's organisation's workshops
	OfOwnGame               Relation = "of own game"                // belongs to a game the caller created
	PaidByOwnKey            Relation = "paid by own key"            // spends from an API key the caller owns
	PaidByOwnOrganisation   Relation = "paid by own organisation"   // spends from a key given to the caller's organisation

	OwnOfCurrentWorkshop Relation = "own, of the current workshop" // created by the caller, in the workshop they are in
	PublicElsewhere      Relation = "public, elsewhere"            // public, not of the workshop the caller is in

	SharedWithMe              Relation = "shared with me"               // given to the caller
	SharedWithCurrentWorkshop Relation = "shared with current workshop" // given to the workshop the caller is in
	SharedWithOwnOrganisation Relation = "shared with own organisation" // given to the caller's organisation
	FreeUseOfOwnOrganisation  Relation = "free use of own organisation" // the key the caller's organisation lets its members play with

	// Via: the caller holds the secret that opens this one entity, never
	// listable.
	ViaShareLink      Relation = "via share link"
	ViaInviteLink     Relation = "via invite link" // an invite link: whoever has it may follow it
	ViaInvite         Relation = "via invite"      // through an invite to it the caller accepts, a link or one addressed to them
	ViaGuestSessionID Relation = "via guest session id"
)

// Context is a situation the caller is in; they hold the grants of every
// context they are in.
type Context string

const (
	Always          Context = "always"
	InWorkshop      Context = "in a workshop"      // the caller is in a workshop
	OutsideWorkshop Context = "outside a workshop" // the caller is in none
	PublicShown     Context = "public shown"       // the caller's workshop shows public games
	OthersShown     Context = "others shown"       // the caller's workshop shows the games of the others in it
	SharingAllowed  Context = "sharing allowed"    // the caller's workshop lets those in it share games through its key
	Registering     Context = "registering"        // the caller signed in, but has not completed registration
	// Suspended: the caller's organisation is suspended; their only context,
	// so only these grants hold.
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
