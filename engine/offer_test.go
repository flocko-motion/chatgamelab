package engine

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"engine/internal/ports"
)

// both allows pictures and narration.
var both = Toggles{Images: true, AudioOut: true}

// mockAdventureOffer allows low and medium with pictures and narration, and
// defaults to medium with pictures only.
func mockAdventureOffer(customizable bool) Offer {
	return Offer{
		Platform: PlatformMock,
		ID:       "share-a",
		Caveats: Caveats{
			Allow:        map[Tier]Toggles{TierLow: both, TierMedium: both},
			Default:      Choice{Tier: TierMedium, Toggles: Toggles{Images: true}},
			Customizable: customizable,
		},
	}
}

// game is a spec and the live offers it runs on, as a platform hands them over.
type game struct {
	spec   SessionSpec
	offers []Offer
}

func (g game) opening() (settlement, []string, bool) { return g.spec.opening(g.offers) }

func check(g game) []Issue { return Validate(g.spec, g.offers) }

func adventure(offers ...Offer) game {
	return game{spec: SessionSpec{Genre: GenreAdventure}, offers: offers}
}

func launchOrFail(t *testing.T, spec game) *Session {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s, err := Launch(ctx, spec.spec, spec.offers)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

// eventually polls, for what arrives on the graph's own goroutines.
func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func spent(s *Session, node string) UsageRecord {
	for _, record := range s.Usage().ByNode {
		if record.Node == node {
			return record
		}
	}
	return UsageRecord{}
}

// --- When the dialog appears ---

func TestOneFixedOfferStartsAtOnce(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false)))
	snapshot := s.Snapshot()
	if snapshot.Choosing || snapshot.Settled == nil {
		t.Fatalf("snapshot = %+v, want settled without a dialog", snapshot)
	}
	if got := snapshot.Settled.Choice; got.Tier != TierMedium || got.Toggles != (Toggles{Images: true}) {
		t.Errorf("settled %+v, want the default", got)
	}
}

func TestOneCustomizableOfferWaitsForTheChoice(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(true)))
	if !s.Snapshot().Choosing {
		t.Fatal("a customizable offer must show the dialog")
	}
	if err := s.Say("hello"); err != ErrChoosing {
		t.Errorf("Say before the choice = %v, want ErrChoosing", err)
	}
}

func TestTwoOffersAlwaysWaitForTheChoice(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false), MockOffer()))
	options := s.Options()
	if !options.Choosing || len(options.Offers) != 2 {
		t.Fatalf("options = %+v, want a dialog with both offers", options)
	}
	if err := s.Choose("mock", nil); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Choosing {
		t.Error("still choosing after the choice")
	}
}

// An offer that cannot run the genre is not shown.
func TestOptionsLeaveOutOffersThatCannotRunTheGenre(t *testing.T) {
	text := mockAdventureOffer(false)
	text.Caveats = Caveats{
		Allow:   map[Tier]Toggles{TierLow: {}},
		Default: Choice{Tier: TierLow},
	}
	spec := game{spec: SessionSpec{Genre: GenreNPCLive}, offers: []Offer{text, MockOffer()}}
	s := launchOrFail(t, spec)
	options := s.Options()
	if len(options.Offers) != 1 || options.Offers[0].Offer != "mock" {
		t.Errorf("options = %+v, want only the offer that runs npc-live", options.Offers)
	}
	// One runnable offer with nothing to choose starts at once.
	if options.Choosing {
		t.Error("a single runnable fixed offer must not show a dialog")
	}
}

// The options carry what the dialog shows: labels and allowances, models as a detail.
func TestOptionsDescribeEachOffer(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(true)))
	offer := s.Options().Offers[0]
	if offer.Name != "Mock" || !offer.Customizable || len(offer.Tiers) != 2 {
		t.Fatalf("offer = %+v", offer)
	}
	low := offer.Tiers[0]
	if low.Tier != TierLow || low.Label != "Economy" || low.Toggles != both || low.Models[ModelImage] == "" {
		t.Errorf("low = %+v", low)
	}
}

// --- The choice ---

func TestChoiceWithinTheCaveatsStartsTheGame(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(true)))
	if err := s.Choose("share-a", &Choice{Tier: TierLow}); err != nil {
		t.Fatal(err)
	}
	settled := s.Snapshot().Settled
	// Less than the default: a lower tier, pictures off.
	if settled.Choice != (Choice{Tier: TierLow}) {
		t.Errorf("settled %+v, want low with everything off", settled.Choice)
	}
	if settled.Preference.Platform != PlatformMock || settled.Preference.Offer != "share-a" {
		t.Errorf("preference = %+v, want the platform and the offer's ref", settled.Preference)
	}
}

