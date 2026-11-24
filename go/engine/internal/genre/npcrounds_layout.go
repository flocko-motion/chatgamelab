package genre

import "engine/internal/ports"

// npcRoundsLayout draws NPC-Rounds' blocks top down: the cover, the start,
// the player's two inputs, the character. Settings and outputs hang off
// their blocks and have no place of their own. Rearrange it in the dev
// server's pipeline view and paste the copied layout over this one.
var npcRoundsLayout = ports.Layout{
	"gallery-image":      {X: 110, Y: 40},
	"cast":               {X: 260, Y: 40},
	"cast-fallback":      {X: 360, Y: 40},
	"start-game":         {X: 110, Y: 130},
	"player-input-text":  {X: 0, Y: 220},
	"player-input-audio": {X: 220, Y: 220},
	"hearing":            {X: 220, Y: 300},
	"character":          {X: 110, Y: 390},
}
