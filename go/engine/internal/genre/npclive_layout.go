package genre

import "engine/internal/ports"

// npcLiveLayout draws NPC Live top down: the portrait, the start, the
// conversation, then what reaches the player and what failed. Rearrange it in
// the dev server's pipeline view and paste the copied layout over this one.
var npcLiveLayout = ports.Layout{
	"portrait-prompt":    {Row: 0, Col: 0},
	"portrait":           {Row: 1, Col: 0},
	"start-game":         {Row: 2, Col: 1},
	"player-input-audio": {Row: 3, Col: 2},
	"live-session":       {Row: 4, Col: 1},
	"observer":           {Row: 4, Col: 2},
	"out-image":          {Row: 5, Col: 0},
	"out-text":           {Row: 5, Col: 1},
	"out-audio":          {Row: 5, Col: 2},
	"image-failed":       {Row: 6, Col: 0},
}
