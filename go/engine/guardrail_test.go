package engine_test

import (
	"context"
	"slices"
	"testing"

	"engine"
)

// A guardrail edited during play is reported with its provenance and the
// one the platform resolved, stored with the spec, and kept by a resumed
// session; setting it back to the resolved one clears the edit.
func TestGuardrailEditedDuringPlay(t *testing.T) {
	store := &engine.MemoryPersist{}
	s := launchStored(t, store, engine.SessionSpec{
		Genre: engine.GenreAdventure, ID: "guard", Guardrail: "No violence.",
		GuardrailSource: &engine.GuardrailSource{Level: "workshop", Name: "Klasse 7b"},
	})
	waitForTurn(t, s, "[story turn 2]")

	report := s.Guardrail()
	if report.Text != "No violence." || report.Edited || report.Source == nil || report.Source.Name != "Klasse 7b" {
		t.Errorf("before an edit the session reports %+v", report)
	}
	if !slices.Contains(report.ReadBy, "expand") || !slices.Contains(report.ReadBy, "outline") {
		t.Errorf("the guardrail is read by %v, want the story thread's blocks", report.ReadBy)
	}
	if len(report.Frequencies) != 2 {
		t.Errorf("frequencies %+v, want outline's and expand's", report.Frequencies)
	}

	if err := s.SetGuardrail("Gentle fights are fine."); err != nil {
		t.Fatal(err)
	}
	report = s.Guardrail()
	if report.Text != "Gentle fights are fine." || report.Resolved != "No violence." || !report.Edited {
		t.Errorf("after an edit the session reports %+v", report)
	}
	s.Close()

	stored, err := store.LoadState(context.Background(), "guard")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Spec.GuardrailEdit == nil || *stored.Spec.GuardrailEdit != "Gentle fights are fine." || stored.Spec.Guardrail != "No violence." {
		t.Errorf("stored guardrail %q, edit %v", stored.Spec.Guardrail, stored.Spec.GuardrailEdit)
	}

	resumed := resume(t, store, "guard")
	if got := resumed.Guardrail(); got.Text != "Gentle fights are fine." || !got.Edited {
		t.Errorf("the resumed session reports %+v", got)
	}
	if err := resumed.SetGuardrail("No violence."); err != nil {
		t.Fatal(err)
	}
	if got := resumed.Guardrail(); got.Edited {
		t.Errorf("set back to the resolved one, the session still reports an edit: %+v", got)
	}
}
