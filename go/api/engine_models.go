// package: api / schemaf endpoints
// type:    logic
// job:     shows the admin the engine's models and model table, and lets them assign models, set entries and probe models
// limits:  the secure core decides and stores (core/engine_overrides.go); the engine checks and runs them
package api

import (
	"context"
	"slices"
	"time"

	"cgl/core"
	"cgl/session"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"engine"
)

// EngineModelsResp is every platform's tier assignments and model table, as
// the engine runs them and as its code sets them.
type EngineModelsResp struct {
	Platforms []EngineModelsPlatform `json:"platforms"`
}

// EngineModelsPlatform is one platform's models.
type EngineModelsPlatform struct {
	ID   engine.Platform `json:"id"`
	Name string          `json:"name"`
	// Configurable: the admin may assign its models and set their entries.
	Configurable bool                `json:"configurable"`
	Assignments  []ModelSlotView     `json:"assignments"`
	Models       []ModelTableRowView `json:"models"`
}

// ModelSlotView is the model a tier runs a task on.
type ModelSlotView struct {
	Tier    engine.Tier      `json:"tier"`
	Type    engine.ModelType `json:"type"`
	Model   string           `json:"model"`
	Default string           `json:"default"`
	// Assigned: the admin assigned the model; ModifiedAt says when.
	Assigned   bool       `json:"assigned"`
	ModifiedAt *time.Time `json:"modifiedAt,omitempty"`
}

// ModelTableRowView is a model's entry in the model table.
type ModelTableRowView struct {
	Model string            `json:"model"`
	Facts engine.ModelFacts `json:"facts"`
	// Default is the code's entry; nil for a model only the admin added.
	Default *engine.ModelFacts `json:"default,omitempty"`
	// Overridden: the admin set the entry and it applies. Stale: the admin
	// set it under another app version, so the code's figures apply again.
	Overridden bool       `json:"overridden"`
	Stale      bool       `json:"stale"`
	ModifiedAt *time.Time `json:"modifiedAt,omitempty"`
	// InUse: a tier runs a task on the model.
	InUse bool `json:"inUse"`
}

// GetEngineModelsEndpoint shows the admin the engine's models: per platform
// the model each tier runs each task on, and the model table, each as it
// runs and as the code sets it, with the admin's changes marked.
type GetEngineModelsEndpoint struct{}

