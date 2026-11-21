package genre

import "engine/internal/ports"

// adventureLayout draws Adventure top down: preparation, the start, the
// player's input, the story, then what reaches the player and what failed.
// Rearrange it in the dev server's pipeline view and paste the copied layout
// over this one.
var adventureLayout = ports.Layout{
	"image-style":        {Row: 0, Col: 0},
	"adapted-style":      {Row: 0, Col: 1},
	"condensed-scenario": {Row: 0, Col: 3},
	"condense-scenario":  {Row: 0, Col: 4},
	"start-game":         {Row: 1, Col: 2},
	"problem":            {Row: 1, Col: 4},
	"player-input-text":  {Row: 2, Col: 1},
	"player-input-audio": {Row: 2, Col: 3},
	"rephrase":           {Row: 3, Col: 1},
	"rephrase-audio":     {Row: 3, Col: 3},
	"props-store":        {Row: 4, Col: 0},
	"outline":            {Row: 4, Col: 2},
	"image-prompt":       {Row: 4, Col: 5},
	"expand":             {Row: 5, Col: 2},
	"image":              {Row: 5, Col: 5},
	"out-props":          {Row: 6, Col: 0},
	"out-said":           {Row: 6, Col: 1},
	"out-text":           {Row: 6, Col: 2},
	"out-audio":          {Row: 6, Col: 3},
	"out-image":          {Row: 6, Col: 5},
	"turn-failed":        {Row: 7, Col: 2},
	"voice-failed":       {Row: 7, Col: 3},
	"image-failed":       {Row: 7, Col: 5},
}