func TestChoiceOutsideTheCaveatsIsRefused(t *testing.T) {
	for name, choice := range map[string]Choice{
		"tier not allowed":   {Tier: TierHigh},
		"toggle not allowed": {Tier: TierLow, Toggles: Toggles{AudioIn: true}},
		"unknown tier":       {Tier: "luxurious"},
	} {
		t.Run(name, func(t *testing.T) {
			s := launchOrFail(t, adventure(mockAdventureOffer(true)))
			if err := s.Choose("share-a", &choice); err == nil {
				t.Error("a choice outside the caveats was accepted")
			}
			if !s.Snapshot().Choosing {
				t.Error("a refused choice must leave the dialog up")
			}
		})
	}
}

// With customizable off the default applies, whatever the player sends.
func TestChoiceOnAFixedOfferTakesTheDefault(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false), MockOffer()))
	if err := s.Choose("share-a", &Choice{Tier: TierLow, Toggles: both}); err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().Settled.Choice; got != (Choice{Tier: TierMedium, Toggles: Toggles{Images: true}}) {
		t.Errorf("settled %+v, want the default", got)
	}
}

func TestChoiceMustMeetTheGenresNeeds(t *testing.T) {
	custom := MockOffer()
	custom.Caveats.Customizable = true
	s := launchOrFail(t, game{spec: SessionSpec{Genre: GenreNPCLive}, offers: []Offer{custom}})
	err := s.Choose("mock", &Choice{Tier: TierLow, Toggles: Toggles{AudioLive: true}})
	if err == nil || !strings.Contains(err.Error(), "images") {
		t.Errorf("err = %v, want the missing images refused", err)
	}
}

// Nothing runs before the choice, so nothing is billed.
func TestNothingIsBilledBeforeTheChoice(t *testing.T) {
	s := launchOrFail(t, game{spec: SessionSpec{Genre: GenreNPCLive, Scenario: "a keeper"}, offers: []Offer{
		MockOffer(), secondMock(),
	}})
	time.Sleep(150 * time.Millisecond)
	if cost := s.Usage().TotalCost; cost != 0 {
		t.Fatalf("billed %v before the choice", cost)
	}
	if s.Snapshot().Started {
		t.Fatal("the game started before the choice")
	}

	if err := s.Choose("mock", nil); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the portrait to be billed", func() bool { return spent(s, "portrait").Images == 1 })
}

// A reloading page finds the session still choosing, and the same options.
func TestReloadWhileChoosingShowsTheDialogAgain(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(true), MockOffer()))
	first, again := s.Options(), s.Options()
	if !s.Snapshot().Choosing || !again.Choosing || len(again.Offers) != len(first.Offers) {
		t.Errorf("reload saw %+v, want the dialog as before", again)
	}
	if len(s.Topology().Nodes) == 0 {
		t.Error("the graph must be there while choosing")
	}
}

// --- Toggles during play ---

// Pictures off stops new pictures and their billing; pictures already made stay.
func TestImagesOffStopsNewPicturesAndTheirBilling(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false)))
	if err := s.Say("I cross the bridge"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a picture", func() bool { return spent(s, "image").Images == 1 })

	if err := s.SetToggle(ToggleImages, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Say("I look around"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the second turn", func() bool { return spent(s, "expand").OutputTokens > 350 })
	time.Sleep(50 * time.Millisecond)
	if got := spent(s, "image").Images; got != 1 {
		t.Errorf("pictures billed = %d, want only the one made before images went off", got)
	}
	if n := countHistory(s, "image"); n != 1 {
		t.Errorf("history holds %d pictures, want the one already made", n)
	}
	if pref := s.Preference(); pref.Toggles.Images {
		t.Error("the preference still has images on")
	}
}

func TestToggleOnNeedsTheOffersAllowance(t *testing.T) {
	narrow := mockAdventureOffer(false)
	narrow.Caveats.Allow = map[Tier]Toggles{TierMedium: {Images: true}}
	s := launchOrFail(t, adventure(narrow))
	if err := s.SetToggle(ToggleAudioOut, true); err == nil {
		t.Error("narration went on though the offer does not allow it")
	}
	if err := s.SetToggle(ToggleImages, false); err != nil {
		t.Errorf("off must always be allowed: %v", err)
	}
	if err := s.SetToggle(ToggleImages, true); err != nil {
		t.Errorf("back on within the allowance: %v", err)
	}
}

// A toggle off at the start has a wired block that does nothing.
func TestToggleOffLeavesItsBlockUnbilled(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false)))
	if err := s.Say("I cross the bridge"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the turn", func() bool { return spent(s, "expand").OutputTokens > 0 })
	time.Sleep(50 * time.Millisecond)
	if got := spent(s, "tts"); got.AudioSeconds != 0 {
		t.Errorf("narration billed %v with its toggle off", got)
	}
	if !slices.ContainsFunc(s.Topology().Nodes, func(n ports.TopologyNode) bool { return n.Name == "tts" }) {
		t.Error("the narration block must stay wired")
	}
}