func (GetEngineModelsEndpoint) Method() string             { return "GET" }
func (GetEngineModelsEndpoint) Path() string               { return "/api/system/engine-models" }
func (GetEngineModelsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (GetEngineModelsEndpoint) Handle(ctx context.Context, _ GetEngineModelsReq) (EngineModelsResp, error) {
	return engineModelsView(core.ReadEngineOverrides(ctx, RunningVersion()))
}

// GetEngineModelsReq takes no parameters.
type GetEngineModelsReq struct{}

// AssignEngineModelEndpoint assigns a model to a tier's task; admins only.
//
// The model must be in the platform's model table and fit the task: a
// hearing model takes audio, an image model makes pictures, a story or
// speech model has a context window (400 otherwise). An empty model gives
// the task back its default. Answers the engine's models as they now are.
type AssignEngineModelEndpoint struct{}

func (AssignEngineModelEndpoint) Method() string { return "PUT" }
func (AssignEngineModelEndpoint) Path() string   { return "/api/system/engine-models/assignment" }
func (AssignEngineModelEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (AssignEngineModelEndpoint) Handle(ctx context.Context, req AssignEngineModelReq) (EngineModelsResp, error) {
	slot := engine.ModelSlot{Platform: req.Platform, Tier: req.Tier, Type: req.Type}
	return engineModelsView(core.AssignModel(ctx, slot, req.Model, RunningVersion()))
}

// AssignEngineModelReq is the tier's task and the model; empty resets it.
type AssignEngineModelReq struct {
	Platform engine.Platform  `json:"platform"`
	Tier     engine.Tier      `json:"tier"`
	Type     engine.ModelType `json:"type"`
	Model    string           `json:"model"`
}

// SetEngineModelEntryEndpoint sets a model's entry in the model table, adding
// the model if the table lacks it; admins only.
//
// Null facts give a code model back the code's figures and remove a model
// the admin added. A model a tier runs a task on cannot leave the table or
// lose what the task needs (400). A change to a code model applies until the
// next app version. Answers the engine's models as they now are.
type SetEngineModelEntryEndpoint struct{}

func (SetEngineModelEntryEndpoint) Method() string { return "PUT" }
func (SetEngineModelEntryEndpoint) Path() string   { return "/api/system/engine-models/entry" }
func (SetEngineModelEntryEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (SetEngineModelEntryEndpoint) Handle(ctx context.Context, req SetEngineModelEntryReq) (EngineModelsResp, error) {
	return engineModelsView(core.SetModelEntry(ctx, req.Platform, req.Model, req.Facts, RunningVersion()))
}

// SetEngineModelEntryReq is the model and its entry; null facts reset it.
type SetEngineModelEntryReq struct {
	Platform engine.Platform            `json:"platform"`
	Model    string                     `json:"model"`
	Facts    *engine.ModelFactsOverride `json:"facts"`
}

// ProbeEngineModelEndpoint asks whether a model answers, with the smallest
// request its type takes; admins only.
//
// Paid by the admin's own payers on the platform (400 no_api_key without
// one). Text and image models only for now; any other type answers
// problem "no_probe". A model that does not answer is a result, with the
// provider's message.
type ProbeEngineModelEndpoint struct{}

func (ProbeEngineModelEndpoint) Method() string { return "POST" }
func (ProbeEngineModelEndpoint) Path() string   { return "/api/system/engine-models/probe" }
func (ProbeEngineModelEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (ProbeEngineModelEndpoint) Handle(ctx context.Context, req ProbeEngineModelReq) (ProbeEngineModelResp, error) {
	r, err := session.ProbeModel(ctx, req.Platform, req.Type, req.Model)
	return ProbeEngineModelResp{Answers: r.Answers, Problem: r.Problem, Message: r.Message, Cost: r.Cost, Priced: r.Priced}, err
}

// ProbeEngineModelReq is the model to probe and the task it would run.
type ProbeEngineModelReq struct {
	Platform engine.Platform  `json:"platform"`
	Type     engine.ModelType `json:"type"`
	Model    string           `json:"model"`
}

// ProbeEngineModelResp says whether the model answered, and what the probe cost.
type ProbeEngineModelResp struct {
	Answers bool    `json:"answers"`
	Problem string  `json:"problem,omitempty"`
	Message string  `json:"message,omitempty"`
	Cost    float64 `json:"cost"`
	Priced  bool    `json:"priced"`
}

// engineModelsView is the engine's models as the admin sees them, from the
// catalogue as it runs, as the code sets it, and the stored overrides.
func engineModelsView(stored core.EngineOverrides, err error) (EngineModelsResp, error) {
	if err != nil {
		return EngineModelsResp{}, err
	}
	defaults := map[engine.Platform]engine.PlatformInfo{}
	for _, p := range engine.DefaultPlatforms() {
		defaults[p.ID] = p
	}
	assigned := map[engine.ModelSlot]core.ModelAssignment{}
	for _, a := range stored.Assignments {
		assigned[a.Slot] = a
	}
	entries := map[engine.Platform]map[string]core.ModelEntry{}
	for _, e := range stored.Entries {
		if entries[e.Platform] == nil {
			entries[e.Platform] = map[string]core.ModelEntry{}
		}
		entries[e.Platform][e.Model] = e
	}

	out := EngineModelsResp{Platforms: []EngineModelsPlatform{}}
	for _, p := range engine.Platforms() {
		if p.ID == engine.PlatformMock {
			continue
		}
		code := defaults[p.ID]
		view := EngineModelsPlatform{ID: p.ID, Name: p.Name, Configurable: p.Configurable,
			Assignments: []ModelSlotView{}, Models: []ModelTableRowView{}}
		used := map[string]bool{}
		for _, tier := range p.Tiers {
			codeSet := modelSetOf(code, tier.Tier)
			for _, kind := range engine.ModelTypes() {
				model, listed := tier.Models[kind]
				if !listed {
					continue
				}
				used[model.Name] = true
				slot := ModelSlotView{Tier: tier.Tier, Type: kind, Model: model.Name, Default: codeSet[kind].Name}
				if a, ok := assigned[engine.ModelSlot{Platform: p.ID, Tier: tier.Tier, Type: kind}]; ok {
					slot.Assigned, slot.ModifiedAt = true, &a.ModifiedAt
				}
				view.Assignments = append(view.Assignments, slot)
			}
		}
		names := make([]string, 0, len(p.Models))
		for name := range p.Models {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			row := ModelTableRowView{Model: name, Facts: p.Models[name], InUse: used[name]}
			if def, ok := code.Models[name]; ok {
				row.Default = &def
			}
			if e, ok := entries[p.ID][name]; ok {
				row.Overridden, row.Stale, row.ModifiedAt = e.Applies, !e.Applies, &e.ModifiedAt
			}
			view.Models = append(view.Models, row)
		}
		out.Platforms = append(out.Platforms, view)
	}
	return out, nil
}

// modelSetOf is a platform's model set for a tier, empty if it lacks one.
func modelSetOf(p engine.PlatformInfo, tier engine.Tier) engine.ModelSet {
	for _, t := range p.Tiers {
		if t.Tier == tier {
			return t.Models
		}
	}
	return engine.ModelSet{}
}
