package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"engine/internal/adapters"
)

// ErrChoosing is what playing a session answers before the player has chosen.
var ErrChoosing = errors.New("the session is waiting for the player's choice")

// Settled is what a session runs on, sent on the `settled` stream whenever it
// is decided or changes. Preference is the player's wish, for the platform to
// store with the spec; Choice is what runs, which differs from the wish when the
// offers no longer allow it, and Changes says how.
type Settled struct {
	Offer      string     `json:"offer"`
	Platform   Platform   `json:"platform"`
	Choice     Choice     `json:"choice"`
	Preference Preference `json:"preference"`
	Changes    []string   `json:"changes,omitempty"`
}

// Options is what the player may choose: the offers that can run the game,
// what each allows, and what the session settled on once it has.
type Options struct {
	Choosing bool          `json:"choosing"`
	Offers   []OfferOption `json:"offers"`
	Settled  *Settled      `json:"settled,omitempty"`
}

// OfferOption is one offer as a player sees it.
type OfferOption struct {
	// Offer is the offer's ID, which a choice names.
	Offer        string       `json:"offer"`
	Platform     Platform     `json:"platform"`
	Name         string       `json:"name"`
	Customizable bool         `json:"customizable"`
	Default      Choice       `json:"default"`
	Tiers        []TierOption `json:"tiers"`
}

// TierOption is one tier an offer allows, with the toggles allowed at it and,
// as a detail, the models it runs on.
type TierOption struct {
	Tier    Tier                 `json:"tier"`
	Label   string               `json:"label"`
	Toggles Toggles              `json:"toggles"`
	Models  map[ModelType]string `json:"models"`
}

// Options lists what the player may choose from.
func (s *Session) Options() Options {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := Options{Choosing: s.settled == nil, Offers: []OfferOption{}, Settled: s.settledView()}
	offers := s.offers
	for _, i := range runnableOffers(s.spec.Genre, offers, soundOffers(offers)) {
		offer := offers[i]
		entry, _ := catalogueEntry(offer.Platform)
		option := OfferOption{
			Offer:        offer.ID,
			Platform:     offer.Platform,
			Name:         entry.Name,
			Customizable: offer.Caveats.Customizable,
			Default:      normal(offer.Caveats.Default),
		}
		for _, tier := range offer.Caveats.tiers() {
			allowed, _ := offer.Caveats.allowed(tier)
			models := map[ModelType]string{}
			if set, err := entry.Models(tier); err == nil {
				for _, kind := range slices.Concat([]ModelType{ModelTooling, ModelStory}, modelTypes(allowed)) {
					if model, ok := set[kind]; ok {
						models[kind] = model.Name
					}
				}
			}
			option.Tiers = append(option.Tiers, TierOption{Tier: tier, Label: tier.Label(), Toggles: allowed, Models: models})
		}
		out.Offers = append(out.Offers, option)
	}
	return out
}

// Choose settles a session waiting for the player: an offer by its ID and, if
// the offer is customizable, a choice within its caveats. A choice outside them
// is refused. The game starts once the choice is accepted.
func (s *Session) Choose(offer string, choice *Choice) error {
	defer s.flush()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled != nil {
		return errors.New("the session has already settled")
	}
	settled, err := chosen(s.spec.Genre, s.offers, offer, choice)
	if err != nil {
		return err
	}
	s.start(settled, nil)
	return nil
}

// SetToggle switches a feature on or off during play. Off is always allowed,
// and stops new work while what was made stays; on is allowed when the session's
// offer allows it at the session's tier. The change becomes the preference.
func (s *Session) SetToggle(toggle Toggle, on bool) error {
	defer s.flush()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled == nil {
		return ErrChoosing
	}
	if !slices.Contains(adapters.AllToggles(), toggle) {
		return fmt.Errorf("unknown toggle %q", toggle)
	}
	if on {
		allowed, _ := s.settled.offer.Caveats.allowed(s.settled.Tier)
		if !allowed.Has(toggle) {
			return fmt.Errorf("the offer does not allow %s at %s", toggle, s.settled.Tier)
		}
	}
	s.settled.Toggles = s.settled.Toggles.With(toggle, on)
	pref := *s.spec.Preference
	pref.Toggles = pref.Toggles.With(toggle, on)
	s.spec.Preference = &pref
	s.applySwitches()
	// A switch the player made explains nothing new; why the offers narrowed still holds.
	s.announce(s.changes)
	return nil
}

// Adapt takes the platform's current offers, as a share changed while the game
// runs. Offers with errors are left out. The stored preference is fitted to the
// rest: a tier or toggle no longer allowed gives way, and the player is told
// why. A session never changes platform: if none of the offers runs on it, or
// what remains no longer meets the genre's needs, the session ends with an
// error.
func (s *Session) Adapt(offers []Offer) error {
	err := s.adapt(offers)
	s.flush()
	if err != nil {
		s.endAfterDelivery()
	}
	return err
}