// --- Stored preferences ---

func storedPreference(tier Tier, toggles Toggles) *Preference {
	return &Preference{Platform: PlatformMock, Offer: "share-a", Choice: Choice{Tier: tier, Toggles: toggles}}
}

func TestStoredPreferenceSkipsTheDialog(t *testing.T) {
	spec := adventure(mockAdventureOffer(true), MockOffer())
	spec.spec.Preference = storedPreference(TierLow, both)
	s := launchOrFail(t, spec)
	settled := s.Snapshot().Settled
	if settled == nil || settled.Offer != "share-a" || settled.Choice != (Choice{Tier: TierLow, Toggles: both}) {
		t.Errorf("settled = %+v, want the stored preference on its offer", settled)
	}
}

// A share that no longer allows what the player chose gives way rather than refusing.
func TestStoredPreferenceIsFittedToTheOffers(t *testing.T) {
	narrow := mockAdventureOffer(false)
	narrow.Caveats.Allow = map[Tier]Toggles{TierLow: {AudioOut: true}}
	narrow.Caveats.Default = Choice{Tier: TierLow}
	spec := adventure(narrow)
	spec.spec.Preference = storedPreference(TierMedium, both)

	issues := check(spec)
	if hasError(issues) {
		t.Fatalf("a fitted preference must not be refused: %v", issues)
	}
	s := launchOrFail(t, spec)
	settled := s.Snapshot().Settled
	if settled.Choice != (Choice{Tier: TierLow, Toggles: Toggles{AudioOut: true}}) {
		t.Errorf("settled %+v, want low with narration only", settled.Choice)
	}
	if len(settled.Changes) != 2 {
		t.Errorf("changes = %v, want the tier and the pictures explained", settled.Changes)
	}
	// The wish stays, so a share that allows more again gets it back.
	if settled.Preference.Tier != TierMedium || !settled.Preference.Toggles.Images {
		t.Errorf("preference = %+v, want the wish kept", settled.Preference)
	}
}

func TestClosestTier(t *testing.T) {
	for _, tc := range []struct {
		allowed []Tier
		wanted  Tier
		want    Tier
	}{
		{[]Tier{TierLow, TierMedium, TierHigh}, TierMedium, TierMedium},
		{[]Tier{TierLow, TierMedium}, TierHigh, TierMedium},
		{[]Tier{TierLow, TierHigh}, TierMedium, TierLow},
		{[]Tier{TierHigh}, TierLow, TierHigh},
	} {
		if got := closestTier(tc.allowed, tc.wanted); got != tc.want {
			t.Errorf("closestTier(%v, %s) = %s, want %s", tc.allowed, tc.wanted, got, tc.want)
		}
	}
}

func TestStoredPreferenceKeepsItsSource(t *testing.T) {
	other := mockAdventureOffer(false)
	other.ID = "share-b"
	spec := adventure(other, mockAdventureOffer(false))
	spec.spec.Preference = storedPreference(TierMedium, Toggles{})
	if settled, _, _ := spec.opening(); settled.offer.ID != "share-a" {
		t.Errorf("settled on offer %q, want the one the preference names", settled.offer.ID)
	}
}

func TestStoredPreferenceNeverChangesPlatform(t *testing.T) {
	openai := oneOffer(PlatformOpenAI, TierLow, both)
	spec := adventure(openai)
	spec.spec.Preference = storedPreference(TierLow, Toggles{})
	if !hasErrorAt(check(spec), "preference") {
		t.Errorf("issues = %v, want the platform change refused", check(spec))
	}
}

