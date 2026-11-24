package engine

import (
	"fmt"

	"engine/internal/adapters"
)

// modelNeeds is what a model type's work needs its model to take and make:
// the hearing model takes audio, the speech model makes it, the image model
// makes pictures, and every model answers in text unless it only draws.
var modelNeeds = map[ModelType]struct{ in, out Medium }{
	ModelTooling: {adapters.MediumText, adapters.MediumText},
	ModelStory:   {adapters.MediumText, adapters.MediumText},
	ModelImage:   {adapters.MediumText, adapters.MediumImage},
	ModelSpeech:  {adapters.MediumText, adapters.MediumAudio},
	ModelHearing: {adapters.MediumAudio, adapters.MediumText},
	ModelLive:    {adapters.MediumAudio, adapters.MediumAudio},
}

// threadModelTypes are the model types a story thread runs on, which need a
// context window to compact by.
var threadModelTypes = []ModelType{ModelStory, ModelSpeech}

// catalogueProblems says what is wrong with a platform's tables, one line
// each, nothing when they hold together: every model a tier names has an
// entry in the model table, so it is priced; it takes and makes what its
// type's work needs; and a model a story thread runs on has a context window.
// The code's tables must pass it (a test), and so must the code's tables with
// an admin's overrides on them (SetOverrides).
func catalogueProblems(p adapters.Platform) []string {
	var problems []string
	for _, tier := range p.Tiers {
		for _, kind := range adapters.ModelTypes() {
			model, listed := tier.Models[kind]
			if !listed {
				continue
			}
			where := fmt.Sprintf("%s/%s/%s", p.ID, tier.Tier, kind)
			facts, known := p.Facts[model.Name]
			if !known {
				problems = append(problems, fmt.Sprintf("%s: %s is not in the model table", where, model.Name))
				continue
			}
			need := modelNeeds[kind]
			if !facts.Takes(need.in) || !facts.Makes(need.out) {
				problems = append(problems, fmt.Sprintf("%s: %s takes %v and makes %v, the work needs it to take %s and make %s",
					where, model.Name, facts.In, facts.Out, need.in, need.out))
			}
			for _, thread := range threadModelTypes {
				if kind == thread && facts.Window <= 0 {
					problems = append(problems, fmt.Sprintf("%s: %s has no context window", where, model.Name))
				}
			}
		}
	}
	return problems
}
