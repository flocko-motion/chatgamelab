// package: session / engine sessions from the platform's data
// type:    logic
// job:     resolves which youth-protection rule (Jugendschutz) applies to a play, with a reasoning trace.
// limits:  the facts come from the core (core.ReadPlayRuleFacts); the rule is enforced in generation, not here.
package session

import (
	"strings"

	"cgl/core"
	"cgl/obj"
)

// This file is the single, canonical home for the youth-protection rule
// cascade. It is the code counterpart of the spec (docs/chatgamelab.typ),
// Jugendschutz, "Which rules apply". Keep the two in lockstep: resolveRule is
// the only entry point, and the two cascades map one to one onto the spec's
// two rule sets.

// Rule is the youth-protection rule that applies to a play. Reasoning is a
// readable trace of the decision, so the insights view can explain why it
// was chosen.
type Rule struct {
	Text       *string // the rule (nil only if a site default is unset)
	Source     string  // obj.ConstraintSource*
	SourceName string  // the workshop's or organisation's name; empty for the site's
	Reasoning  string
}

// resolveRule decides which rule applies to a play. The discriminator is who
// plays, never how they arrived (spec, Jugendschutz):
//
//   - a guest who came through a share link (not known): the "Gäste"
//     cascade: the link's workshop, the link's organisation, the link
//     author's own rule;
//   - a known player: the "Angemeldete Nutzer" cascade: the link's workshop,
//     the link's organisation, the current workshop (the one they take part
//     in), the organisation (their own, else the current workshop's), the
//     site's rule for their age group.
func resolveRule(f core.PlayRuleFacts) Rule {
	if !f.LinkGuest && f.Player != nil {
		r := knownPlayerRule(f, f.Player, true)
		r.Reasoning = "Player is a signed-in user (" + ageGroupLabel(f.Player.AgeGroup) + "); " + r.Reasoning
		return r
	}
	if f.LinkWorkshop != nil || f.LinkOrganisation != nil || f.Author != nil {
		r := guestRule(f)
		r.Reasoning = "Player is a guest playing through a share link; " + r.Reasoning
		return r
	}
	// a guest with no link should not occur; the strictest site rule is safe
	text, source := siteRule(f, nil)
	return Rule{Text: text, Source: source,
		Reasoning: "No player and no share link; applying the strictest site default (" + ageGroupLabel(nil) + ")."}
}

// knownPlayerRule is the "Angemeldete Nutzer" cascade; withLink feeds the
// link's workshop and organisation as its first stages.
func knownPlayerRule(f core.PlayRuleFacts, p *core.PlayerRules, withLink bool) Rule {
	var skipped []string
	if withLink {
		if r, ok := fromSource(f.LinkWorkshop, obj.ConstraintSourceWorkshop, "the share's workshop", &skipped); ok {
			return r
		}
		if r, ok := fromSource(f.LinkOrganisation, obj.ConstraintSourceOrganisation, "the share's organisation", &skipped); ok {
			return r
		}
	}
	if r, ok := fromSource(p.Workshop, obj.ConstraintSourceWorkshop, "the current workshop", &skipped); ok {
		return r
	}
	if r, ok := fromSource(p.Organisation, obj.ConstraintSourceOrganisation, "the player's organisation", &skipped); ok {
		return r
	}
	text, source := siteRule(f, p.AgeGroup)
	return Rule{Text: text, Source: source,
		Reasoning: reason(skipped, "falling back to the site default for age group "+ageGroupLabel(p.AgeGroup))}
}

// guestRule is the "Gäste" cascade: we don't know the guest, so they are
// protected last by the rule the link's author chose for their own play.
func guestRule(f core.PlayRuleFacts) Rule {
	var skipped []string
	if r, ok := fromSource(f.LinkWorkshop, obj.ConstraintSourceWorkshop, "the share's workshop", &skipped); ok {
		return r
	}
	if r, ok := fromSource(f.LinkOrganisation, obj.ConstraintSourceOrganisation, "the share's organisation", &skipped); ok {
		return r
	}
	if f.Author != nil {
		r := knownPlayerRule(f, f.Author, false)
		r.Reasoning = reason(skipped, "falling back to the share author's own constraint ("+r.Reasoning+")")
		return r
	}
	return Rule{Reasoning: reason(skipped, "no constraint could be resolved")}
}

// fromSource is the rule a source sets, if it sets one; otherwise it notes
// the source as skipped.
func fromSource(s *core.RuleSource, source, what string, skipped *[]string) (Rule, bool) {
	if s == nil {
		return Rule{}, false
	}
	if text := trimRule(s.Rule); text != nil {
		return Rule{Text: text, Source: source, SourceName: s.Name,
			Reasoning: reason(*skipped, what+" “"+s.Name+"” sets a constraint → applied")}, true
	}
	*skipped = append(*skipped, what+" sets none")
	return Rule{}, false
}

// siteRule is the site's rule for an age group; unknown: the strictest.
func siteRule(f core.PlayRuleFacts, ageGroup *string) (*string, string) {
	group := obj.AgeGroupU13
	if ageGroup != nil {
		group = *ageGroup
	}
	switch group {
	case obj.AgeGroupU13p:
		return trimRule(f.Site[obj.AgeGroupU13p]), obj.ConstraintSourceSite13p
	case obj.AgeGroupU18:
		return trimRule(f.Site[obj.AgeGroupU18]), obj.ConstraintSourceSite18
	}
	return trimRule(f.Site[obj.AgeGroupU13]), obj.ConstraintSourceSite13
}

// reason joins the skipped stages with the decisive one into one clause.
func reason(skipped []string, decisive string) string {
	if len(skipped) == 0 {
		return decisive
	}
	return strings.Join(skipped, "; ") + "; " + decisive
}

// ageGroupLabel is an age group as the reasoning names it.
func ageGroupLabel(ageGroup *string) string {
	if ageGroup == nil {
		return "under 13 / unknown"
	}
	switch *ageGroup {
	case obj.AgeGroupU13:
		return "13–17 without parental consent"
	case obj.AgeGroupU13p:
		return "13–17 with parental consent"
	case obj.AgeGroupU18:
		return "18+"
	}
	return *ageGroup
}

// trimRule is a rule trimmed, nil if empty.
func trimRule(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}
