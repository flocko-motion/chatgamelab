package engine

import "errors"

// Problem names why a session cannot start or go on, so the playback-engine
// can tell the player in its own words.
type Problem string

const (
	// ProblemNoKey: no key is available for the game.
	ProblemNoKey Problem = "no_key"
	// ProblemGenre: the available key cannot run the game's genre.
	ProblemGenre Problem = "genre"
	// ProblemNoPlays: the share has no plays left.
	ProblemNoPlays Problem = "no_plays"
	// ProblemKeyRejected: the provider refuses the key.
	ProblemKeyRejected Problem = "key_rejected"
	// ProblemUnreachable: the provider does not answer.
	ProblemUnreachable Problem = "unreachable"
	// ProblemStale: the session was stored by another engine version, or its
	// wiring changed, and can no longer be resumed.
	ProblemStale Problem = "stale"
	// ProblemUnknown: there is no such session.
	ProblemUnknown Problem = "unknown"
	// ProblemDenied: whoever asked may not play this.
	ProblemDenied Problem = "denied"
	// ProblemFailed: anything else.
	ProblemFailed Problem = "failed"
)

// Problems is every problem, the source of the generated TypeScript union.
func Problems() []Problem {
	return []Problem{
		ProblemNoKey, ProblemGenre, ProblemNoPlays, ProblemKeyRejected, ProblemUnreachable,
		ProblemStale, ProblemUnknown, ProblemDenied, ProblemFailed,
	}
}

// Refusal is an error that names its problem. It travels as JSON in a
// transport's error body and on the `problem` stream.
type Refusal struct {
	Problem Problem `json:"problem"`
	Message string  `json:"message"`
}

func (r *Refusal) Error() string {
	if r.Message == "" {
		return string(r.Problem)
	}
	return r.Message
}

// Is matches any refusal of the same problem, so errors.Is(err, ErrStale)
// holds whatever the message.
func (r *Refusal) Is(target error) bool {
	t, ok := target.(*Refusal)
	return ok && t.Problem == r.Problem
}

// Refuse makes a refusal; a platform returns one from its create function.
func Refuse(problem Problem, message string) error {
	return &Refusal{Problem: problem, Message: message}
}

// Sentinels for errors.Is; each matches every refusal of its problem.
var (
	ErrKeyRejected = &Refusal{Problem: ProblemKeyRejected}
	ErrUnreachable = &Refusal{Problem: ProblemUnreachable}
	ErrStale       = &Refusal{Problem: ProblemStale}
	ErrUnknown     = &Refusal{Problem: ProblemUnknown}
)

// RefusalOf reads an error as a refusal, ProblemFailed for any other error.
func RefusalOf(err error) *Refusal {
	var r *Refusal
	if errors.As(err, &r) {
		return &Refusal{Problem: r.Problem, Message: err.Error()}
	}
	return &Refusal{Problem: ProblemFailed, Message: err.Error()}
}
