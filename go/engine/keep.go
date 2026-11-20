package engine

import (
	"context"
	"time"
)

// checkpointDelay gathers the changes of a streaming turn into one write.
const checkpointDelay = 250 * time.Millisecond

// saveTimeout bounds the last checkpoint, written as the session ends.
const saveTimeout = 5 * time.Second

// log records an event in the history and stores it, in one order for both.
func (s *Session) log(ev SessionEvent) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	if !s.recorder.record(ev) {
		return
	}
	_ = s.spec.Persist.SaveEvent(s.ctx, s.spec.ID, ev)
	s.checkpoint()
}

// checkpoint asks for the session's state to be saved soon.
func (s *Session) checkpoint() {
	select {
	case s.dirty <- struct{}{}:
	default:
	}
}

// keep writes checkpoints until the session ends, and a last one then.
func (s *Session) keep() {
	defer close(s.kept)
	for {
		select {
		case <-s.ctx.Done():
			_ = s.saveState()
			return
		case <-s.dirty:
		}
		select {
		case <-s.ctx.Done():
		case <-time.After(checkpointDelay):
		}
		_ = s.saveState()
	}
}

// saveState stores the spec with the current preference, and the blocks'
// state. It outlives the session's context, so the last checkpoint is written.
func (s *Session) saveState() error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(s.ctx), saveTimeout)
	defer cancel()
	return s.spec.Persist.SaveState(ctx, s.spec.ID, s.stored())
}

// stored is what Resume needs back.
func (s *Session) stored() Stored {
	s.mu.Lock()
	spec := s.spec
	if spec.Preference != nil {
		pref := *spec.Preference
		spec.Preference = &pref
	}
	s.mu.Unlock()
	return Stored{
		Version: SchemaVersion,
		Spec:    spec,
		State: SessionState{
			Blocks: s.wiring.Graph.ExportState(),
			Usage:  s.usage.report().ByNode,
		},
	}
}
