package core

import (
	"context"
	"encoding/json"
	"time"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/log"

	"engine"
)

// The admin replaces the models of the engine's configurable platforms and
// sets their entries in the model table, as providers retire models and change
// prices faster than releases come (#589). Both are site settings: the admin
// reads and edits them ({Read|Edit, SiteSettings, Everywhere}). The engine
// checks every change before it is stored (engine.CheckOverrides), so what is
// stored always holds together, and takes it at once (engine.SetOverrides).
//
// A change to a model the code's table has applies under the app version it
// was set under: a new version brings the code's figures back, as its
// developers update the table with each release. A model only the admin added
// has no figures of the code to come back to, and keeps its entry.

// ModelAssignment is what the admin assigned to a tier's task: a model, and
// for a picture a quality (empty: the slot's own).
type ModelAssignment struct {
	Slot       engine.ModelSlot
	Choice     engine.SlotChoice
	ModifiedAt time.Time
}

// ModelEntry is a model's entry in the model table as the admin set it.
type ModelEntry struct {
	Platform   engine.Platform
	Model      string
	Facts      engine.ModelFactsOverride
	AppVersion string // the app version it was set under
	// Applies is false for a change to a code model set under another app
	// version: the code's figures apply again.
	Applies    bool
	ModifiedAt time.Time
}

// EngineOverrides are the admin's assignments and model entries, as stored.
type EngineOverrides struct {
	Assignments []ModelAssignment
	Entries     []ModelEntry
}

// ReadEngineOverrides are the admin's overrides, read for the app version
// running: {Read, SiteSettings, Everywhere}, the admin.
func ReadEngineOverrides(ctx context.Context, version string) (EngineOverrides, error) {
	if !granted(ctx, rules.Read, rules.SiteSettings, rules.Everywhere) {
		return EngineOverrides{}, forbidden("only the admin reads the engine's models")
	}
	stored, _, err := loadEngineOverrides(ctx, store.Queries(), version)
	return stored, err
}

// AssignModel assigns a model to a tier's task, for a picture with a
// quality, or with an empty model gives the task back its default: {Edit,
// SiteSettings, Everywhere}, the admin. The model must be in the model table
// and fit the task (engine.CheckOverrides). It answers the overrides as they
// now are.
func AssignModel(ctx context.Context, slot engine.ModelSlot, choice engine.SlotChoice, version string) (EngineOverrides, error) {
	if !granted(ctx, rules.Edit, rules.SiteSettings, rules.Everywhere) {
		return EngineOverrides{}, forbidden("only the admin assigns the engine's models")
	}
	return changeEngineOverrides(ctx, version, func(o *engine.Overrides) {
		if choice.Model == "" {
			delete(o.Models, slot)
		} else {
			o.Models[slot] = choice
		}
	}, func(q *db.Queries) error {
		if choice.Model == "" {
			return q.V2DeleteEngineModelAssignment(ctx, db.V2DeleteEngineModelAssignmentParams{
				Platform: string(slot.Platform), Tier: string(slot.Tier), ModelType: string(slot.Type),
			})
		}
		return q.V2SetEngineModelAssignment(ctx, db.V2SetEngineModelAssignmentParams{
			Platform: string(slot.Platform), Tier: string(slot.Tier), ModelType: string(slot.Type),
			Model: choice.Model, Quality: choice.Quality, ModifiedBy: account(ctx),
		})
	})
}

// ResetModelAssignments gives every task of a platform back its default:
// {Edit, SiteSettings, Everywhere}, the admin. It answers the overrides as
// they now are.
func ResetModelAssignments(ctx context.Context, platform engine.Platform, version string) (EngineOverrides, error) {
	if !granted(ctx, rules.Edit, rules.SiteSettings, rules.Everywhere) {
		return EngineOverrides{}, forbidden("only the admin assigns the engine's models")
	}
	return changeEngineOverrides(ctx, version, func(o *engine.Overrides) {
		for slot := range o.Models {
			if slot.Platform == platform {
				delete(o.Models, slot)
			}
		}
	}, func(q *db.Queries) error {
		return q.V2DeleteEngineModelAssignments(ctx, string(platform))
	})
}

