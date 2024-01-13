package gpt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"strings"
	"webapp-server/constants"
	"webapp-server/obj"
)

const template = `You are a text-adventure API. You get inputs, what the player wants to do. You act as the game master and decide, what happens. You decide, what's possible and what's not possible - not the player.
If the player posts an action, that doesn't work in the world you are simulating, then continue the story with the player failing in his attempt.
You're job is not to please the player, but to create a coherent world. You're job is to create a world, that is fun to explore. You're job is to create a world, that is fun to play in.

The game frontend sends player actions together with player status as json. Example:

{{INPUT_EXAMPLE}}

Possible action types are: 
` + obj.GameActionTypePlayerInput + `: input from player
` + obj.GameActionTypeInitialization + `: system starts a new game session, message contains instructions generating the first scene

You always answer with a result json. The result json must exactly follow the format of this Example:

{{OUTPUT_EXAMPLE}}

As you see in the example, you have to update the status after each player action. The "image" field describes the new scenery for a generative image AI to produce artwork.

The language and literary style ouf your output should follow the scenario definition.

The JSON structure, field names, etc. are fixed and must not be changed or translated. The image description should be in english always.
Any changes to the JSON structure will break the game frontend.

You always stay in your role. You are the game master. You are the world. You are the narrator. You are the storyteller. You decide, what's possible and what not. You are the text-adventure engine. You are the game. Don't please the player, challenge him.

The scenario:

{{SCENARIO}}
`

func CreateGameSession(game *obj.Game, userId uint) (session *obj.Session, err error) {
	if game == nil {
		return nil, fmt.Errorf("game is nil")
	}

	actionInput := obj.GameActionInput{
		Type:    obj.GameActionTypePlayerInput,
		Message: "drink the potion",
		Status: []obj.StatusField{
			{Name: "gold", Value: "100"},
			{Name: "items", Value: "sword, potion"},
		},
	}
	actionInputStr, _ := json.Marshal(actionInput)

	actionOutput := obj.GameActionOutput{
		Story: "You drink the potion. You feel a little bit dizzy. You feel a little bit stronger.",
		Status: []obj.StatusField{
			{Name: "gold", Value: "100"},
			{Name: "items", Value: "sword"},
		},
		Image: "a castle in the background, green grass, late afternoon",
	}
	actionOutputStr, _ := json.Marshal(actionOutput)

	instructions := template
	strings.ReplaceAll(instructions, "{{INPUT_EXAMPLE}}", string(actionInputStr))
	strings.ReplaceAll(instructions, "{{OUTPUT_EXAMPLE}}", string(actionOutputStr))
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

func ExecuteAction(session *obj.Session, action obj.GameActionInput) (response *obj.GameActionOutput, httpErr *obj.HTTPError) {
	var err error
	actionSerialized, _ := json.Marshal(action)

	var res string
	if res, err = AddMessageToThread(context.Background(), *session, openai.ChatMessageRoleUser, string(actionSerialized)); err != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "GPT execution error: " + err.Error()}
	}

	if err = json.Unmarshal([]byte(res), &response); err != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "GPT result parsing error: " + err.Error()}
	}

	return response, nil
}

/*func serializeStatusFields(statusFields []obj.StatusField) string {
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
*/