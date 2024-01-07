package gpt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"webapp-server/constants"
	"webapp-server/obj"
)

const template = `You are a text-adventure API. You get inputs, what the player wants to do. You act as the game master and decide, what happens. You decide, what's possible and what's not possible - not the player.
If the player posts an action, that doesn't work in the world you are simulating, then continue the story with the player failing in his attempt.
You're job is not to please the player, but to create a coherent world. You're job is to create a world, that is fun to explore. You're job is to create a world, that is fun to play in.

The game frontend sends player actions together with player status as json. Example:

{
  "action": "Enter the red door using red key",
 "status": {{STATUS}}
}

You answer with a result json. The result json must exactly follow the format of this Example:

{
  "story": "You opened the red door with the key. The key stuck in the door. You're now outside the castle.",
 "status": {{STATUS}},
"image":"a castle in the background, green grass, late afternoon"
}

As you see in the example, you have to update the status after each player action. The "image" field describes the new scenery for a generative image AI to produce artwork. 

The player input and the story output should be in the language that the scenario (see below) is written in. 
The JSON structure, field names, etc. are fixed and must not be changed. The image description should be in english always.

The scenario:

{{SCENARIO}}
`

func CreateGameSession(game *obj.Game, userId uint) (session *obj.Session, err error) {
	if game == nil {
		return nil, fmt.Errorf("game is nil")
	}
	// TODO: this is just a placeholder, should be configured in game
	statusFields := []obj.StatusField{
		{Name: "gold", Value: "100"},
		{Name: "items", Value: "sword, potion"},
	}

	instructions := game.Scenario
	strings.ReplaceAll(instructions, "{{STATUS}}", serializeStatusFields(statusFields))
	strings.ReplaceAll(instructions, "{{SCENARIO}}", game.Scenario)

	assistantName := fmt.Sprintf("%s #%d", constants.ProjectName, game.ID)
	assistantId, threadId, err := initAssistant(context.Background(), assistantName, instructions)
	if err != nil {
		return nil, err
	}
	return &obj.Session{
		GameID:      game.ID,
		AssistantID: assistantId,
		ThreadID:    threadId,
		UserID:      userId,
	}, nil
}

func serializeStatusFields(statusFields []obj.StatusField) string {
	fields := make([]map[string]string, len(statusFields))
	for i, statusField := range statusFields {
		fields[i] = map[string]string{
			statusField.Name: statusField.Value,
		}
	}
	bytes, err := json.Marshal(fields)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
