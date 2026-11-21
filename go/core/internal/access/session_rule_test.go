package access

import (
	"testing"

	"cgl/core/internal/rules"
)

// Only staff and heads change how often a session repeats the
// youth-protection rule.
func TestOnlyStaffAndHeadsTuneYouthProtection(t *testing.T) {
	for role, want := range map[rules.Role]bool{
		rules.Staff: true, rules.Head: true,
		rules.Admin: false, rules.Individual: false, rules.Guest: false,
	} {
		roles := []rules.Role{role, rules.SignedIn, rules.Everyone}
		if role == rules.Guest {
			roles = []rules.Role{role, rules.Everyone}
		}
		got := Relations(roles, []rules.Context{rules.Always, rules.InWorkshop, rules.OutsideWorkshop}, rules.Edit, rules.SessionRule)[rules.Own]
		if got != want {
			t.Errorf("%s may tune: %v, want %v", role, got, want)
		}
	}
}
