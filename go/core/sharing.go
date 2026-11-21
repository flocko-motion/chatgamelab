package core

// DRAFT: the sharing model of #451, scaffolded like the rules. The grant table
// (rules.go) says who may share; this file says where a share may go and what
// always holds for the chain. The terms of a share (tiers, features, passing
// on) are left open.

// Level is where a share sits in its chain. The workshop line runs between
// LevelColleague and LevelWorkshop: above it keys are organised, from the
// workshop downward they are only spent.
type Level string

const (
	LevelKey          Level = "key"          // the key itself; its owner is the source
	LevelOrganisation Level = "organisation" // given to the owner's organisation
	LevelColleague    Level = "colleague"    // given to a colleague in the organisation
	LevelWorkshop     Level = "workshop"     // bound to a workshop: its budget
	LevelLink         Level = "link"         // a share link: one game, whoever holds it
)

// Move says that a share at one level may mint a derived share at another.
type Move struct {
	From Level
	To   Level
}

// Minting is where a share may go: sharing always mints a derived share, and
// a move that is not listed here is refused. Every move stays inside the
// owner's organisation: never across organisations, never to individuals.
var Minting = []Move{
	{LevelKey, LevelOrganisation},
	{LevelKey, LevelColleague},
	{LevelKey, LevelWorkshop},
	{LevelKey, LevelLink},

	{LevelOrganisation, LevelColleague}, // open: may a head hand the organisation's key to a teacher?
	{LevelOrganisation, LevelWorkshop},

	{LevelColleague, LevelWorkshop},

	{LevelWorkshop, LevelLink}, // below the workshop line: only downward, only links
}

// Money is an amount in the smallest currency unit. Budgets count money, not
// tokens: tokens cost different amounts per provider and model.
type Money int64

// KeyShare is a derived share: its own row, with a reference to its source,
// so the chain is stored, not reconstructed.
type KeyShare struct {
	id     string
	source *KeyShare // nil: minted straight from the key
	level  Level
	holder string // organisation, colleague, workshop or game, by level
	budget *Money // nil: no budget of its own, the source's applies
	spent  Money
	// terms: open (tiers, features, a default, whether players may choose,
	// whether it may be passed on)
}

// KeySharer is the capability to pass a share on: the caller holds Share on
// it (rules.go) and, once terms exist, the share permits it. The double
// condition: the person's allowance and the share's permission.
type KeySharer struct {
	share *KeyShare
}

// Mint derives a share from the sharer's share. Pseudo code:
//
//	refuse unless Move{s.level, to} is listed in Minting
//	refuse unless the holder is inside the key owner's organisation
//	refuse unless budget <= remaining(s.share)          // attenuation
//	refuse unless terms ⊆ terms(s.share)                // open
//	insert KeyShare{source: s.share, level: to, holder, budget}
func (s *KeySharer) Mint(to Level, holder string, budget *Money) (*KeyShare, error) {
	panic("draft")
}

// Revoke ends a share and everything derived from it, share links included.
// Anyone above a share in its chain may revoke it; nothing flows upward: a
// receiver cannot revoke, change or widen the share they were given.
//
//	refuse unless s.share is an ancestor of target       // revoking flows downward
//	delete target and every share whose chain passes through it
func (s *KeySharer) Revoke(target *KeyShare) error {
	panic("draft")
}

// Spend charges a session's cost to a share and every share above it: a
// level's budget is always part of the budget above it.
//
//	refuse if the model has no price                    // fail closed: unknown is not free
//	for share := range chain(from, up to the key):
//	    refuse if share.budget != nil && share.spent+cost > *share.budget
//	for share := range chain(...): share.spent += cost
func Spend(from *KeyShare, cost Money) error {
	panic("draft")
}

// Invariants of the chain, held for everyone, admins included:
//
//   - sharing always mints; there are no pointers to another holder's share
//   - a share never leaves its holder: a workshop's share is not re-bound
//   - a derived share never gives more than its source (budget, terms)
//   - money moves only by the decision of whoever it belongs to, or of
//     someone they shared it with