// ResetModelEntries gives every code model of a platform back the code's
// figures, and removes the models the admin added that no tier runs; one a
// tier runs keeps its entry: {Edit, SiteSettings, Everywhere}, the admin. It
// answers the overrides as they now are.
func ResetModelEntries(ctx context.Context, platform engine.Platform, version string) (EngineOverrides, error) {
	if !granted(ctx, rules.Edit, rules.SiteSettings, rules.Everywhere) {
		return EngineOverrides{}, forbidden("only the admin sets the engine's model table")
	}
	codeModels := map[string]bool{}
	for _, p := range engine.DefaultPlatforms() {
		if p.ID == platform {
			for name := range p.Models {
				codeModels[name] = true
			}
		}
	}
	var dropped []string
	return changeEngineOverrides(ctx, version, func(o *engine.Overrides) {
		inUse := map[string]bool{}
		for slot, choice := range o.Models {
			if slot.Platform == platform {
				inUse[choice.Model] = true
			}
		}
		for name := range o.Facts[platform] {
			if codeModels[name] || !inUse[name] {
				delete(o.Facts[platform], name)
				dropped = append(dropped, name)
			}
		}
	}, func(q *db.Queries) error {
		for _, name := range dropped {
			if err := q.V2DeleteEngineModelFacts(ctx, db.V2DeleteEngineModelFactsParams{Platform: string(platform), Model: name}); err != nil {
				return err
			}
		}
		return nil
	})
}

// SetModelEntry sets a model's entry in the model table, adding the model if
// the table lacks it, or with nil gives a code model back its code's figures
// and removes a model the admin added: {Edit, SiteSettings, Everywhere}, the
// admin. A model a tier is assigned cannot leave the table, nor lose what its
// task needs (engine.CheckOverrides). It answers the overrides as they now
// are.
func SetModelEntry(ctx context.Context, platform engine.Platform, model string, facts *engine.ModelFactsOverride, version string) (EngineOverrides, error) {
	if !granted(ctx, rules.Edit, rules.SiteSettings, rules.Everywhere) {
		return EngineOverrides{}, forbidden("only the admin sets the engine's model table")
	}
	if model == "" {
		return EngineOverrides{}, refuse("no model named")
	}
	var entry engine.ModelFactsOverride
	if facts != nil {
		entry = *facts
		entry.Since = time.Now().UTC().Format(time.DateOnly)
	}
	return changeEngineOverrides(ctx, version, func(o *engine.Overrides) {
		if facts == nil {
			delete(o.Facts[platform], model)
			return
		}
		if o.Facts[platform] == nil {
			o.Facts[platform] = map[string]engine.ModelFactsOverride{}
		}
		o.Facts[platform][model] = entry
	}, func(q *db.Queries) error {
		if facts == nil {
			return q.V2DeleteEngineModelFacts(ctx, db.V2DeleteEngineModelFactsParams{Platform: string(platform), Model: model})
		}
		raw, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return q.V2SetEngineModelFacts(ctx, db.V2SetEngineModelFactsParams{
			Platform: string(platform), Model: model, Facts: raw, AppVersion: version, ModifiedBy: account(ctx),
		})
	})
}

// MayProbeModels refuses asking a model whether it answers, unless the caller
// may change the engine's models: {Edit, SiteSettings, Everywhere}, the
// admin. The probe pays as the editor does, through the admin's own payers.
func MayProbeModels(ctx context.Context) error {
	if !granted(ctx, rules.Edit, rules.SiteSettings, rules.Everywhere) {
		return forbidden("only the admin probes the engine's models")
	}
	return nil
}