// --- Adapting a running session ---

func TestAdaptGivesWayToANarrowerShare(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false)))
	narrow := mockAdventureOffer(false)
	narrow.Caveats.Allow = map[Tier]Toggles{TierLow: {}}
	narrow.Caveats.Default = Choice{Tier: TierLow}
	if err := s.Adapt([]Offer{narrow}); err != nil {
		t.Fatal(err)
	}
	settled := s.Snapshot().Settled
	if settled.Choice != (Choice{Tier: TierLow}) || len(settled.Changes) == 0 {
		t.Errorf("settled = %+v, want low, pictures off, and why", settled)
	}

	// The share is restored: the wish comes back.
	if err := s.Adapt([]Offer{mockAdventureOffer(false)}); err != nil {
		t.Fatal(err)
	}
	if got := s.Snapshot().Settled.Choice; got != (Choice{Tier: TierMedium, Toggles: Toggles{Images: true}}) {
		t.Errorf("settled %+v, want the preference back", got)
	}
}

func TestAdaptToAnotherPlatformEndsTheSession(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false)))
	if err := s.Adapt([]Offer{oneOffer(PlatformOpenAI, TierLow, both)}); err == nil {
		t.Fatal("a session switched platform")
	}
	select {
	case <-s.ctx.Done():
	case <-time.After(time.Second):
		t.Error("the session kept running after a hard failure")
	}
}

// The settled event carries the preference for the platform to store.
func TestSettlingIsAnnouncedAndPersisted(t *testing.T) {
	store := &MemoryPersist{}
	spec := adventure(mockAdventureOffer(true))
	spec.spec.Persist = store
	spec.spec.ID = "announced"
	s := launchOrFail(t, spec)
	if err := s.Choose("share-a", &Choice{Tier: TierLow, Toggles: both}); err != nil {
		t.Fatal(err)
	}

	// Delivered by the session's sender, so it arrives shortly after.
	var settled Settled
	eventually(t, "the settled event to be persisted", func() bool {
		for _, ev := range store.Snapshot() {
			if ev.Stream == "settled" && json.Unmarshal([]byte(ev.Value), &settled) == nil {
				return true
			}
		}
		return false
	})
	if settled.Preference.Tier != TierLow || settled.Preference.Toggles != both {
		t.Errorf("persisted %+v, want the player's choice", settled)
	}
}

// --- Validate ---

func TestValidateErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		spec  game
		field string
	}{
		"no offer":           {adventure(), "offers"},
		"unknown genre":      {game{spec: SessionSpec{Genre: "quiz"}, offers: []Offer{MockOffer()}}, "genre"},
		"unknown prompt":     {game{spec: SessionSpec{Genre: GenreNPCLive, PromptOverride: map[string]string{"portrait-x": ""}}, offers: []Offer{MockOffer()}}, "promptOverride.portrait-x"},
		"invalid voice":      {game{spec: SessionSpec{Genre: GenreNPCLive, Voice: "nobody"}, offers: []Offer{oneOffer(PlatformOpenAI, TierLow, live)}}, "voice"},
		"adventure override": {game{spec: SessionSpec{Genre: GenreAdventure, PromptOverride: map[string]string{"observer": ""}}, offers: []Offer{MockOffer()}}, "promptOverride.observer"},
		"no offer runs it":   {game{spec: SessionSpec{Genre: GenreNPCLive}, offers: []Offer{oneOffer(PlatformMistral, TierLow, Toggles{Images: true})}}, "offers"},
	} {
		t.Run(name, func(t *testing.T) {
			if issues := check(tc.spec); !hasErrorAt(issues, tc.field) {
				t.Errorf("issues = %v, want an error at %s", issues, tc.field)
			}
		})
	}
}

