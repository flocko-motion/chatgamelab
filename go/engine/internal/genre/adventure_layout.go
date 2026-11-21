package genre

import "engine/internal/ports"

// adventureLayout draws Adventure's blocks top down: preparation, the start,
// the player's input, the story. Rearrange it in the dev server's pipeline
// view and paste the copied layout over this one.
var adventureLayout = ports.Layout{
	"image-style":        {X: 117, Y: 0},
	"adapted-style":      {X: 199, Y: 1},
	"condense-scenario":  {X: 400, Y: 1},
	"condensed-scenario": {X: 287, Y: 2},
	"start-game":         {X: 200, Y: 75},
	"image-prompt":       {X: 286, Y: 130},
	"image":              {X: 397, Y: 130},
	"rephrase-audio":     {X: 102, Y: 163},
	"player-input-audio": {X: 0, Y: 165},
	"outline":            {X: 200, Y: 199},
	"player-input-text":  {X: 0, Y: 235},
	"rephrase":           {X: 102, Y: 237},
	"props-store":        {X: 301, Y: 239},
	"expand":             {X: 199, Y: 274},
}
