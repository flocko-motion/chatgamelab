package adapters

import (
	"context"
	"errors"
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

// An error the conversation sends is reported as every other call's error is,
// and the event still arrives.
func TestLiveSlotReportsConversationErrors(t *testing.T) {
	events := make(chan LiveEvent, 2)
	var slot LiveSlot
	var reported []error
	slot.Report = func(err error) { reported = append(reported, err) }
	slot.Set(fakeLive{conn: fakeConn{events: events}})

	conn, err := slot.Open(context.Background(), LiveConfig{})
	if err != nil {
		t.Fatal(err)
	}
	events <- LiveEvent{Kind: EventError, Err: ErrNoCredit}
	close(events)

	var got []LiveEvent
	for e := range conn.Events() {
		got = append(got, e)
	}
	if len(got) != 1 || got[0].Kind != EventError {
		t.Errorf("events = %+v, want the error passed on", got)
	}
	if len(reported) != 1 || !errors.Is(reported[0], ErrNoCredit) {
		t.Errorf("reported %v, want the conversation's error", reported)
	}
}

type fakeLive struct{ conn LiveConn }

func (f fakeLive) Open(context.Context, LiveConfig) (LiveConn, error) { return f.conn, nil }

type fakeConn struct{ events chan LiveEvent }

func (fakeConn) ID() string                             { return "fake" }
func (fakeConn) Answer() string                         { return "" }
func (fakeConn) Instruct(context.Context, string) error { return nil }
func (fakeConn) Inform(context.Context, string) error   { return nil }
func (fakeConn) SayAloud(context.Context, string) error { return nil }
func (c fakeConn) Events() <-chan LiveEvent             { return c.events }
func (fakeConn) Close() error                           { return nil }
