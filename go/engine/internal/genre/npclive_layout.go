package genre

import "engine/internal/ports"

// npcLiveLayout draws NPC Live's blocks top down: the portrait, the start,
// the conversation. Rearrange it in the dev server's pipeline view and paste
// the copied layout over this one.
var npcLiveLayout = ports.Layout{
	"portrait-prompt":    {X: 0, Y: 0},
	"portrait":           {X: 0, Y: 64},
	"start-game":         {X: 150, Y: 128},
	"player-input-audio": {X: 300, Y: 192},
	"live-session":       {X: 150, Y: 256},
	"observer":           {X: 300, Y: 256},
}
