package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"engine/internal/adapters"
	"engine/internal/adapters/mistral"
	"engine/internal/adapters/openai"
	"engine/internal/ports"
)

// CheckOffer asks the offer's provider whether it accepts the key, with a
// request that costs nothing. It blocks, and answers nil, or a refusal:
// ErrKeyRejected for a key the provider refuses, ErrUnreachable for a provider
// that does not answer, ProblemNoKey for a key that cannot be had. The mock
// always passes.
func CheckOffer(ctx context.Context, offer Offer) error {
	if offer.Platform == PlatformMock {
		return nil
	}
	if offer.Keys == nil {
		return &Refusal{Problem: ProblemNoKey, Message: fmt.Sprintf("offer %q has no key", offer.ID)}
	}
	keys := adapters.KeyFunc(offer.Keys)
	var err error
	switch offer.Platform {
	case PlatformOpenAI:
		err = openai.Check(ctx, keys)
	case PlatformMistral:
		err = mistral.Check(ctx, keys)
	default:
		return fmt.Errorf("unknown platform %q", offer.Platform)
	}
	switch {
	case err == nil:
		return nil
	case errors.Is(err, adapters.ErrKeyRejected):
		return &Refusal{Problem: ProblemKeyRejected, Message: fmt.Sprintf("%s rejected the key: %v", offer.Platform, err)}
	case errors.Is(err, adapters.ErrUnreachable):
		return &Refusal{Problem: ProblemUnreachable, Message: fmt.Sprintf("%s did not answer: %v", offer.Platform, err)}
	default:
		return &Refusal{Problem: ProblemNoKey, Message: err.Error()}
	}
}

// watchKeys tells the player when the provider rejects the key during play,
// which says something other than a lost connection.
func (s *Session) watchKeys() {
	report := func(err error) {
		if !errors.Is(err, adapters.ErrKeyRejected) {
			return
		}
		s.problem(&Refusal{Problem: ProblemKeyRejected, Message: err.Error()})
	}
	s.slots.live.Report = report
	s.slots.tool.Report = report
	s.slots.image.Report = report
	s.slots.thread.Report = report
	s.slots.hearing.Report = report
}

// failed ends a game a problem sink was reached in, telling the player why,
// by the failed call's reason. A rejected key the slots have already told is
// not told twice.
func (s *Session) failed(value string) {
	var failure ports.Failure
	_ = json.Unmarshal([]byte(value), &failure)
	problem := problemOf(failure.Reason)
	s.ended.Do(func() {
		s.mu.Lock()
		told := problem == ProblemKeyRejected && s.toldRejected
		s.mu.Unlock()
		if !told {
			s.problem(&Refusal{Problem: problem, Message: fmt.Sprintf("%s: %s", failure.Node, failure.Message)})
		}
		s.endAfterDelivery()
	})
}

// problemOf names the problem a block's failure reason is.
func problemOf(reason string) Problem {
	switch reason {
	case ports.ReasonKeyRejected:
		return ProblemKeyRejected
	case ports.ReasonUnreachable:
		return ProblemUnreachable
	}
	return ProblemFailed
}

// problem puts a refusal on the `problem` stream.
func (s *Session) problem(r *Refusal) {
	blob, err := json.Marshal(r)
	if err != nil {
		return
	}
	s.mu.Lock()
	if r.Problem == ProblemKeyRejected {
		s.toldRejected = true
	}
	s.post(SessionEvent{Stream: "problem", Value: string(blob)})
	s.mu.Unlock()
	s.flush()
}
