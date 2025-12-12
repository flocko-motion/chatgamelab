package gpt

import (
	"cgl/constants"
	"cgl/db"
	"cgl/obj"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

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

	assistantName := fmt.Sprintf("%s #%d", constants.ProjectName, game.ID)
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
