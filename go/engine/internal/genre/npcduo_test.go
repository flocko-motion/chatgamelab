package genre

import (
	"testing"
	"time"

	"engine/internal/blocks"
	"engine/internal/ports"
)

// Each character's words carry its seat; what the first hears is the
// player's unless the second has just spoken; the second's ear is dropped.
func TestTheDuoFloorTellsWhoSpoke(t *testing.T) {
	floor := &duoFloor{spoke: map[string]time.Time{}}
	a, b := floor.ear(SeatFirst, SeatSecond, true), floor.ear(SeatSecond, SeatFirst, false)
	now := time.Now()
	at := func(speaker string, after time.Duration) ports.Fragment {
		return ports.Fragment{Speaker: speaker, Text: "x", At: now.Add(after)}
	}

	if f, keep := a(at(blocks.SpeakerCharacter, 0)); !keep || f.Speaker != SeatFirst {
		t.Errorf("the first's words = %+v, %t", f, keep)
	}
	if _, keep := b(at(blocks.SpeakerPlayer, 0)); keep {
		t.Error("the second's ear was kept; the player would be heard twice")
	}
	if f, keep := a(at(blocks.SpeakerPlayer, time.Second)); !keep || f.Speaker != blocks.SpeakerPlayer {
		t.Errorf("the player, with the second quiet = %+v, %t", f, keep)
	}
	if f, keep := b(at(blocks.SpeakerCharacter, 2*time.Second)); !keep || f.Speaker != SeatSecond {
		t.Errorf("the second's words = %+v, %t", f, keep)
	}
	if _, keep := a(at(blocks.SpeakerPlayer, 3*time.Second)); keep {
		t.Error("what the first heard just after the second spoke was taken for the player")
	}
	if _, keep := a(at(blocks.SpeakerPlayer, 2*time.Second+echoWindow+time.Second)); !keep {
		t.Error("the player, well after the second spoke, was dropped")
	}
}
