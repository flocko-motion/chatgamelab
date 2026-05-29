package db

import (
	"context"
	"strings"

	"cgl/obj"

	"github.com/google/uuid"
)

// This file is the single, canonical home for Jugendschutz (youth-protection)
// constraint resolution. It is the code counterpart of JUGENDSCHUTZ.md, section
// "Wer bekommt welchen Constraint?". Keep the two in lockstep: the public
// ResolveConstraint below is the ONLY entry point callers should use, and the two
// private cascades map one-to-one onto the spec's two rule sets.

// ConstraintResolution is the full result of resolving which youth-protection
// constraint applies. Reasoning is a human-readable trace, assembled branch-by-branch
// as the decision is made, so the AI insights view can explain *why* a constraint was
// chosen (e.g. "Player is an anonymous guest playing via a shared link; the share's
// workshop “Kinderworkshop (Org X)” sets a constraint → applied").
type ConstraintResolution struct {
	Text       *string // constraint text (nil only if a site default is unset)
	Source     string  // obj.ConstraintSource* label
	SourceName string  // human-readable origin (workshop/org name); empty for site-by-age
	Reasoning  string  // human-readable trace of the decision path
}

// ResolveConstraint decides which youth-protection constraint applies to a play
// situation. It is the canonical entry point — no caller outside this file should
// resolve constraints any other way.
//
// The discriminator is WHO is playing, never how they arrived (JUGENDSCHUTZ.md):
//
//   - Anonymous guest (a user created for a share link — marked by PrivateShareID —
//     or no user at all) → the "Gäste" cascade (resolveGuestShareConstraint):
//     share-workshop → share-org → the share author's own constraint.
//   - Any authenticated, non-guest player → the "Angemeldete Nutzer" cascade
//     (resolveAuthenticatedConstraint):
//     share-workshop → share-org → own-workshop → own-org → own-age.
//
// player is the user playing (real user, guest user, or nil for a pure anonymous
// context). share is the share the game is played through, or nil for direct play
// — its workshop/org only feed stages 1–2 when present.
func ResolveConstraint(ctx context.Context, player *obj.User, share *obj.GameShare) ConstraintResolution {
	if player != nil && !IsGuestUser(ctx, player.ID) {
		res := resolveAuthenticatedConstraint(ctx, player, share)
		res.Reasoning = "Player is an authenticated user (" + ageGroupLabel(player.AgeGroup) + "); " + res.Reasoning
		return res
	}
	if share != nil {
		res := resolveGuestShareConstraint(ctx, share)
		res.Reasoning = "Player is an anonymous guest playing via a shared link; " + res.Reasoning
		return res
	}
	// Anonymous play with no share context should not occur; stay safe with the
	// strictest site default rather than returning no constraint at all.
	text, source := resolveAgeConstraint(ctx, nil)
	return ConstraintResolution{
		Text:      text,
		Source:    source,
		Reasoning: "No player and no share context; applying the strictest site default (" + ageGroupLabel(nil) + ").",
	}
}

// IsGuestUser reports whether a user is an anonymous guest account — one created for
// a private share link, marked by a non-NULL private_share_id. ResolveConstraint
// treats guests as anonymous: we don't know who they are, so they fall back to the
// share author's protection rather than their own age. Mirrors the guest check in
// canAccessGame (permissions.go). Also used by the play routes to gate session access.
func IsGuestUser(ctx context.Context, userID uuid.UUID) bool {
	appUser, err := queries().GetUserByID(ctx, userID)
	return err == nil && appUser.PrivateShareID.Valid
}

