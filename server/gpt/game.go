package gpt

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"net/http"
	"strings"
	"time"
	"webapp-server/constants"
	"webapp-server/db"
	"webapp-server/obj"
)

const template = `You are a text-adventure API. You get inputs, what the player wants to do. You act as the game master and decide, what happens. You decide, what's possible and what's not possible - not the player.
If the player posts an action, that doesn't work in the world you are simulating, then continue the story with the player failing in his attempt.
You're job is not to please the player, but to create a coherent world. You're job is to create a world, that is fun to explore. You're job is to create a world, that is fun to play in.

The game frontend sends player actions together with player status as json. Example:

{{INPUT_EXAMPLE}}

Possible action types are: 
` + obj.GameInputTypeAction + `: action, which the player wants to do
` + obj.GameInputTypeIntro + `: system starts a new game session, message contains instructions generating the first scene

When you receive a player action, you continue the story based on his actions and update the player status.

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

func CreateGameSession(game *obj.Game, userId uint, apiKey string) (session *obj.Session, err error) {
	if game == nil {
		return nil, fmt.Errorf("game is nil")
	}

	log.Printf("CreateGameSession, game.ID %d, userId %d", game.ID, userId)

	actionInput := obj.GameActionInput{
		Type:    obj.GameInputTypeAction,
		Message: "drink the potion",
		Status: []obj.StatusField{
			{Name: "gold", Value: "100"},
			{Name: "items", Value: "sword, potion"},
		},
	}
	actionInputStr, _ := json.Marshal(actionInput)

	actionOutput := obj.GameActionOutputGpt{
		Story: "You drink the potion. You feel a little bit dizzy. You feel a little bit stronger.",
		Status: []obj.StatusField{
			{Name: "gold", Value: "100"},
			{Name: "items", Value: "sword"},
		},
		Image: "a castle in the background, green grass, late afternoon",
	}
	actionOutputStr, _ := json.Marshal(actionOutput)

	instructions := template
	instructions = strings.ReplaceAll(instructions, "{{INPUT_EXAMPLE}}", string(actionInputStr))
	instructions = strings.ReplaceAll(instructions, "{{OUTPUT_EXAMPLE}}", string(actionOutputStr))
	instructions = strings.ReplaceAll(instructions, "{{SCENARIO}}", game.Scenario)
	log.Printf("Instructions: %s", instructions)

	assistantName := fmt.Sprintf("%s Game #%d", constants.ProjectName, game.ID)
	assistantId, assistantModel, threadId, err := initAssistant(context.Background(), assistantName, instructions, apiKey)
	if err != nil {
		log.Printf("initAssistant failed: %s", err.Error())
		return nil, err
	}
	return &obj.Session{
		GameID:                game.ID,
		AssistantID:           assistantId,
		AssistantInstructions: instructions,
		ThreadID:              threadId,
		UserID:                userId,
		Model:                 assistantModel,
	}, nil
}

func ExecuteAction(session *obj.Session, game *obj.Game, action obj.GameActionInput, apiKey string) (response *obj.GameActionOutput, httpErr *obj.HTTPError) {
	var err error
	actionSerialized, _ := json.Marshal(action)
	log.Printf("ExecuteAction, session %d, action %s", session.ID, string(actionSerialized))

	var gptResponse string
	timeStart := time.Now()
	if gptResponse, err = AddMessageToThread(
		context.Background(),
		*session,
		openai.ChatMessageRoleUser,
		string(actionSerialized),
		apiKey,
	); err != nil {
		log.Printf("AddMessageToThread failed: %s", err.Error())
		return nil, &obj.HTTPError{StatusCode: 500, Message: "GPT error: " + err.Error()}
	}
	gptResponse = strings.TrimPrefix(gptResponse, "```json")
	gptResponse = strings.TrimSuffix(gptResponse, "```")
	gptResponse = strings.TrimSpace(gptResponse)
	log.Printf("GPT responded: %s", gptResponse)

	if err = json.Unmarshal([]byte(gptResponse), &response); err != nil {
		response = &obj.GameActionOutput{
			Type:  obj.GameOutputTypeError,
			Error: fmt.Sprintf("failed parsing gpt output: %s", err.Error()),
		}
	} else {
		response.Type = obj.GameOutputTypeStory
	}

	response.ChapterId = action.ChapterId
	response.SessionHash = session.Hash
	response.RawInput = string(actionSerialized)
	response.RawOutput = gptResponse
	response.Agent = obj.AgentInfo{
		Key:             ".." + apiKey[len(apiKey)-4:],
		Model:           session.Model,
		Assistant:       session.AssistantID,
		Thread:          session.ThreadID,
		ComputationTime: time.Since(timeStart).String(),
	}
	response.Image = fmt.Sprintf("%s - %s", response.Image, game.ImageStyle)
	if action.ChapterId == 1 {
		response.AssistantInstructions = session.AssistantInstructions
	}

	if _, err = db.AddChapter(session.ID, action.ChapterId, response.RawInput, response.RawOutput, response.Image); err != nil {
		return nil, &obj.HTTPError{StatusCode: http.StatusInternalServerError, Message: "Failed adding chapter"}
	}

	go func() {
		var image []byte
		var imageErr *obj.HTTPError
		if image, imageErr = GenerateImage(context.Background(), apiKey, response.Image); imageErr != nil {
			log.Printf("failed generating image: %s", imageErr)
			return
		}
		if imageErr = db.SetImage(session.ID, action.ChapterId, image); imageErr != nil {
			log.Printf("failed saving image to chapter: %s", imageErr)
			return
		}
		log.Printf("sucessfully generated and stored image for session %d chapter %d", session.ID, action.ChapterId)

		report := obj.SessionUsageReport{
			SessionID: session.ID,
			ApiKey:    apiKey[:8] + "..",
			GameID:    game.ID,
			UserID:    session.UserID,
			UserName:  "-",
			Action:    "gen-image",
			Error:     fmt.Sprintf("%v", imageErr),
		}

		db.WriteSessionUsageReport(report)
	}()

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
