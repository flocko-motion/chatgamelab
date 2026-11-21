package session

import (
	"testing"

	"cgl/core"
	"cgl/obj"
)

func str(s string) *string { return &s }

// The cascades follow the spec's two rule sets (Jugendschutz, "Which rules
// apply"): who plays decides, the first source that sets a rule wins.
func TestResolveRule(t *testing.T) {
	adult := str(obj.AgeGroupU18)
	site := map[string]*string{obj.AgeGroupU13: str("u13"), obj.AgeGroupU13p: str("u13p"), obj.AgeGroupU18: str("u18")}
	cases := []struct {
		name   string
		facts  core.PlayRuleFacts
		want   string
		source string
	}{
		{"a known player without workshop: the site's for their age",
			core.PlayRuleFacts{Player: &core.PlayerRules{AgeGroup: adult}, Site: site}, "u18", obj.ConstraintSourceSite18},
		{"an unknown age: the strictest",
			core.PlayRuleFacts{Player: &core.PlayerRules{}, Site: site}, "u13", obj.ConstraintSourceSite13},
		{"their workshop before their organisation",
			core.PlayRuleFacts{Player: &core.PlayerRules{
				Workshop:     &core.RuleSource{Name: "W", Rule: str(" ws ")},
				Organisation: &core.RuleSource{Name: "O", Rule: str("org")}}, Site: site}, "ws", obj.ConstraintSourceWorkshop},
		{"an empty workshop rule falls through",
			core.PlayRuleFacts{Player: &core.PlayerRules{
				Workshop:     &core.RuleSource{Name: "W", Rule: str("  ")},
				Organisation: &core.RuleSource{Name: "O", Rule: str("org")}}, Site: site}, "org", obj.ConstraintSourceOrganisation},
		{"the link's workshop before the player's own",
			core.PlayRuleFacts{Player: &core.PlayerRules{Workshop: &core.RuleSource{Name: "W", Rule: str("ws")}},
				LinkWorkshop: &core.RuleSource{Name: "L", Rule: str("link")}, Site: site}, "link", obj.ConstraintSourceWorkshop},
		{"a guest: the link's organisation",
			core.PlayRuleFacts{LinkGuest: true, LinkOrganisation: &core.RuleSource{Name: "O", Rule: str("org")}, Site: site}, "org", obj.ConstraintSourceOrganisation},
		{"a guest, no link rule: the author's own, without the author's age skipped",
			core.PlayRuleFacts{LinkGuest: true, LinkWorkshop: &core.RuleSource{Name: "L"},
				Author: &core.PlayerRules{AgeGroup: adult}, Site: site}, "u18", obj.ConstraintSourceSite18},
		{"a guest with no link at all: the strictest",
			core.PlayRuleFacts{LinkGuest: true, Site: site}, "u13", obj.ConstraintSourceSite13},
	}
	for _, c := range cases {
		r := resolveRule(c.facts)
		if r.Text == nil || *r.Text != c.want || r.Source != c.source {
			t.Errorf("%s: got %v %q, want %q %q (%s)", c.name, r.Text, r.Source, c.want, c.source, r.Reasoning)
		}
	}
}
