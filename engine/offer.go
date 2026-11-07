package engine

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"engine/internal/adapters"
	"engine/internal/adapters/openai"
	"engine/internal/genre"
)

// Offer is one way a session may run: a platform, a key and the caveats on
// using that key. A platform hands a session one or more offers, rebuilt from
// its key cascade each time, so they may change while a game runs.
type Offer struct {
	Platform Platform `json:"platform"`

	// ID names the offer uniquely, such as the shared key's id. A choice and a
	// stored preference name an offer by it, as the offers are rebuilt each time.
	ID string `json:"id"`

	// Keys is a function so the persisted spec holds no secret and a rotated key applies on the next call.
	Keys KeyFunc `json:"-"`

	Caveats Caveats `json:"caveats"`
}

// Caveats select what an offer allows. They never change a model or a
// parameter: those are the tier table's.
type Caveats struct {
	// Allow maps each tier the offer may be used at to the toggles allowed there;
	// a tier with no entry is not offered.
	Allow map[Tier]Toggles `json:"allow"`

	// Default is what a session gets when nobody chooses, inside Allow.
	Default Choice `json:"default"`

	// Customizable lets the player choose within Allow; off, Default applies.
	Customizable bool `json:"customizable,omitempty"`
}

// Choice is a tier and the toggles on at it.
type Choice struct {
	Tier    Tier    `json:"tier"`
	Toggles Toggles `json:"toggles"`
}

// Preference is the player's wish, persisted with the session: a platform, the
// chosen offer's ID and a choice. Every settle fits it to the offers at hand.
type Preference struct {
	Platform Platform `json:"platform"`
	Offer    string   `json:"offer,omitempty"`
	Choice
}

// MockOffer is the mock platform with every tier and toggle allowed and no
// choice to make: what a dev launcher or a test runs a spec on when it makes no
// offer. An embedded engine never falls back to it.
func MockOffer() Offer {
	all := Toggles{Images: true, AudioOut: true, AudioIn: true, AudioLive: true}
	return Offer{
		Platform: PlatformMock,
		ID:       "mock",
		Caveats: Caveats{
			Allow:   map[Tier]Toggles{TierLow: all, TierMedium: all, TierHigh: all},
			Default: Choice{Tier: TierMedium, Toggles: all},
		},
	}
}

// Severity says whether an issue stops a launch.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue is one finding about a spec. Field is a path into the spec's JSON,
// such as "offers[0].caveats.default".
type Issue struct {
	Severity Severity `json:"severity"`
	Field    string   `json:"field"`
	Message  string   `json:"message"`
}

func (i Issue) String() string { return i.Field + ": " + i.Message }

// settlement is what a session runs on: one offer and one choice within it.
type settlement struct {
	offer Offer
	Choice
}

