// package: obj / shared domain objects
// type:    logic
// job:     caps an AI quality tier at the tier a free-use key is released up to.
// limits:  pure comparison; which tier is requested or released is decided by the callers (-> db/ai_quality_tier.go).
package obj

// Who released a free-use key up to a tier, named when a session plays below the tier it asked for.
const (
	AiQualityTierLimitedByOrganization = "organization"
	AiQualityTierLimitedByServer       = "server"
)

// The key a user's own tier plays with, outside a workshop.
const (
	UserTierSourceOwn          = "own"          // their own default key: every tier
	UserTierSourceOrganization = "organization" // their organisation's free-use key, up to its release
	UserTierSourceServer       = "server"       // the server's free-use key, up to its release
	UserTierSourceNone         = "none"         // no key they could play with
)

// CapAiQualityTier returns requested, lowered to limit when it is higher.
// capped reports whether it was lowered. An empty or unknown limit caps nothing.
func CapAiQualityTier(requested, limit string) (applied string, capped bool) {
	limitRank := AiModelTierPriority(limit)
	if limitRank < 0 || AiModelTierPriority(requested) <= limitRank {
		return requested, false
	}
	return limit, true
}
