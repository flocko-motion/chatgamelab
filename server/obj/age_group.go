package obj

// Age-group cohorts — stable wire values used in the `ageGroup` JSON field
// and persisted in app_user.age_group. The CHECK constraint on that column
// enforces this exact set (plus NULL for guests).
//
// Keep in sync with web/src/constants/ageGroup.ts.
const (
	AgeGroupU13  = "u13"  // 13-17, no parental consent on file (strictest; fallback when age is unknown)
	AgeGroupU13p = "u13p" // 13-17, with parental consent on file
	AgeGroupU18  = "u18"  // 18+
)

// Prompt-constraint source labels — written to game_session_message.prompt_constraint_source
// and surfaced in the player's session-details UI. Stable wire values.
const (
	ConstraintSourceWorkshop     = "workshop"
	ConstraintSourceOrganisation = "organisation"
	ConstraintSourceSite13       = "site13"  // matches AgeGroupU13
	ConstraintSourceSite13p      = "site13p" // matches AgeGroupU13p
	ConstraintSourceSite18       = "site18"  // matches AgeGroupU18
)