// Validate reports everything wrong with a spec and the offers it would run on,
// before anything is built. Launch and Resume refuse any error; a platform may
// call it ahead of a launch. An issue's field "offers[i]" names an offer by its
// position in the offers given.
func Validate(spec SessionSpec, offers []Offer) []Issue {
	var issues []Issue
	report := func(severity Severity, field, format string, args ...any) {
		issues = append(issues, Issue{Severity: severity, Field: field, Message: fmt.Sprintf(format, args...)})
	}

	knownGenre := slices.Contains(Genres(), spec.Genre)
	if !knownGenre {
		report(SeverityError, "genre", "unknown genre %q", spec.Genre)
	}

	for _, name := range slices.Sorted(maps.Keys(spec.PromptOverride)) {
		if !slices.Contains(genrePrompts(spec.Genre), name) {
			report(SeverityError, "promptOverride."+name, "no such prompt %q (%s has %s)",
				name, spec.Genre, strings.Join(genrePrompts(spec.Genre), ", "))
		}
	}

	if len(offers) == 0 {
		report(SeverityError, "offers", "the spec makes no offer")
		return issues
	}
	// An offer with errors is left out, and its errors are warnings; only no
	// usable offer at all refuses the spec.
	sound := make([]bool, len(offers))
	for i, found := range offersIssues(offers) {
		sound[i] = !hasError(found)
		if sound[i] {
			issues = append(issues, found...)
			continue
		}
		for _, issue := range found {
			issue.Severity = SeverityWarning
			issues = append(issues, issue)
		}
		report(SeverityWarning, fmt.Sprintf("offers[%d]", i), "left out: the offer has errors")
	}
	if !slices.Contains(sound, true) {
		report(SeverityError, "offers", "no usable offer")
		return issues
	}
	speaksOpenAI := false
	for i, offer := range offers {
		speaksOpenAI = speaksOpenAI || (sound[i] && offer.Platform == PlatformOpenAI)
	}
	if speaksOpenAI && spec.Voice != "" && !openai.ValidVoice(spec.Voice) {
		report(SeverityError, "voice", "unknown voice %q", spec.Voice)
	}
	if !knownGenre {
		return issues
	}

	runnable := runnableOffers(spec.Genre, offers, sound)
	switch {
	case spec.Preference != nil:
		_, changes, fail := fit(*spec.Preference, offers, sound, spec.Genre)
		if fail != "" {
			report(SeverityError, "preference", "%s", fail)
		}
		for _, change := range changes {
			report(SeverityWarning, "preference", "%s", change)
		}
	case len(runnable) == 0:
		report(SeverityError, "offers", "no offer can run %s, which needs %v", spec.Genre, spec.Genre.Needs())
	}

	if spec.ImageStyle != "" && !slices.ContainsFunc(runnable, func(i int) bool { return allowsImages(offers[i]) }) {
		report(SeverityWarning, "imageStyle", "no offer allows images, so the style is unused")
	}
	return issues
}

// offerIssues checks one offer's caveats against its platform.
func offerIssues(field string, offer Offer) []Issue {
	var issues []Issue
	report := func(severity Severity, field, format string, args ...any) {
		issues = append(issues, Issue{Severity: severity, Field: field, Message: fmt.Sprintf(format, args...)})
	}

	entry, known := catalogueEntry(offer.Platform)
	if !known {
		report(SeverityError, field+".platform", "unknown platform %q", offer.Platform)
		return issues
	}
	if offer.Keys == nil && offer.Platform != PlatformMock {
		report(SeverityError, field+".keys", "platform %q needs a Keys function", offer.Platform)
	}

	caveats := field + ".caveats"
	if len(offer.Caveats.Allow) == 0 {
		report(SeverityError, caveats+".allow", "the offer allows no tier")
	}
	seen := map[Tier]Tier{}
	for _, tier := range slices.Sorted(maps.Keys(offer.Caveats.Allow)) {
		at := caveats + ".allow." + string(tier)
		if !tier.Known() {
			report(SeverityError, at, "unknown tier %q", tier)
			continue
		}
		if tier.Retired() {
			report(SeverityWarning, at, "%q is read as %q", tier, tier.Normal())
		}
		if other, twice := seen[tier.Normal()]; twice {
			report(SeverityError, at, "%q and %q are the same tier", other, tier)
		}
		seen[tier.Normal()] = tier
		for _, toggle := range offer.Caveats.Allow[tier].Beyond(entry.Toggles) {
			report(SeverityError, at+"."+string(toggle), "platform %q has no %s", offer.Platform, toggle)
		}
	}

	def := offer.Caveats.Default
	issues = append(issues, tierIssues(caveats+".default.tier", def.Tier)...)
	if def.Tier.Known() {
		allowed, ok := offer.Caveats.allowed(def.Tier)
		switch {
		case !ok:
			report(SeverityError, caveats+".default", "tier %q is not in the offer's allowance", def.Tier.Normal())
		case len(def.Toggles.Beyond(allowed)) > 0:
			report(SeverityError, caveats+".default", "%v not allowed at %s", def.Toggles.Beyond(allowed), def.Tier.Normal())
		}
	}
	return issues
}

// tierIssues refuses an unknown tier and warns on a retired one.
func tierIssues(field string, tier Tier) []Issue {
	switch {
	case !tier.Known():
		return []Issue{{Severity: SeverityError, Field: field, Message: fmt.Sprintf("unknown tier %q", tier)}}
	case tier.Retired():
		return []Issue{{Severity: SeverityWarning, Field: field, Message: fmt.Sprintf("%q is read as %q", tier, tier.Normal())}}
	}
	return nil
}

