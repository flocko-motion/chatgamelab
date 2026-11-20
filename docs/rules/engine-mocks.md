---
id: engine-mocks
applies_to:
  - "go/engine/internal/adapters/**/*.go"
  - "go/engine/internal/blocks/**/*.go"
  - "go/engine/internal/genre/**/*.go"
trigger: "adds or changes an adapter interface, a block, or a genre's wiring"
---

# Every AI mechanism has a mock

The engine runs without any key: on the mock platform, a game plays from start
to end, in development and in tests, at no cost and deterministically.

- **Every adapter interface has a mock implementation** in
  `internal/adapters/mock`, arriving in the same change as the interface. It
  answers plausibly for the block that uses it (text a block can pass on, a
  picture, audio, a structured result that parses), bills invented usage from
  the mock's price table, and takes a `Pace` so tests can run it fast.
- **Every model type a tier lists has a mock model** in the mock platform's
  tiers.
- **Every genre plays through on the mock platform**: a wiring test drives it
  from its opening to a few turns, every output stream receiving something.

```go
// Bad: a new adapter interface, implemented for OpenAI only
type Speech interface { Speak(ctx context.Context, text string, voice string) ([]byte, Usage, error) }
// openai/speech.go … and nothing in mock/

// Good: the interface, its real implementations and its mock arrive together
// adapters.go: type Speech interface { … }
// openai/speech.go, mock/speech.go, mock/tiers.go lists a speech model
```
