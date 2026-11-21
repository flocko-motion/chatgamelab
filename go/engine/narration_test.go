package engine

import (
	"strings"
	"testing"
	"time"
)

// Adventure plays on the mock with narration on, off, and switched mid-game.
// On, Expand's turns are spoken on the speaking story model and every output
// stream receives something; off, they are silent on the text story model.
func TestNarrationFollowsItsToggle(t *testing.T) {
	t.Run("on", func(t *testing.T) {
		g := adventure(mockAdventureOffer(false))
		g.spec.Preference = storedPreference(TierMedium, both)
		s := launchOrFail(t, g)

		say(t, s, "I cross the bridge")
		seen := streams(t, s, "text", "audio", "image", "props")
		if !seen["audio"] {
			t.Error("narration on, the turn was not heard")
		}
		eventually(t, "spoken audio billed on mock-realtime", func() bool {
			return spentOn(s, "expand", "mock-realtime").AudioOutputTokens > 0
		})
	})

	t.Run("off", func(t *testing.T) {
		s := launchOrFail(t, adventure(mockAdventureOffer(false)))

		say(t, s, "I cross the bridge")
		if seen := streams(t, s, "text", "image", "props"); seen["audio"] {
			t.Error("narration off, the turn was heard")
		}
		eventually(t, "the prose billed on mock-story", func() bool { return spentOn(s, "expand", "mock-story").OutputTokens > 0 })
		if got := spentOn(s, "expand", "mock-story"); got.AudioOutputTokens != 0 {
			t.Errorf("expand spent %+v on mock-story, want silent prose", got)
		}
		if got := spentOn(s, "expand", "mock-realtime"); got != (UsageRecord{}) {
			t.Errorf("expand spent %+v on the speaking model with narration off", got)
		}
	})

	t.Run("switched mid-game", func(t *testing.T) {
		s := launchOrFail(t, adventure(mockAdventureOffer(false)))

		say(t, s, "I cross the bridge")
		if seen := streams(t, s, "text", "image", "props"); seen["audio"] {
			t.Error("narration off, the first turn was heard")
		}
		eventually(t, "the first turn billed", func() bool { return spentOn(s, "expand", "mock-story").OutputTokens > 0 })
		firstTurn := spentOn(s, "expand", "mock-story").OutputTokens

		if err := s.SetToggle(ToggleAudioOut, true); err != nil {
			t.Fatal(err)
		}
		drain(s)
		say(t, s, "I look back")
		if seen := streams(t, s, "audio", "text"); !seen["audio"] {
			t.Error("narration switched on, the next turn was not heard")
		}
		// The thread went on to the speaking model, seeded with the story.
		eventually(t, "the second turn kept", func() bool { return hasTurn(s, "[story turn 4]") })
		eventually(t, "outline billed on mock-realtime, the whole thread switched", func() bool {
			return spentOn(s, "outline", "mock-realtime").OutputTokens > 0
		})

		if err := s.SetToggle(ToggleAudioOut, false); err != nil {
			t.Fatal(err)
		}
		drain(s)
		before := spentOn(s, "expand", "mock-realtime")
		say(t, s, "I walk on")
		if seen := streams(t, s, "text"); seen["audio"] {
			t.Error("narration switched off, the turn was still heard")
		}
		eventually(t, "the third turn kept", func() bool { return hasTurn(s, "[story turn 6]") })
		eventually(t, "the third turn billed on mock-story", func() bool {
			return spentOn(s, "expand", "mock-story").OutputTokens > firstTurn
		})
		if got := spentOn(s, "expand", "mock-realtime"); got != before {
			t.Errorf("expand went on spending on the speaking model: %+v, was %+v", got, before)
		}
		// Each model's record holds only what was spent on it.
		if got := spentOn(s, "expand", "mock-story"); got.AudioOutputTokens != 0 {
			t.Errorf("expand's mock-story record carries the speaking model's audio: %+v", got)
		}
	})
}

func say(t *testing.T, s *Session, line string) {
	t.Helper()
	if err := s.Say(line); err != nil {
		t.Fatal(err)
	}
}

// streams reads the session's events until Expand has finished a turn and
// each wanted stream has carried something, and reports every stream seen.
func streams(t *testing.T, s *Session, wanted ...string) map[string]bool {
	t.Helper()
	seen := map[string]bool{}
	working, finished := false, false
	deadline := time.After(5 * time.Second)
	for {
		select {
		case e := <-s.Events():
			seen[e.Stream] = true
			if e.Stream == "state" && strings.Contains(e.Value, `"node":"expand"`) {
				working = working || strings.Contains(e.Value, `"phase":"working"`)
				finished = finished || working && strings.Contains(e.Value, `"phase":"ready"`)
			}
		case <-deadline:
			t.Fatalf("the turn never finished with %v; saw %v", wanted, seen)
		}
		missing := !finished
		for _, stream := range wanted {
			missing = missing || !seen[stream]
		}
		if !missing {
			return seen
		}
	}
}

// drain reads what is left of the last turn's events.
func drain(s *Session) {
	for {
		select {
		case <-s.Events():
		case <-time.After(50 * time.Millisecond):
			return
		}
	}
}

// spentOn is what a block spent on one model.
func spentOn(s *Session, node, model string) UsageRecord {
	for _, record := range s.Usage().ByNode {
		if record.Node == node && record.Model == model {
			return record
		}
	}
	return UsageRecord{}
}

// hasTurn reports whether the history holds a whole reply of this turn.
func hasTurn(s *Session, prefix string) bool {
	for _, e := range s.History() {
		if e.Stream == "text" && strings.HasPrefix(e.Value, prefix) {
			return true
		}
	}
	return false
}