// allowed is the toggles allowed at a tier, reading retired names.
func (c Caveats) allowed(tier Tier) (Toggles, bool) {
	for name, toggles := range c.Allow {
		if name.Normal() == tier.Normal() {
			return toggles, true
		}
	}
	return Toggles{}, false
}

// tiers is the offer's allowed tiers, cheapest first.
func (c Caveats) tiers() []Tier {
	var out []Tier
	for _, tier := range adapters.Tiers() {
		if _, ok := c.allowed(tier); ok {
			out = append(out, tier)
		}
	}
	return out
}

// runnable reports whether an offer can run a genre: at its default when the
// player may not choose, otherwise at some allowed tier.
func runnable(g Genre, offer Offer) bool {
	if !offer.Caveats.Customizable {
		return missing(g, offer.Caveats.Default.Toggles) == nil
	}
	for _, tier := range offer.Caveats.tiers() {
		allowed, _ := offer.Caveats.allowed(tier)
		if missing(g, allowed) == nil {
			return true
		}
	}
	return false
}

// runnableOffers indexes the sound offers that can run the genre: the ones a
// player is shown.
func runnableOffers(g Genre, offers []Offer, sound []bool) []int {
	var out []int
	for i, offer := range offers {
		if sound[i] && runnable(g, offer) {
			out = append(out, i)
		}
	}
	return out
}

func allowsImages(offer Offer) bool {
	for _, toggles := range offer.Caveats.Allow {
		if toggles.Images {
			return true
		}
	}
	return false
}

// opening decides how a session begins: settled at once from a stored
// preference or a single offer with nothing to choose, or waiting for the
// player. Validate has passed.
func (spec SessionSpec) opening(offers []Offer) (settled settlement, changes []string, choosing bool) {
	sound := soundOffers(offers)
	if spec.Preference != nil {
		settled, changes, _ = fit(*spec.Preference, offers, sound, spec.Genre)
		return settled, changes, false
	}
	runnable := runnableOffers(spec.Genre, offers, sound)
	if len(runnable) == 1 && !offers[runnable[0]].Caveats.Customizable {
		offer := offers[runnable[0]]
		return settlement{offer: offer, Choice: normal(offer.Caveats.Default)}, nil, false
	}
	return settlement{}, nil, true
}

// fit settles a stored preference against the offers at hand: the offer with
// the preference's ID, otherwise another on the same platform. It never refuses
// a preference for being outside the caveats: it takes what is still allowed
// and says what changed. It fails only when no offer runs on the preference's
// platform, as a session never changes platform, or when what remains no
// longer meets the genre's needs.
func fit(pref Preference, offers []Offer, sound []bool, g Genre) (settlement, []string, string) {
	index := -1
	for i, offer := range offers {
		if !sound[i] || offer.Platform != pref.Platform {
			continue
		}
		if index < 0 || (pref.Offer != "" && offer.ID == pref.Offer && offers[index].ID != pref.Offer) {
			index = i
		}
	}
	if index < 0 {
		return settlement{}, nil, fmt.Sprintf("no offer runs on %s, and a session never changes platform", pref.Platform)
	}
	offer := offers[index]

	var changes []string
	wanted := pref.Tier.Normal()
	tier := closestTier(offer.Caveats.tiers(), wanted)
	if tier != wanted {
		changes = append(changes, fmt.Sprintf("tier %s instead of %s: %s is no longer allowed", tier, wanted, wanted))
	}
	allowed, _ := offer.Caveats.allowed(tier)
	toggles := pref.Toggles
	for _, toggle := range pref.Toggles.Beyond(allowed) {
		toggles = toggles.With(toggle, false)
		changes = append(changes, fmt.Sprintf("%s off: no longer allowed at %s", toggle, tier))
	}

	if lacks := missing(g, toggles); lacks != nil {
		return settlement{}, changes, fmt.Sprintf("%s needs %v, which the offer no longer allows", g, lacks)
	}
	return settlement{offer: offer, Choice: Choice{Tier: tier, Toggles: toggles}}, changes, ""
}