// A broken offer is left out: its errors become warnings, and only a spec
// with no usable offer is refused.
func TestBrokenOffersAreLeftOut(t *testing.T) {
	outside := mockAdventureOffer(false)
	outside.Caveats.Default = Choice{Tier: TierHigh}
	lacking := oneOffer(PlatformMistral, TierLow, Toggles{AudioLive: true})
	keyless := oneOffer(PlatformOpenAI, TierLow, Toggles{})
	keyless.Keys = nil
	twice := oneOffer(PlatformMock, TierHigh, Toggles{})
	twice.Caveats.Allow["max"] = Toggles{}
	nameless := MockOffer()
	nameless.ID = ""

	for name, tc := range map[string]struct {
		offer Offer
		field string
	}{
		"unknown platform": {oneOffer("ionos", TierLow, Toggles{}), "offers[0].platform"},
		"default outside":  {outside, "offers[0].caveats.default"},
		"toggle it lacks":  {lacking, "offers[0].caveats.allow.low.audio-live"},
		"no key":           {keyless, "offers[0].keys"},
		"tier named twice": {twice, "offers[0].caveats.allow.max"},
		"no id":            {nameless, "offers[0].id"},
	} {
		t.Run(name, func(t *testing.T) {
			alone := check(adventure(tc.offer))
			if !hasErrorAt(alone, "offers") || !hasWarningAt(alone, tc.field) {
				t.Errorf("alone: issues = %v, want no usable offer and a warning at %s", alone, tc.field)
			}

			bag := adventure(tc.offer, secondMock())
			issues := check(bag)
			if hasError(issues) {
				t.Fatalf("a broken offer next to a sound one must not refuse the spec: %v", issues)
			}
			if !hasWarningAt(issues, "offers[0]") || !hasWarningAt(issues, tc.field) {
				t.Errorf("issues = %v, want it left out with its problem as a warning", issues)
			}
			s := launchOrFail(t, bag)
			if settled := s.Snapshot().Settled; settled == nil || settled.Offer != "mock-2" {
				t.Errorf("settled = %+v, want the sound offer", settled)
			}
		})
	}
}

// An ID an earlier offer uses leaves the later one out.
func TestDuplicateIDIsLeftOut(t *testing.T) {
	custom := MockOffer()
	custom.Caveats.Customizable = true
	spec := adventure(MockOffer(), custom)
	if issues := check(spec); hasError(issues) || !hasWarningAt(issues, "offers[1].id") {
		t.Errorf("issues = %v, want the second offer left out", issues)
	}
	s := launchOrFail(t, spec)
	if s.Snapshot().Choosing {
		t.Error("the left-out customizable offer still asked for a choice")
	}
}

func TestAdaptLeavesOutBrokenOffers(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false)))
	broken := mockAdventureOffer(false)
	broken.Caveats.Default = Choice{Tier: TierHigh}
	sound := mockAdventureOffer(false)
	sound.ID = "share-b"
	if err := s.Adapt([]Offer{broken, sound}); err != nil {
		t.Fatalf("a broken offer next to a sound one ended the session: %v", err)
	}
	if got := s.Snapshot().Settled.Offer; got != "share-b" {
		t.Errorf("settled on %q, want the sound offer", got)
	}
	if got := len(s.Options().Offers); got != 1 {
		t.Errorf("options list %d offers, want only the sound one", got)
	}
}

// Nobody reads the stream: events pile up to the buffer's end, and the
// session must still answer.
func TestNoReaderNeverHangsTheSession(t *testing.T) {
	s := launchOrFail(t, adventure(mockAdventureOffer(false)))
	for i := range 2 * cap(s.events) {
		if err := s.SetToggle(ToggleImages, i%2 == 0); err != nil {
			t.Fatal(err)
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = s.SetToggle(ToggleImages, false)
		_ = s.Snapshot()
		_ = s.Options()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the session hung on a full stream")
	}
}

func hasWarningAt(issues []Issue, field string) bool {
	for _, i := range issues {
		if i.Severity == SeverityWarning && i.Field == field {
			return true
		}
	}
	return false
}

func TestValidateWarnings(t *testing.T) {
	spec := adventure(oneOffer(PlatformMock, TierLow, Toggles{}))
	spec.spec.ImageStyle = "watercolour"
	issues := check(spec)
	if hasError(issues) {
		t.Fatalf("a warning must not stop a launch: %v", issues)
	}
	if !slices.ContainsFunc(issues, func(i Issue) bool {
		return i.Severity == SeverityWarning && i.Field == "imageStyle"
	}) {
		t.Errorf("issues = %v, want a warning that the style is unused", issues)
	}
}

func countHistory(s *Session, stream string) int {
	n := 0
	for _, ev := range s.History() {
		if ev.Stream == stream {
			n++
		}
	}
	return n
}

// secondMock is the mock under another ID, so two mock offers make a bag.
func secondMock() Offer {
	o := MockOffer()
	o.ID = "mock-2"
	return o
}
