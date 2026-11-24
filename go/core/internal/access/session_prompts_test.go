package access

import (
	"testing"

	"cgl/core/internal/rules"
)

// Admins, heads and staff adapt the prompts of a session they start.
func TestAdminsHeadsAndStaffPresetPrompts(t *testing.T) {
	for role, want := range map[rules.Role]bool{
		rules.Staff: true, rules.Head: true, rules.Admin: true,
		rules.Individual: false, rules.Guest: false,
	} {
		roles := []rules.Role{role, rules.SignedIn, rules.Everyone}
		if role == rules.Guest {
			roles = []rules.Role{role, rules.Everyone}
		}
		got := Relations(roles, []rules.Context{rules.Always, rules.InWorkshop, rules.OutsideWorkshop}, rules.Edit, rules.SessionPrompts)[rules.Own]
		if got != want {
			t.Errorf("%s may preset prompts: %v, want %v", role, got, want)
		}
	}
}
