package adapters

import (
	"context"
	"testing"
)

// echoThread reports the continuation it was handed as the reply.
type echoThread struct{ model string }

func (e echoThread) Model() string { return e.model }

func (e echoThread) Continue(_ context.Context, turn ThreadTurn) (ThreadReply, error) {
	return ThreadReply{Text: turn.Continuation.ID, Continuation: Continuation{ID: "next", Model: e.model}}, nil
}

// A continuation is only used on the offer and model that made it: another
// key may be another account, and a Mistral conversation cannot change model.
func TestThreadSlotDropsAContinuationMadeElsewhere(t *testing.T) {
	var slot ThreadSlot
	slot.Set(echoThread{model: "m1"}, "offer-a")
	ctx := context.Background()

	for name, tc := range map[string]struct {
		from Continuation
		want string
	}{
		"same offer and model": {Continuation{ID: "c", Offer: "offer-a", Model: "m1"}, "c"},
		"another offer":        {Continuation{ID: "c", Offer: "offer-b", Model: "m1"}, ""},
		"another model":        {Continuation{ID: "c", Offer: "offer-a", Model: "m0"}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			reply, err := slot.Continue(ctx, ThreadTurn{Continuation: tc.from})
			if err != nil {
				t.Fatal(err)
			}
			if reply.Text != tc.want {
				t.Errorf("the thread was handed %q, want %q", reply.Text, tc.want)
			}
			if reply.Continuation.Offer != "offer-a" {
				t.Errorf("continuation = %+v, want it stamped with the offer", reply.Continuation)
			}
		})
	}
}

func TestThreadSlotUnset(t *testing.T) {
	var slot ThreadSlot
	if _, err := slot.Continue(context.Background(), ThreadTurn{}); err != ErrUnset {
		t.Fatalf("got %v, want ErrUnset", err)
	}
}
