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
type LiveSlot struct{ held atomic.Pointer[Live] }

// Set replaces the implementation; the next conversation opens on it.
func (s *LiveSlot) Set(live Live) { s.held.Store(&live) }

func (s *LiveSlot) Open(ctx context.Context, cfg LiveConfig) (LiveConn, error) {
	held := s.held.Load()
	if held == nil || *held == nil {
		return nil, ErrUnset
	}
	return (*held).Open(ctx, cfg)
}

// ToolSlot is a Tool whose implementation can be replaced.
type ToolSlot struct{ held atomic.Pointer[Tool] }

// Set replaces the implementation; the next query runs on it.
func (s *ToolSlot) Set(tool Tool) { s.held.Store(&tool) }

func (s *ToolSlot) Query(ctx context.Context, system, user string) (string, Usage, error) {
	held := s.held.Load()
	if held == nil || *held == nil {
		return "", Usage{}, ErrUnset
	}
	return (*held).Query(ctx, system, user)
}

// ImageSlot is an Image whose implementation can be replaced.
type ImageSlot struct{ held atomic.Pointer[Image] }

// Set replaces the implementation; the next picture is made by it.
func (s *ImageSlot) Set(image Image) { s.held.Store(&image) }

func (s *ImageSlot) Generate(ctx context.Context, req ImageRequest) ([]byte, Usage, error) {
	held := s.held.Load()
	if held == nil || *held == nil {
		return nil, Usage{}, ErrUnset
	}
	return (*held).Generate(ctx, req)
}
