// Package access turns the grant table into what a query needs: for a caller's
// roles, the relations through which they may do an action on an entity. Each
// relation becomes one boolean parameter of the entity's query. DRAFT.
package access

import "cgl/core/internal/rules"

// Relations are the relations through which the given roles may do action on
// entity in the given contexts: every grant of those roles, in one of those
// contexts, that names both.
func Relations(roles []rules.Role, contexts []rules.Context, action rules.Action, entity rules.Entity) map[rules.Relation]bool {
	granted := map[rules.Relation]bool{}
	for _, role := range roles {
		for _, g := range rules.Rules[role].For(contexts) {
			if g.Action == action && g.Entity == entity {
				granted[g.Relation] = true
			}
		}
	}
	return granted
}
