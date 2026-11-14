package game

import (
	"testing"

	"cgl/obj"

	"github.com/google/uuid"
)

func TestShareAllowedForGame(t *testing.T) {
	creator := uuid.New()
	other := uuid.New()

	game := func(public bool) *obj.Game {
		return &obj.Game{Public: public, Meta: obj.Meta{CreatedBy: uuid.NullUUID{UUID: creator, Valid: true}}}
	}
	share := func(by *uuid.UUID) *obj.GameShare { return &obj.GameShare{CreatedBy: by} }

	cases := []struct {
		name string
		game *obj.Game
		by   *uuid.UUID
		want bool
	}{
		{"public game, link by other", game(true), &other, true},
		{"public game, link by creator", game(true), &creator, true},
		{"private game, link by creator", game(false), &creator, true},
		{"private game, link by other", game(false), &other, false},
		{"private game, link without creator", game(false), nil, true},
		{"private game without creator, link by other", &obj.Game{}, &other, false},
	}
	for _, c := range cases {
		if got := shareAllowedForGame(share(c.by), c.game); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
