package wordtoken

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestWordList(t *testing.T) {
	if len(words) != 2048 {
		t.Fatalf("want 2048 words, got %d", len(words))
	}
	valid := regexp.MustCompile(`^[a-z]{4,10}$`)
	prefixes := map[string]string{}
	for _, w := range words {
		if !valid.MatchString(w) {
			t.Errorf("invalid word %q", w)
		}
		if other, ok := prefixes[w[:4]]; ok {
			t.Errorf("prefix %q shared by %q and %q", w[:4], other, w)
		}
		prefixes[w[:4]] = w
	}
}

func TestGenerate(t *testing.T) {
	index := map[string]bool{}
	for _, w := range words {
		index[w] = true
	}
	for _, n := range []int{3, 4} {
		token, err := Generate(n)
		if err != nil {
			t.Fatal(err)
		}
		parts := strings.Split(token, "-")
		if len(parts) != n {
			t.Fatalf("want %d words, got %q", n, token)
		}
		for _, p := range parts {
			if !index[p] {
				t.Errorf("word %q not in list", p)
			}
		}
		if Normalize(token) != token {
			t.Errorf("generated token %q is not normalised", token)
		}
	}
}

// 40,960 draws over 2048 words: each word is expected 20 times. A modulo-biased
// or constant generator leaves words unseen or far over-represented.
func TestGenerateDistribution(t *testing.T) {
	counts := map[string]int{}
	for i := 0; i < 20*len(words); i++ {
		token, err := Generate(1)
		if err != nil {
			t.Fatal(err)
		}
		counts[token]++
	}
	if len(counts) < len(words)*99/100 {
		t.Errorf("only %d of %d words drawn", len(counts), len(words))
	}
	for w, c := range counts {
		if c > 60 {
			t.Errorf("word %q drawn %d times, expected ~20", w, c)
		}
	}
}

func TestGenerateUnique(t *testing.T) {
	ctx := context.Background()

	calls := 0
	token, err := GenerateUnique(ctx, 3, func(context.Context, string) (bool, error) {
		calls++
		return calls < 3, nil
	})
	if err != nil || calls != 3 || strings.Count(token, "-") != 2 {
		t.Errorf("retry: token=%q calls=%d err=%v", token, calls, err)
	}

	token, err = GenerateUnique(ctx, 3, func(_ context.Context, tok string) (bool, error) {
		return strings.Count(tok, "-") == 2, nil
	})
	if err != nil || strings.Count(token, "-") != 3 {
		t.Errorf("fallback to n+1: token=%q err=%v", token, err)
	}

	if _, err = GenerateUnique(ctx, 3, func(context.Context, string) (bool, error) {
		return true, nil
	}); err == nil {
		t.Error("want error when every token is taken")
	}

	boom := errors.New("boom")
	if _, err = GenerateUnique(ctx, 3, func(context.Context, string) (bool, error) {
		return false, boom
	}); !errors.Is(err, boom) {
		t.Errorf("want lookup error, got %v", err)
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"apfel-otter-turm":     "apfel-otter-turm",
		"  Apfel Otter  Turm ": "apfel-otter-turm",
		"apfel.otter_turm":     "apfel-otter-turm",
		"apfel--otter - turm":  "apfel-otter-turm",
		"Möwe Grün Straße":     "moewe-gruen-strasse",
		"APFEL\tOTTER\nTURM":   "apfel-otter-turm",
		"":                     "",
		"participant-abc-def":  "participant-abc-def",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