// ApplyEngineOverrides hands the stored overrides to the engine, for the app
// version running: a before-serve hook, the core's own job, without a caller.
// What a release no longer holds together with, such as an assignment of a
// model the code dropped, does not apply; it is logged, and the admin sees it
// on the settings page.
func ApplyEngineOverrides(ctx context.Context, version string) {
	_, overrides, err := loadEngineOverrides(ctx, store.Queries(), version)
	if err != nil {
		log.Warn("engine overrides unreadable, the engine runs on its defaults", "error", err)
		return
	}
	err = engine.SetOverrides(overrides)
	if err == nil {
		return
	}
	log.Warn("engine overrides do not hold together, model assignments left out", "error", err)
	overrides.Models = map[engine.ModelSlot]engine.SlotChoice{}
	if err := engine.SetOverrides(overrides); err != nil {
		log.Warn("engine model entries do not hold together, the engine runs on its defaults", "error", err)
		_ = engine.SetOverrides(engine.Overrides{})
	}
}

// changeEngineOverrides changes the stored overrides, all or none: change
// makes the engine's overrides as they will be, which the engine must take,
// and write stores the change. The engine then runs on them.
func changeEngineOverrides(ctx context.Context, version string, change func(*engine.Overrides), write func(*db.Queries) error) (EngineOverrides, error) {
	var next engine.Overrides
	err := store.InTx(ctx, func(q *db.Queries) error {
		_, current, err := loadEngineOverrides(ctx, q, version)
		if err != nil {
			return err
		}
		change(&current)
		if err := engine.CheckOverrides(current); err != nil {
			return refuse("%s", err.Error())
		}
		next = current
		return write(q)
	})
	if err != nil {
		return EngineOverrides{}, err
	}
	if err := engine.SetOverrides(next); err != nil {
		return EngineOverrides{}, err
	}
	log.Info("engine overrides changed", "by", account(ctx).UUID)
	stored, _, err := loadEngineOverrides(ctx, store.Queries(), version)
	return stored, err
}

// loadEngineOverrides reads the stored overrides, as stored and as the engine
// takes them: a code model's entry set under another app version is left out.
func loadEngineOverrides(ctx context.Context, q *db.Queries, version string) (EngineOverrides, engine.Overrides, error) {
	stored := EngineOverrides{}
	overrides := engine.Overrides{Models: map[engine.ModelSlot]engine.SlotChoice{}, Facts: map[engine.Platform]map[string]engine.ModelFactsOverride{}}

	assignments, err := q.V2EngineModelAssignments(ctx)
	if err != nil {
		return stored, overrides, err
	}
	for _, a := range assignments {
		slot := engine.ModelSlot{Platform: engine.Platform(a.Platform), Tier: engine.Tier(a.Tier), Type: engine.ModelType(a.ModelType)}
		choice := engine.SlotChoice{Model: a.Model, Quality: a.Quality}
		stored.Assignments = append(stored.Assignments, ModelAssignment{Slot: slot, Choice: choice, ModifiedAt: a.ModifiedAt})
		overrides.Models[slot] = choice
	}

	codeModels := map[engine.Platform]engine.ModelTable{}
	for _, p := range engine.DefaultPlatforms() {
		codeModels[p.ID] = p.Models
	}
	rows, err := q.V2EngineModelFacts(ctx)
	if err != nil {
		return stored, overrides, err
	}
	for _, r := range rows {
		var facts engine.ModelFactsOverride
		if err := json.Unmarshal(r.Facts, &facts); err != nil {
			return stored, overrides, err
		}
		platform := engine.Platform(r.Platform)
		_, inCode := codeModels[platform][r.Model]
		applies := !inCode || r.AppVersion == version
		stored.Entries = append(stored.Entries, ModelEntry{
			Platform: platform, Model: r.Model, Facts: facts, AppVersion: r.AppVersion, Applies: applies, ModifiedAt: r.ModifiedAt,
		})
		if !applies {
			continue
		}
		if overrides.Facts[platform] == nil {
			overrides.Facts[platform] = map[string]engine.ModelFactsOverride{}
		}
		overrides.Facts[platform][r.Model] = facts
	}
	return stored, overrides, nil
}
