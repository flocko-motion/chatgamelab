package blocks

import (
	"testing"
	"time"
)

// GPT-Live marks no end of utterance, so the engine derives one — and where it
// puts the boundary is what a player reads as a line of dialogue. Both halves
// of the rule are pinned here, because both have been got wrong: silence alone
// cuts mid-clause, and punctuation alone cuts at every full stop.
func TestSilenceEndsAnUtteranceOnlyAtASentence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pending string
		want    time.Duration
	}{
		{
			"mid-sentence waits, however long the pause",
			"Oui. Mais je",
			danglingGap,
		},
		{
			"a finished sentence ends on an ordinary pause",
			"You shall not pass.",
			pauseGap,
		},
		{
			"a long reply is cut sooner, so it arrives in readable pieces",
			"One. Two. Three. Four. Five.",
			groupedGap,
		},
		{
			"closing punctuation does not hide the full stop before it",
			`"Nobody crosses," he said…`,
			pauseGap,
		},
		{
			"an unfinished sentence after several finished ones still waits",
			"One. Two. Three. Four. Five. And then",
			danglingGap,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := silenceAfter(tc.pending); got != tc.want {
				t.Errorf("%q waits %v, want %v", tc.pending, got, tc.want)
			}
		})
	}
}
