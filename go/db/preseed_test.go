package db

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"engine"
)

// Every demo game file must seed: it parses, has a name and a known genre.
func TestDevGameFilesParse(t *testing.T) {
	dir, err := checkoutPath(devGamesDir)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no demo games in %s", dir)
	}
	for _, path := range paths {
		file, err := readGameFile(path)
		if err != nil {
			t.Errorf("%s: %v", filepath.Base(path), err)
			continue
		}
		if file.Name == "" {
			t.Errorf("%s: no name", filepath.Base(path))
		}
		if file.Genre != "" && !slices.Contains(engine.Genres(), engine.Genre(file.Genre)) {
			t.Errorf("%s: unknown genre %q", filepath.Base(path), file.Genre)
		}
	}
}

// Half of the demo games are public, and every dev user owns public and
// private ones, so every list and filter has something to show.
func TestDevGamesHalfPublic(t *testing.T) {
	dir, err := checkoutPath(devGamesDir)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	slots := devUserSlots()
	public, owned := 0, map[int][2]int{} // slot -> [private, public]
	for i := range paths {
		counts := owned[i%len(slots)]
		if devGamePublic(i) {
			public++
			counts[1]++
		} else {
			counts[0]++
		}
		owned[i%len(slots)] = counts
	}
	if half := len(paths) / 2; public < half || public > half+1 {
		t.Errorf("%d of %d demo games are public, want half", public, len(paths))
	}
	for i, slot := range slots {
		if c := owned[i]; c[0] == 0 || c[1] == 0 {
			t.Errorf("%s owns %d private and %d public demo games, want both", slot.name, c[0], c[1])
		}
	}
}

// Demo games have a history to sort by: created within the last year, changed
// at or after that, played a varying number of times, public ones more. The
// same file always gets the same history.
func TestDevGameHistory(t *testing.T) {
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	plays := map[int32]bool{}
	var publicPlays, privatePlays int32
	for i := range 30 {
		name := fmt.Sprintf("game-%d.yaml", i)
		h := devGameHistory(name, devGamePublic(i), now)
		if h != devGameHistory(name, devGamePublic(i), now) {
			t.Fatalf("%s: history differs between runs", name)
		}
		if h.createdAt.Before(now.AddDate(-1, 0, 0)) || h.createdAt.After(now) {
			t.Errorf("%s: created %v, want within the last year", name, h.createdAt)
		}
		if h.modifiedAt.Before(h.createdAt) || h.modifiedAt.After(now) {
			t.Errorf("%s: modified %v, want between creation %v and now", name, h.modifiedAt, h.createdAt)
		}
		if h.playCount < 0 {
			t.Errorf("%s: %d plays", name, h.playCount)
		}
		plays[h.playCount] = true
		if devGamePublic(i) {
			publicPlays += h.playCount
		} else {
			privatePlays += h.playCount
		}
	}
	if len(plays) < 10 {
		t.Errorf("only %d different play counts across 30 games, want them to vary", len(plays))
	}
	if publicPlays <= privatePlays {
		t.Errorf("public games played %d times, private %d: want public ones played more", publicPlays, privatePlays)
	}
}
