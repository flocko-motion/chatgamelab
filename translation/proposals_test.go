package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The checklist reads back what was written, ticks included, also for texts
// with quotes, backticks and line breaks.
func TestProposalsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.MkdirAll(filepath.Join(dir, "translation"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	in := []proposal{
		{Lang: "fr", Path: "common.save", Now: "Sauver", Proposed: "Enregistrer", Reason: "\"Sauver\" means rescue;\n the UI verb is Enregistrer."},
		{Lang: "de", Path: "x.y", Now: "a `b`", Proposed: "Zeile 1\nZeile 2 \"c\"", Reason: "keeps the line break", Ticked: true},
	}
	writeProposals(in)
	got, err := readProposals()
	if err != nil {
		t.Fatal(err)
	}
	in[0].Reason = "\"Sauver\" means rescue; the UI verb is Enregistrer." // one line in the list
	want := []proposal{in[1], in[0]}                                      // by language
	if !reflect.DeepEqual(got, want) {
		t.Errorf("read back\n%+v\nwant\n%+v", got, want)
	}

	addProposals([]proposal{in[0], {Lang: "fr", Path: "common.cancel", Now: "Annul", Proposed: "Annuler", Reason: "typo"}})
	got, _ = readProposals()
	if len(got) != 3 || !got[0].Ticked {
		t.Errorf("adding keeps the list and its ticks, without doubles: %+v", got)
	}
}