// resolveAuthenticatedConstraint implements the JUGENDSCHUTZ.md "Angemeldete Nutzer"
// cascade for a known (non-guest) player. First non-empty stage wins:
//  1. Workshop from the share (if `share` references a workshop)
//  2. Organisation from the share (if `share` references an org)
//  3. Workshop from the player's own context
//  4. Organisation from the player's own context
//  5. Site constraint by the player's age group (always configured, so a known
//     player always receives a non-empty source label).
//
// share is nil when no share context applies (direct play of an owned game, or the
// recursive author lookup inside resolveGuestShareConstraint).
func resolveAuthenticatedConstraint(ctx context.Context, user *obj.User, share *obj.GameShare) ConstraintResolution {
	var skipped []string
	if share != nil {
		if share.WorkshopID != nil {
			if c, name := resolveShareWorkshopConstraint(ctx, *share.WorkshopID); c != nil {
				return ConstraintResolution{c, obj.ConstraintSourceWorkshop, name,
					reason(skipped, "the share's workshop “"+name+"” sets a constraint → applied")}
			}
			skipped = append(skipped, "the share's workshop sets none")
		}
		if share.InstitutionID != nil {
			if inst, err := queries().GetInstitutionByID(ctx, *share.InstitutionID); err == nil && inst.PromptConstraints.Valid {
				if c := trimConstraint(&inst.PromptConstraints.String); c != nil {
					return ConstraintResolution{c, obj.ConstraintSourceOrganisation, inst.Name,
						reason(skipped, "the share's organisation “"+inst.Name+"” sets a constraint → applied")}
				}
			}
			skipped = append(skipped, "the share's organisation sets none")
		}
	}
	if user.Role != nil && user.Role.Workshop != nil {
		if c := trimConstraint(user.Role.Workshop.PromptConstraints); c != nil {
			name := formatWorkshopSourceName(user.Role.Workshop)
			return ConstraintResolution{c, obj.ConstraintSourceWorkshop, name,
				reason(skipped, "the player's workshop “"+name+"” sets a constraint → applied")}
		}
		skipped = append(skipped, "the player's workshop sets none")
	}
	if user.Role != nil && user.Role.Institution != nil {
		if c := trimConstraint(user.Role.Institution.PromptConstraints); c != nil {
			return ConstraintResolution{c, obj.ConstraintSourceOrganisation, user.Role.Institution.Name,
				reason(skipped, "the player's organisation “"+user.Role.Institution.Name+"” sets a constraint → applied")}
		}
		skipped = append(skipped, "the player's organisation sets none")
	}
	text, source := resolveAgeConstraint(ctx, user.AgeGroup)
	return ConstraintResolution{text, source, "",
		reason(skipped, "falling back to the site default for age group "+ageGroupLabel(user.AgeGroup))}
}

// resolveGuestShareConstraint implements the JUGENDSCHUTZ.md "Gäste" cascade for an
// anonymous guest playing via a shared link. First non-empty stage wins:
//  1. Workshop from the share
//  2. Organisation from the share
//  3. The share author's own constraint — resolveAuthenticatedConstraint applied to
//     the account that created the share (with share=nil; stages 1–2 are handled here).
//
// Stage 3 means: if the game carries no workshop/org context, the constraint that
// would protect the author themselves is used. We don't know the guest, so the
// player is protected by whatever the publisher chose for their own play.
func resolveGuestShareConstraint(ctx context.Context, gameShare *obj.GameShare) ConstraintResolution {
	var skipped []string
	if gameShare.WorkshopID != nil {
		if c, name := resolveShareWorkshopConstraint(ctx, *gameShare.WorkshopID); c != nil {
			return ConstraintResolution{c, obj.ConstraintSourceWorkshop, name,
				reason(skipped, "the share's workshop “"+name+"” sets a constraint → applied")}
		}
		skipped = append(skipped, "the share's workshop sets none")
	}
	if gameShare.InstitutionID != nil {
		inst, err := queries().GetInstitutionByID(ctx, *gameShare.InstitutionID)
		if err == nil && inst.PromptConstraints.Valid {
			if c := trimConstraint(&inst.PromptConstraints.String); c != nil {
				return ConstraintResolution{c, obj.ConstraintSourceOrganisation, inst.Name,
					reason(skipped, "the share's organisation “"+inst.Name+"” sets a constraint → applied")}
			}
		}
		skipped = append(skipped, "the share's organisation sets none")
	}
	if gameShare.CreatedBy != nil {
		author, err := GetUserByID(ctx, *gameShare.CreatedBy)
		if err == nil && author != nil {
			res := resolveAuthenticatedConstraint(ctx, author, nil)
			res.Reasoning = reason(skipped, "falling back to the share author's own constraint ("+res.Reasoning+")")
			return res
		}
	}
	return ConstraintResolution{Reasoning: reason(skipped, "no constraint could be resolved")}
}