// adapt does Adapt's work under the lock; a hard failure is returned for Adapt
// to end the session once the error has been delivered.
func (s *Session) adapt(offers []Offer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.offers = offers
	if s.settled == nil {
		return nil
	}

	settled, changes, fail := fit(*s.spec.Preference, offers, soundOffers(offers), s.spec.Genre)
	if fail != "" {
		s.post(Event{Stream: "error", Value: "the game cannot continue: " + fail})
		return errors.New(fail)
	}
	s.settled = &settled
	s.equip()
	s.applySwitches()
	s.announce(changes)
	return nil
}

// start settles the session and runs the game. The preference is the player's
// choice unless the spec already carried one, which stays the wish. Called with
// mu held, once.
func (s *Session) start(settled settlement, changes []string) {
	s.settled = &settled
	if s.spec.Preference == nil {
		s.spec.Preference = &Preference{Platform: settled.offer.Platform, Offer: settled.offer.ID, Choice: settled.Choice}
	}
	s.usage.price(settled.offer.Platform)
	s.equip()
	s.applySwitches()
	s.announce(changes)
	s.wiring.Graph.Start(s.ctx)
}

// equip puts the settled offer's adapters in the slots the blocks call.
func (s *Session) equip() {
	live, tool, image, err := s.spec.adapters(*s.settled)
	if err != nil {
		s.post(Event{Stream: "error", Value: err.Error()})
		return
	}
	s.slots.live.Set(live)
	s.slots.tool.Set(tool)
	s.slots.image.Set(image)
}

// applySwitches sets every switchable block from the settled toggles.
func (s *Session) applySwitches() {
	for toggle, blocks := range s.wiring.Switches {
		for _, block := range blocks {
			block.SetActive(s.settled.Toggles.Has(toggle))
		}
	}
}

// announce sends what the session settled on, so the player can show it and
// the platform can store the preference.
func (s *Session) announce(changes []string) {
	s.changes = changes
	view := s.settledView()
	blob, err := json.Marshal(view)
	if err != nil {
		return
	}
	s.post(Event{Stream: "settled", Value: string(blob)})
}

// settledView is the settlement on the wire, or nil while choosing. Called with mu held.
func (s *Session) settledView() *Settled {
	if s.settled == nil {
		return nil
	}
	return &Settled{
		Offer:      s.settled.offer.ID,
		Platform:   s.settled.offer.Platform,
		Choice:     s.settled.Choice,
		Preference: *s.spec.Preference,
		Changes:    s.changes,
	}
}

// post queues an event while mu is held; flush hands it to the sender once mu
// is released.
func (s *Session) post(ev Event) { s.pending = append(s.pending, ev) }

// flush hands the queued events to the sender, in order. It never waits on
// the stream: a send can wait on a full stream nobody reads, and neither the
// session nor the caller that changed it may wait with it. Called without mu.
func (s *Session) flush() {
	s.mu.Lock()
	pending := s.pending
	s.pending = nil
	s.mu.Unlock()
	if len(pending) == 0 {
		return
	}
	s.out.mu.Lock()
	s.out.queue = append(s.out.queue, pending...)
	s.out.mu.Unlock()
	s.out.wake()
}

// endAfterDelivery ends the session once everything queued has been delivered,
// so a hard failure's error reaches the player before the session goes.
func (s *Session) endAfterDelivery() {
	s.out.mu.Lock()
	s.out.ending = true
	s.out.mu.Unlock()
	s.out.wake()
}

// outbox holds events for the sender, which delivers them in order.
type outbox struct {
	mu     sync.Mutex
	queue  []Event
	ending bool
	ready  chan struct{}
}

func (o *outbox) wake() {
	select {
	case o.ready <- struct{}{}:
	default:
	}
}

// deliver is the sender: it emits queued events in order until the session ends.
func (s *Session) deliver() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.out.ready:
		}
		for {
			s.out.mu.Lock()
			queue, ending := s.out.queue, s.out.ending
			s.out.queue = nil
			s.out.mu.Unlock()
			if len(queue) == 0 {
				if ending {
					s.once.Do(s.cancel)
					return
				}
				break
			}
			for _, ev := range queue {
				s.emit(ev)
			}
		}
	}
}

// emit puts one event on the stream and through persistence. Only the sender calls it.
func (s *Session) emit(ev Event) {
	_ = s.spec.Persist.Save(s.ctx, s.spec.ID, ev)
	select {
	case s.events <- ev:
	case <-s.ctx.Done():
	}
}

// playing refuses play before the player has chosen.
func (s *Session) playing() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled == nil {
		return ErrChoosing
	}
	return nil
}

// Preference is the player's current wish, for whoever stores the session;
// nil while choosing.
func (s *Session) Preference() *Preference {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.spec.Preference == nil {
		return nil
	}
	pref := *s.spec.Preference
	return &pref
}

// modelTypes is the model types the toggles that are on add.
func modelTypes(t Toggles) []ModelType {
	var out []ModelType
	for _, toggle := range t.On() {
		out = append(out, toggle.ModelType())
	}
	return out
}
