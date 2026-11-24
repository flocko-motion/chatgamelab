package engine_test

import (
	"context"
	"slices"
	"testing"

	"engine"
)

// Adventure's dice are rolled once at launch and stored with the spec, so a
// resumed opening is the same one.
func TestDiceAreRolledOnceAndStored(t *testing.T) {
	store := &engine.MemoryPersist{}
	first := launchStored(t, store, engine.SessionSpec{Subgenre: engine.SubgenreAdventureRounds, ID: "dice"})
	first.Close()

	stored, err := store.LoadState(context.Background(), "dice")
	if err != nil {
		t.Fatal(err)
	}
	dice := stored.Spec.Dice
	if len(dice) != 5 {
		t.Fatalf("stored dice %v, want five", dice)
	}
	for _, value := range dice {
		if value < 1 || value > 100 {
			t.Errorf("a die rolled %d", value)
		}
	}

	resumed := resume(t, store, "dice")
	resumed.Close()
	again, _ := store.LoadState(context.Background(), "dice")
	if !slices.Equal(again.Spec.Dice, dice) {
		t.Errorf("a resume rolled again: %v, then %v", dice, again.Spec.Dice)
	}
}

// A constraint frequency changed during play is stored with the spec, so a
// resumed session keeps it; an unknown block or a frequency out of range is
// refused.
func TestConstraintFrequencyChangesDuringPlay(t *testing.T) {
	store := &engine.MemoryPersist{}
	s := launchStored(t, store, engine.SessionSpec{Subgenre: engine.SubgenreAdventureRounds, ID: "freq"})
	if err := s.SetConstraintFrequency("expand", 4); err != nil {
		t.Fatal(err)
	}
	for block, n := range map[string]int{"rephrase": 2, "expand": 0, "outline": 11} {
		if err := s.SetConstraintFrequency(block, n); err == nil {
			t.Errorf("%s at %d was accepted", block, n)
		}
	}
	s.Close()

	stored, err := store.LoadState(context.Background(), "freq")
	if err != nil {
		t.Fatal(err)
	}
	if got := stored.Spec.ConstraintFrequency; got["expand"] != 4 || len(got) != 1 {
		t.Errorf("stored frequencies %v, want expand at 4", got)
	}
	resume(t, store, "freq").Close()
}

// A spec's constraint frequencies and dice are checked before a launch.
func TestValidateChecksFrequenciesAndDice(t *testing.T) {
	for name, spec := range map[string]engine.SessionSpec{
		"unknown block":   {Subgenre: engine.SubgenreAdventureRounds, ConstraintFrequency: map[string]int{"rephrase": 2}},
		"out of range":    {Subgenre: engine.SubgenreAdventureRounds, ConstraintFrequency: map[string]int{"expand": 11}},
		"npc-live has no": {Subgenre: engine.SubgenreNPCLive, ConstraintFrequency: map[string]int{"expand": 2}},
		"a die past 100":  {Subgenre: engine.SubgenreAdventureRounds, Dice: []int{7, 101}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := engine.Launch(context.Background(), spec, mockOffers()); err == nil {
				t.Error("the spec was launched")
			}
		})
	}
	ok := engine.SessionSpec{Subgenre: engine.SubgenreAdventureRounds, ConstraintFrequency: map[string]int{"expand": 3, "outline": 1}, Dice: []int{1, 100}}
	s, err := engine.Launch(context.Background(), ok, mockOffers())
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
}
