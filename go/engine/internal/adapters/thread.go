package adapters

import (
	"context"
	"time"
)

// Exchange is one call on a thread: what was sent and the whole reply. A
// thread's transcript is its exchanges, oldest first.
type Exchange struct {
	// Block is the block that made the call, so each block counts its own.
	Block string `json:"block,omitempty"`
	Input string `json:"input"`
	// Directive says the input is the game's own message, not the player's.
	Directive bool `json:"directive,omitempty"`
	// Constraints are what the call carried of the youth-protection rule,
	// worded; empty when the call carried none.
	Constraints string `json:"constraints,omitempty"`
	Reply       string `json:"reply"`
}

// Continuation is what continuing a provider's thread needs back: the
// provider's handle, and what it was made on, since a handle is only good on
// the offer and model that made it.
type Continuation struct {
	// ID is the provider's handle: OpenAI's response ID, Mistral's
	// conversation ID. Empty starts a fresh thread.
	ID string `json:"id,omitempty"`
	// Offer is the offer's ID; another key may be another provider account.
	Offer string `json:"offer,omitempty"`
	Model string `json:"model,omitempty"`
	// Made is when the provider stored it, for a provider that forgets.
	Made time.Time `json:"made,omitzero"`
}

// ThreadTurn is one call that continues a thread.
type ThreadTurn struct {
	// System is the thread's System Message. It is sent once, at the start of
	// a provider thread, a fresh one seeded from the transcript included; a
	// continued thread already holds it.
	System string
	Input  string
	// Directive sends the input as the game's own message: a developer
	// message on OpenAI. Mistral has no such role, and sends it as the user's.
	Directive bool
	// Constraints are the youth-protection rule, worded, for this call; empty
	// sends none. They never share a message with designer text where the
	// provider has a message of their own for them.
	Constraints string
	// Schema makes the turn structured: the reply is JSON of this shape. Nil
	// is a plain turn, its reply free text.
	Schema *Schema
	// Speak makes a plain turn spoken: the model says its reply, the
	// transcript streams to Piece and the audio to Audio, and the reply's
	// Text is the transcript, so the transcript stays text. Valid only
	// without a Schema. A thread on a model that cannot speak answers a
	// spoken turn as a plain one.
	Speak        bool
	Continuation Continuation
	// Transcript seeds a fresh provider thread when the continuation cannot be
	// used, so a session never ends because a provider forgot its thread.
	Transcript []Exchange
	// Piece hears each piece of the reply as it arrives; nil hears nothing.
	Piece func(string)
	// Audio hears each chunk of a spoken reply as it arrives, 24 kHz mono
	// PCM16; nil hears nothing.
	Audio func([]byte)
}

// Exchange is the turn as its transcript keeps it, once the reply is in.
func (t ThreadTurn) Exchange(block, reply string) Exchange {
	return Exchange{Block: block, Input: t.Input, Directive: t.Directive, Constraints: t.Constraints, Reply: reply}
}

// ThreadReply is a finished call: the whole reply, what it cost, and the
// continuation the next call goes on from.
type ThreadReply struct {
	Text         string
	Usage        Usage
	Continuation Continuation
}

// Thread continues conversations held by the provider.
type Thread interface {
	// Model is what the thread runs on, which a continuation must match.
	Model() string
	// Continue makes one call. A failure leaves the thread as it was, so the
	// same turn may be made again.
	Continue(ctx context.Context, turn ThreadTurn) (ThreadReply, error)
}

// HearAudio hands a chunk of spoken reply to the turn's audio listener, if
// it has one.
func (t ThreadTurn) HearAudio(chunk []byte) {
	if t.Audio != nil && len(chunk) > 0 {
		t.Audio(chunk)
	}
}

// Hear hands a piece to the turn's listener, if it has one.
func (t ThreadTurn) Hear(piece string) {
	if t.Piece != nil && piece != "" {
		t.Piece(piece)
	}
}
