package mock

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"engine/internal/adapters"
)

// threadPrefix starts every continuation the mock makes; the number after it
// is the turn the thread has reached.
const threadPrefix = "mock-thread-"

// Thread answers each call with its turn number, so a resumed thread is told
// apart from a fresh one: a continuation goes on counting, a fresh thread
// seeded with a transcript counts on from its exchanges, and a fresh thread
// starts at one. The reply streams word by word, spread over the Pace.
type Thread struct{ Pace Pace }

func (Thread) Model() string { return "mock-story" }

func (t Thread) Continue(ctx context.Context, turn adapters.ThreadTurn) (adapters.ThreadReply, error) {
	n := len(turn.Transcript) + 1
	if before, ok := strings.CutPrefix(turn.Continuation.ID, threadPrefix); ok {
		if reached, err := strconv.Atoi(before); err == nil {
			n = reached + 1
		}
	}

	text := fmt.Sprintf("[story turn %d] %s", n, turn.Input)
	words := strings.SplitAfter(text, " ")
	step := time.Duration(0)
	if t.Pace > 0 {
		step = t.Pace / time.Duration(len(words))
	}
	for _, word := range words {
		wait(ctx, step)
		if err := ctx.Err(); err != nil {
			return adapters.ThreadReply{}, err
		}
		turn.Hear(word)
	}

	seen := len(turn.Instructions) + len(turn.Input)
	return adapters.ThreadReply{
		Text: text,
		Usage: adapters.Usage{
			Model:        t.Model(),
			InputTokens:  int64(400 + seen),
			OutputTokens: int64(len(text)),
		},
		Continuation: adapters.Continuation{
			ID:    threadPrefix + strconv.Itoa(n),
			Model: t.Model(),
			Made:  time.Now().UTC(),
		},
	}, nil
}
