package genre

import "engine/internal/ports"

// npcLiveLayout draws NPC Live's blocks top down: the preparation, the start,
// the conversation. Settings and outputs hang off their blocks and have no
// place of their own. Rearrange it in the dev server's pipeline view and
// paste the copied layout over this one.
var npcLiveLayout = ports.Layout{
	"gallery-image":      {X: 111, Y: 68},
	"greeting":           {X: 210, Y: 68},
	"voice-brief":        {X: 0, Y: 30},
	"voice":              {X: 0, Y: 100},
	"start-game":         {X: 110, Y: 151},
	"player-input-audio": {X: 5, Y: 279},
	"live-session":       {X: 110, Y: 279},
	"observer":           {X: 260, Y: 279},
}
