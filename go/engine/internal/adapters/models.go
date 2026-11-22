package adapters

import "slices"

// Medium is a kind of content a model takes or makes.
type Medium string

const (
	MediumText  Medium = "text"
	MediumAudio Medium = "audio"
	MediumImage Medium = "image"
)

// ModelFacts is what the engine knows of one model: what it takes, what it
// makes, its price, read on a date from a source, as providers publish no
// prices through their APIs, and, for a model a story thread runs on, its
// context window, read the same way.
type ModelFacts struct {
	In    []Medium `json:"in"`
	Out   []Medium `json:"out"`
	Price Price    `json:"price"`
	// Read is the date the price was read, as YYYY-MM-DD.
	Read   string `json:"read"`
	Source string `json:"source"`
	// Window is the context window in tokens: the most a call's input and
	// reply may hold together. Zero is unknown, and a thread on the model
	// never compacts.
	Window int `json:"window,omitempty"`
	// WindowRead is the date the window was read, as YYYY-MM-DD, and
	// WindowSource where.
	WindowRead   string `json:"windowRead,omitempty"`
	WindowSource string `json:"windowSource,omitempty"`
}

// Windowed is the entry with the model's context window, read on a date from
// a source.
func (f ModelFacts) Windowed(tokens int, read, source string) ModelFacts {
	f.Window, f.WindowRead, f.WindowSource = tokens, read, source
	return f
}

// Takes reports whether the model takes a medium.
func (f ModelFacts) Takes(m Medium) bool { return slices.Contains(f.In, m) }

// Makes reports whether the model makes a medium.
func (f ModelFacts) Makes(m Medium) bool { return slices.Contains(f.Out, m) }

// ModelTable is a platform's models, keyed by model id: one entry for every
// model a tier names. The tiers choose the models; the table says what each
// model is.
type ModelTable map[string]ModelFacts

// Price reports a model's price, and whether the table knows the model, so a
// caller can say "unpriced" over a zero.
func (t ModelTable) Price(model string) (Price, bool) {
	facts, ok := t[model]
	return facts.Price, ok
}

// Window reports a model's context window in tokens, zero when the table
// does not know it.
func (t ModelTable) Window(model string) int { return t[model].Window }

// Clone is a copy sharing no slice with the table.
func (t ModelTable) Clone() ModelTable {
	out := make(ModelTable, len(t))
	for id, facts := range t {
		facts.In = slices.Clone(facts.In)
		facts.Out = slices.Clone(facts.Out)
		out[id] = facts
	}
	return out
}
