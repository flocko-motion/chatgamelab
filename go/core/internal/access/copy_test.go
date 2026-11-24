package access

import (
	"testing"

	"cgl/core/internal/rules"
)

// Copy is reading a game's content and creating a game, nothing more: for
// every role and situation, the copy grant holds exactly through the
// relations reading the content does, and only where creating a game does.
func TestCopyIsReadingAndCreating(t *testing.T) {
	inWorkshop := []rules.Context{rules.Always, rules.InWorkshop}
	situations := map[string][]rules.Context{
		"outside a workshop":        {rules.Always, rules.OutsideWorkshop},
		"in a workshop":             inWorkshop,
		"in a workshop showing all": append(inWorkshop, rules.PublicShown, rules.OthersShown, rules.SharingAllowed),
		"organisation suspended":    {rules.Suspended},
	}
	callers := map[string][]rules.Role{
		"admin":                  {rules.Everyone, rules.Admin, rules.SignedIn},
		"head":                   {rules.Everyone, rules.Head, rules.SignedIn},
		"staff":                  {rules.Everyone, rules.Staff, rules.SignedIn},
		"individual":             {rules.Everyone, rules.Individual, rules.SignedIn},
		"guest signed in":        {rules.Everyone, rules.Guest, rules.SignedIn},
		"guest nobody signed in": {rules.Everyone, rules.Guest},
		"registering":            {rules.Everyone, rules.Individual},
	}
	for name, roles := range callers {
		for situation, contexts := range situations {
			if name == "registering" {
				contexts = []rules.Context{rules.Always, rules.OutsideWorkshop, rules.Registering}
			}
			creates := len(Relations(roles, contexts, rules.Create, rules.GameCard)) > 0
			reads := Relations(roles, contexts, rules.Read, rules.GameContent)
			copies := Relations(roles, contexts, rules.Copy, rules.GameContent)
			for relation := range reads {
				if creates && !copies[relation] {
					t.Errorf("%s, %s: reads content %q and creates games, but may not copy it", name, situation, relation)
				}
			}
			for relation := range copies {
				if !creates || !reads[relation] {
					t.Errorf("%s, %s: copies %q, but does not both read its content and create games", name, situation, relation)
				}
			}
		}
	}
}