// reason joins the skipped-stage notes with the decisive note into one readable clause.
func reason(skipped []string, decisive string) string {
	if len(skipped) == 0 {
		return decisive
	}
	return strings.Join(skipped, "; ") + "; " + decisive
}

// ageGroupLabel renders an age-group code as a human-readable label for reasoning text.
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

// resolveShareWorkshopConstraint reads a share's workshop constraint directly via the
// data layer, bypassing the user-facing access checks in GetWorkshopByID. Constraint
// resolution runs in a system context (e.g. anonymous guest play) with no acting user;
// GetWorkshopByID would reject a private workshop for the nil user, silently dropping
// the workshop constraint and falling through to the org. This mirrors how the org
// branch reads its constraint via queries().GetInstitutionByID directly.
// Returns the trimmed constraint and source name, or (nil, "") if absent/unloadable.
func resolveShareWorkshopConstraint(ctx context.Context, workshopID uuid.UUID) (*string, string) {
	w, err := queries().GetWorkshopByID(ctx, workshopID)
	if err != nil {
		return nil, ""
	}
	c := trimConstraint(nullStringToPtr(w.PromptConstraints))
	if c == nil {
		return nil, ""
	}
	name := w.Name
	if instName, err := GetInstitutionName(ctx, w.InstitutionID); err == nil && instName != "" {
		name = w.Name + " (" + instName + ")"
	}
	return c, name
}

// formatWorkshopSourceName renders a workshop's human-readable origin as
// 'WorkshopName (OrganisationName)' when the parent org is known, or just the
// workshop name otherwise.
func formatWorkshopSourceName(w *obj.Workshop) string {
	if w == nil {
		return ""
	}
	if w.Institution != nil && w.Institution.Name != "" {
		return w.Name + " (" + w.Institution.Name + ")"
	}
	return w.Name
}

// resolveAgeConstraint returns the site-level constraint for the user's age group.
// See obj.AgeGroup* and obj.ConstraintSource* for the value meanings.
// Always returns the source label, even when no constraint text is configured.
func resolveAgeConstraint(ctx context.Context, ageGroup *string) (*string, string) {
	// Default to the strictest cohort when age is unknown.
	effectiveGroup := obj.AgeGroupU13
	if ageGroup != nil {
		effectiveGroup = *ageGroup
	}
	settings, err := GetSystemSettings(ctx)
	switch effectiveGroup {
	case obj.AgeGroupU13:
		if err == nil && settings != nil {
			return trimConstraint(settings.PromptConstraintU13), obj.ConstraintSourceSite13
		}
		return nil, obj.ConstraintSourceSite13
	case obj.AgeGroupU13p:
		if err == nil && settings != nil {
			return trimConstraint(settings.PromptConstraintU13p), obj.ConstraintSourceSite13p
		}
		return nil, obj.ConstraintSourceSite13p
	case obj.AgeGroupU18:
		if err == nil && settings != nil {
			return trimConstraint(settings.PromptConstraintU18), obj.ConstraintSourceSite18
		}
		return nil, obj.ConstraintSourceSite18
	}
	// Unknown/invalid value — fall back to strictest bucket.
	return nil, obj.ConstraintSourceSite13
}

// trimConstraint returns the trimmed constraint string, or nil if empty.
func trimConstraint(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
