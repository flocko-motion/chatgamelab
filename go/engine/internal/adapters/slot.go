package adapters

import (
	"context"
	"errors"
	"sync/atomic"
)

// The slots hold an adapter that can be set after the graph is built and
// replaced while it runs: a session's platform, key and tier are settled after
// its wiring exists, and a changed share replaces them mid-game.

// ErrUnset is what a slot answers before anything was put in it.
var ErrUnset = errors.New("adapters: the session has not settled on a platform yet")

// LiveSlot is a Live whose implementation can be replaced.
type LiveSlot struct {
	held atomic.Pointer[Live]
	// Report hears every error the implementation returns; set before use.
	Report func(error)
}

// Set replaces the implementation; the next conversation opens on it.
func (s *LiveSlot) Set(live Live) { s.held.Store(&live) }

func (s *LiveSlot) Open(ctx context.Context, cfg LiveConfig) (LiveConn, error) {
	held := s.held.Load()
	if held == nil || *held == nil {
		return nil, ErrUnset
	}
	conn, err := (*held).Open(ctx, cfg)
	report(s.Report, err)
	if err != nil || s.Report == nil {
		return conn, err
	}
	return reportingConn{LiveConn: conn, events: reportEvents(conn.Events(), s.Report)}, nil
}

// reportingConn is a conversation whose errors are reported as they arrive,
// as every other call's are: a key refused mid-conversation is told the same.
type reportingConn struct {
	LiveConn
	events <-chan LiveEvent
}

func (c reportingConn) Events() <-chan LiveEvent { return c.events }

// reportEvents passes every event on, reporting each error's on the way, and
// closes when the conversation's events do.
func reportEvents(in <-chan LiveEvent, to func(error)) <-chan LiveEvent {
	out := make(chan LiveEvent, 64)
	go func() {
		defer close(out)
		for e := range in {
			if e.Kind == EventError {
				report(to, e.Err)
			}
			out <- e
		}
	}()
	return out
}

// ToolSlot is a Tool whose implementation can be replaced.
type ToolSlot struct {
	held atomic.Pointer[Tool]
	// Report hears every error the implementation returns; set before use.
	Report func(error)
}

// Set replaces the implementation; the next query runs on it.
func (s *ToolSlot) Set(tool Tool) { s.held.Store(&tool) }

func (s *ToolSlot) Query(ctx context.Context, system, user string) (string, Usage, error) {
	held := s.held.Load()
	if held == nil || *held == nil {
		return "", Usage{}, ErrUnset
	}
	text, used, err := (*held).Query(ctx, system, user)
	report(s.Report, err)
	return text, used, err
}

// QuerySchema asks the held tool for JSON of a schema; a tool that answers
// only in text cannot.
func (s *ToolSlot) QuerySchema(ctx context.Context, system, user string, schema Schema) (string, Usage, error) {
	held := s.held.Load()
	if held == nil || *held == nil {
		return "", Usage{}, ErrUnset
	}
	structured, ok := (*held).(SchemaTool)
	if !ok {
		return "", Usage{}, errors.New("adapters: this platform's tool answers only in text")
	}
	text, used, err := structured.QuerySchema(ctx, system, user, schema)
	report(s.Report, err)
	return text, used, err
}

// ImageSlot is an Image whose implementation can be replaced.
type ImageSlot struct {
	held atomic.Pointer[Image]
	// Report hears every error the implementation returns; set before use.
	Report func(error)
}

// Set replaces the implementation; the next picture is made by it.
func (s *ImageSlot) Set(image Image) { s.held.Store(&image) }

func (s *ImageSlot) Generate(ctx context.Context, req ImageRequest) ([]byte, Usage, error) {
	held := s.held.Load()
	if held == nil || *held == nil {
		return nil, Usage{}, ErrUnset
	}
	picture, used, err := (*held).Generate(ctx, req)
	report(s.Report, err)
	return picture, used, err
}

func report(to func(error), err error) {
	if to != nil && err != nil {
		to(err)
	}
}

// ThreadSlot is a Thread whose implementation can be replaced. It holds the
// offer the implementation runs on, because a continuation made on another
// offer or model cannot be used: another key may be another provider account,
// and a Mistral conversation cannot change model.
type ThreadSlot struct {
	held atomic.Pointer[heldThread]
	// Report hears every error the implementation returns; set before use.
	Report func(error)
}

type heldThread struct {
	thread Thread
	offer  string
}

// Set replaces the implementation and names its offer; the next call runs on
// it, on a fresh provider thread if the offer or the model changed.
func (s *ThreadSlot) Set(thread Thread, offer string) {
	s.held.Store(&heldThread{thread: thread, offer: offer})
}

// Model is what the held implementation runs on, empty before it is set.
func (s *ThreadSlot) Model() string {
	held := s.held.Load()
	if held == nil || held.thread == nil {
		return ""
	}
	return held.thread.Model()
}

func (s *ThreadSlot) Continue(ctx context.Context, turn ThreadTurn) (ThreadReply, error) {
	held := s.held.Load()
	if held == nil || held.thread == nil {
		return ThreadReply{}, ErrUnset
	}
	if turn.Continuation.Offer != held.offer || turn.Continuation.Model != held.thread.Model() {
		turn.Continuation = Continuation{}
	}
	reply, err := held.thread.Continue(ctx, turn)
	report(s.Report, err)
	if err == nil {
		reply.Continuation.Offer = held.offer
	}
	return reply, err
}

// HearingSlot is a Hearing whose implementation can be replaced.
type HearingSlot struct {
	held atomic.Pointer[Hearing]
	// Report hears every error the implementation returns; set before use.
	Report func(error)
}

// Set replaces the implementation; the next clip is heard by it.
func (s *HearingSlot) Set(hearing Hearing) { s.held.Store(&hearing) }

func (s *HearingSlot) Hear(ctx context.Context, system string, clip Clip, schema Schema) (string, Usage, error) {
	held := s.held.Load()
	if held == nil || *held == nil {
		return "", Usage{}, ErrUnset
	}
	reply, used, err := (*held).Hear(ctx, system, clip, schema)
	report(s.Report, err)
	return reply, used, err
}
