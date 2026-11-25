package main

import (
	"reflect"
	"testing"
)

func TestPruneFieldsDropsWhatTheSourceLacks(t *testing.T) {
	src := `{"footer":{"help":"Help"},"list":["a","b"]}`
	s := `{"footer":{"help":"Hilfe","copyright":"JFF"},"old":{"x":"y"},"list":["A","B"]}`

	pruned, dropped, err := pruneFields(s, src)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"footer.copyright", "old.x"}; !reflect.DeepEqual(dropped, want) {
		t.Errorf("dropped = %v, want %v", dropped, want)
	}
	got, _ := leaves(pruned)
	want := map[string]string{"footer.help": "Hilfe", "list[0]": "A", "list[1]": "B"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pruned leaves = %v, want %v", got, want)
	}
}

func TestPruneFieldsKeepsAFileThatMatches(t *testing.T) {
	s := `{"a":"b"}`
	pruned, dropped, err := pruneFields(s, `{"a":"c"}`)
	if err != nil || len(dropped) != 0 || pruned != s {
		t.Errorf("pruneFields = %q, %v, %v; want the file unchanged", pruned, dropped, err)
	}
}

func TestCheckTextsFindsWhatIsWrong(t *testing.T) {
	want := map[string]string{"a": "Hello {{name}}", "b": "Bye", "c": "I agree to the"}
	texts := map[string]string{"a": "Bonjour", "c": "", "x": "extra"}
	got := checkTexts(want, texts)
	wantProblems := []string{
		"a: placeholders [], the English has [{{name}}]",
		"b: missing",
		"x: not asked for",
	}
	if !reflect.DeepEqual(got, wantProblems) {
		t.Errorf("problems = %q, want %q", got, wantProblems)
	}
	if p := checkTexts(want, map[string]string{"a": "Salut {{name}}", "b": "Au revoir", "c": "J'accepte"}); len(p) != 0 {
		t.Errorf("a right translation has problems: %q", p)
	}
}

func TestFillPathsSetsOnlyTheLeavesNamed(t *testing.T) {
	out, err := fillPaths(`{"x":{"a":"A","b":"B"}}`, map[string]string{"x.a": "neu"})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := leaves(out)
	if want := map[string]string{"x.a": "neu", "x.b": "B"}; !reflect.DeepEqual(got, want) {
		t.Errorf("leaves = %v, want %v", got, want)
	}
}

func TestCheckTextsKeepsTags(t *testing.T) {
	want := map[string]string{"a": "Let's <mark>build</mark>"}
	if p := checkTexts(want, map[string]string{"a": "On va cmarkconstruirc/mark"}); len(p) != 1 {
		t.Errorf("a lost tag passes: %q", p)
	}
	if p := checkTexts(want, map[string]string{"a": "On va <mark>construire</mark>"}); len(p) != 0 {
		t.Errorf("a kept tag fails: %q", p)
	}
}

func TestCheckTextsRefusesControlCharacters(t *testing.T) {
	want := map[string]string{"a": "Story & AI"}
	if p := checkTexts(want, map[string]string{"a": "Tarina \x7fp AI"}); len(p) != 1 {
		t.Errorf("a broken ampersand passes: %q", p)
	}
	if p := checkTexts(map[string]string{"a": "x\ny"}, map[string]string{"a": "x\ny"}); len(p) != 0 {
		t.Errorf("a line break fails: %q", p)
	}
}
