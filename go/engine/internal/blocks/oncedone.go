package blocks

import "sync"

// firedMarker is what a block that has done its once-only work exports. The
// engine never interprets a block's state, so the value only has to be
// recognisable to the block.
const firedMarker = "fired"

// once is a block's memory of work it does once per session, such as a line
// sent or a picture shown: kept across a restart, so a resumed session does
// not do it again. Embed it, call done and mark around the work, and the
// block is Resumable.
type once struct {
	// Guarded because ExportState is called from the caller's goroutine while
	// the block's own loop is running: a checkpoint never stops the graph.
	mu    sync.Mutex
	fired bool
}

// done reports whether the work was done, in this session or before a restart.
func (o *once) done() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.fired
}

// mark records the work as done.
func (o *once) mark() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fired = true
}

// ExportState records that the work was done, not what it made: that is
// configuration, or kept elsewhere.
func (o *once) ExportState() string {
	if o.done() {
		return firedMarker
	}
	return ""
}

// RestoreState takes back the work as done before a restart.
func (o *once) RestoreState(s string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fired = s == firedMarker
}