// closestTier is the wanted tier if allowed, else the highest allowed below
// it, else the lowest allowed above it.
func closestTier(allowed []Tier, wanted Tier) Tier {
	rank := func(t Tier) int { return slices.Index(adapters.Tiers(), t) }
	best := Tier("")
	for _, tier := range allowed {
		if tier == wanted {
			return tier
		}
		if rank(tier) < rank(wanted) {
			best = tier
		}
	}
	if best != "" {
		return best
	}
	for _, tier := range allowed {
		if rank(tier) > rank(wanted) {
			return tier
		}
	}
	return wanted
}

// chosen checks a choice sent from the dialog, naming an offer by its ID.
// Unlike a stored preference it is refused outside the caveats: the player was
// shown what is allowed.
func chosen(g Genre, offers []Offer, id string, choice *Choice) (settlement, error) {
	sound := soundOffers(offers)
	index := slices.IndexFunc(offers, func(o Offer) bool { return o.ID == id })
	if id == "" || index < 0 || !sound[index] || !runnable(g, offers[index]) {
		return settlement{}, fmt.Errorf("there is no offer %q to choose", id)
	}
	offer := offers[index]
	picked := offer.Caveats.Default
	if choice != nil && offer.Caveats.Customizable {
		picked = *choice
		if !picked.Tier.Known() {
			return settlement{}, fmt.Errorf("unknown tier %q", picked.Tier)
		}
		allowed, ok := offer.Caveats.allowed(picked.Tier)
		if !ok {
			return settlement{}, fmt.Errorf("offer %q does not allow tier %s", id, picked.Tier.Normal())
		}
		if beyond := picked.Toggles.Beyond(allowed); len(beyond) > 0 {
			return settlement{}, fmt.Errorf("offer %q does not allow %v at %s", id, beyond, picked.Tier.Normal())
		}
	}
	if lacks := missing(g, picked.Toggles); lacks != nil {
		return settlement{}, fmt.Errorf("%s needs %v", g, lacks)
	}
	return settlement{offer: offer, Choice: normal(picked)}, nil
}

// normal reads a retired tier name in a choice.
func normal(c Choice) Choice {
	c.Tier = c.Tier.Normal()
	return c
}

// offersIssues checks every offer: its caveats against its platform, and an ID
// no earlier offer uses.
func offersIssues(offers []Offer) [][]Issue {
	out := make([][]Issue, len(offers))
	seen := map[string]bool{}
	for i, offer := range offers {
		field := fmt.Sprintf("offers[%d]", i)
		found := offerIssues(field, offer)
		switch {
		case offer.ID == "":
			found = append(found, Issue{Severity: SeverityError, Field: field + ".id", Message: "the offer has no id"})
		case seen[offer.ID]:
			found = append(found, Issue{Severity: SeverityError, Field: field + ".id",
				Message: fmt.Sprintf("id %q is used by an earlier offer", offer.ID)})
		}
		seen[offer.ID] = true
		out[i] = found
	}
	return out
}

// soundOffers marks the offers without errors: the ones a session may use.
func soundOffers(offers []Offer) []bool {
	sound := make([]bool, len(offers))
	for i, found := range offersIssues(offers) {
		sound[i] = !hasError(found)
	}
	return sound
}

// genrePrompts is every prompt a genre lets a spec replace.
func genrePrompts(g Genre) []string {
	switch g {
	case GenreNPCLive:
		return slices.Sorted(maps.Keys(genre.NPCLivePrompts))
	}
	return nil
}

func hasError(issues []Issue) bool {
	return slices.ContainsFunc(issues, func(i Issue) bool { return i.Severity == SeverityError })
}

// refusal joins a spec's errors into the one error Launch returns.
func refusal(issues []Issue) error {
	var errs []string
	for _, i := range issues {
		if i.Severity == SeverityError {
			errs = append(errs, i.String())
		}
	}
	if errs == nil {
		return nil
	}
	return fmt.Errorf("invalid session spec: %s", strings.Join(errs, "; "))
}

// catalogueEntry is a platform's own entry, tier tables and all.
func catalogueEntry(id Platform) (adapters.Platform, bool) {
	for _, p := range catalogueEntries() {
		if Platform(p.ID) == id {
			return p, true
		}
	}
	return adapters.Platform{}, false
}
