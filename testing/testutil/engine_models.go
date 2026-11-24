package testutil

import "cgl/apiclient"

// --- The engine's models, for the admin (#589) ---

// EngineModels is every platform's tier assignments and model table.
func (c caller) EngineModels() (EngineModels, error) {
	return decode[EngineModels](c.api.GetApiSystemEngineModelsWithResponse(ctx()))
}

// AssignEngineModel assigns a model to a tier's task on a platform; an empty
// model gives the task back its default.
func (c caller) AssignEngineModel(platform, tier, task, model string) (EngineModels, error) {
	return decode[EngineModels](c.api.PutApiSystemEngineModelsAssignmentWithResponse(ctx(), apiclient.AssignEngineModelReq{
		Platform: apiclient.Platform(platform), Tier: apiclient.Tier(tier), Type: apiclient.ModelType(task), Model: model,
	}))
}

// AddTextModel puts a text model into a platform's model table: it takes
// and makes text, at the prices given per million tokens, with a context
// window.
func (c caller) AddTextModel(platform, model string, inputPerMTok, outputPerMTok float32, window int) (EngineModels, error) {
	text := []apiclient.Medium{"text"}
	return c.setEngineModelEntry(platform, model, &apiclient.ModelFactsOverride{
		In: &text, Out: &text, Window: &window,
		Price: apiclient.Price{InputPerMTok: &inputPerMTok, OutputPerMTok: &outputPerMTok},
	})
}

// ResetEngineModelEntry gives a code model back the code's entry, and
// removes a model the admin added.
func (c caller) ResetEngineModelEntry(platform, model string) (EngineModels, error) {
	return c.setEngineModelEntry(platform, model, nil)
}

func (c caller) setEngineModelEntry(platform, model string, facts *apiclient.ModelFactsOverride) (EngineModels, error) {
	return decode[EngineModels](c.api.PutApiSystemEngineModelsEntryWithResponse(ctx(), apiclient.SetEngineModelEntryReq{
		Platform: apiclient.Platform(platform), Model: model, Facts: facts,
	}))
}

// AssignedModel is the model a platform's tier runs a task on, as the
// engine's models show it, and whether the admin assigned it.
func AssignedModel(m EngineModels, platform, tier, task string) (model string, assigned bool) {
	for _, p := range m.Platforms {
		if string(p.Id) != platform {
			continue
		}
		for _, a := range p.Assignments {
			if string(a.Tier) == tier && string(a.Type) == task {
				return a.Model, a.Assigned
			}
		}
	}
	return "", false
}

// ModelRow is a model's row in a platform's model table, if it has one.
func ModelRow(m EngineModels, platform, model string) (ModelTableRow, bool) {
	for _, p := range m.Platforms {
		if string(p.Id) != platform {
			continue
		}
		for _, r := range p.Models {
			if r.Model == model {
				return r, true
			}
		}
	}
	return ModelTableRow{}, false
}
