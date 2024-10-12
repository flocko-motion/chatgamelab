package gpt

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"strings"
	"time"
	"webapp-server/obj"
)

func newClient(apiKey string) *openai.Client {
	return openai.NewClient(apiKey)
}

func initAssistant(ctx context.Context, name, instructions, apiKey string) (assistantId string, threadId string, err error) {
	log.Printf("initAssistant: %s", name)

	log.Printf("newClient..")
	client := newClient(apiKey)

	models, err := client.ListModels(context.Background())
	if err != nil {
		return "", "", err
	}
	bestModel := ""
	var bestModelVersion float64
	var bestModelDate int64
	for _, model := range models.Models {
		log.Printf("Model: %s", model.ID)
		var modelVersion float64
		var modelDate int64
		if strings.HasPrefix(model.ID, "gpt-3.5-turbo") {
			modelVersion = 3.5
			modelDate = model.CreatedAt
		}
		if strings.HasPrefix(model.ID, "gpt-4-turbo") {
			modelVersion = 4
			modelDate = model.CreatedAt
		}
		if strings.HasPrefix(model.ID, "gpt-4o") {
			modelVersion = 4.1
			modelDate = model.CreatedAt
		}
		// realtime models are not suitable for assistant
		if strings.Contains(model.ID, "-realtime-") {
			continue
		}
		if modelVersion > bestModelVersion || (modelVersion == bestModelVersion && modelDate > bestModelDate) {
			bestModel = model.ID
			bestModelVersion = modelVersion
			bestModelDate = modelDate
		}
	}
	log.Printf("Best model for api key %s: %s", apiKey, bestModel)
	if bestModelVersion < 4 {
		if len(apiKey) < 5 {
			log.Printf("Malformed API key: %s", apiKey)
			return "", "", fmt.Errorf("malformed API key")
		}
		return "", "", fmt.Errorf("API key %s does not have access to GPT-4", apiKey[:5]+"..."+apiKey[len(apiKey)-5:])
	}

	assistantCfg := openai.AssistantRequest{
		Model:        bestModel,
		Instructions: &instructions,
		Name:         &name,
	}

	// DEACTIVATED: we're not updating existing assistants, because this can cause conflicts with running games
	//var assistants openai.AssistantsList
	//log.Printf("ListAssistants..")
	//if assistants, err = client.ListAssistants(ctx, nil, nil, nil, nil); err != nil {
	//	log.Printf("ListAssistants failed: %s", err.Error())
	//	return
	//}
	//
	//log.Printf("Checking existing assistants..")
	//for _, assistant := range assistants.Assistants {
	//	log.Printf("Found assistant: %s", *assistant.Name)
	//	if *assistant.Name == name {
	//		assistantId = assistant.ID
	//		log.Printf("Match! '%s' found, id=%s\n", name, assistant.ID)
	//		break
	//	}
	//}

	var assistant openai.Assistant
	//if assistantId == "" {
	//log.Printf("Assistant '%s' not found, creating\n", name)
	assistant, err = client.CreateAssistant(context.Background(), assistantCfg)
	assistantId = assistant.ID
	log.Printf("Assistant '%s' created, id=%s\n", name, assistant.ID)
	//} else {
	//	assistant, err = client.ModifyAssistant(context.Background(), assistantId, assistantCfg)
	//	log.Printf("Assistant '%s' updated, id=%s\n", name, assistant.ID)
	//}
	if err != nil {
		return
	}

	var thread openai.Thread
	if thread, err = client.CreateThread(ctx, openai.ThreadRequest{
		// it's possible to give a chat history here to continue a conversation
	}); err != nil {
		return
	}
	log.Printf("Thread created: %s\n", thread.ID)
	threadId = thread.ID

	return
}

func AddMessageToThread(ctx context.Context, session obj.Session, role, message, apiKey string) (response string, err error) {
	client := newClient(apiKey)

	var messageObject openai.Message
	if messageObject, err = client.CreateMessage(ctx, session.ThreadID, openai.MessageRequest{
		Role:    role,
		Content: message,
	}); err != nil {
		return
	}
	log.Printf("Message created: %s\n", messageObject.ID)

	var run openai.Run
	if run, err = client.CreateRun(ctx, session.ThreadID, openai.RunRequest{
		AssistantID: session.AssistantID,
	}); err != nil {
		return
	}
	log.Printf("Run %s created", run.ID)

	for run.Status == openai.RunStatusQueued || run.Status == openai.RunStatusInProgress {
		if run, err = client.RetrieveRun(ctx, session.ThreadID, run.ID); err != nil {
			return
		}
		time.Sleep(1 * time.Second)
	}
	log.Printf("Run %s completed", run.ID)

	limit := 1
	var msgList openai.MessagesList
	if msgList, err = client.ListMessage(ctx, session.ThreadID, &limit, nil, nil, nil); err != nil {
		return
	}
	if len(msgList.Messages) != 1 {
		err = fmt.Errorf("expected 1 message, got %d", len(msgList.Messages))
		return
	}
	log.Println("Fetched messages")

	content := msgList.Messages[0].Content
	if len(content) != 1 {
		err = fmt.Errorf("expected 1 content, got %d", len(content))
		return
	}
	if content[0].Type != "text" {
		err = fmt.Errorf("expected text content, got %s", content[0].Type)
		return
	}
	response = content[0].Text.Value
	return
}
