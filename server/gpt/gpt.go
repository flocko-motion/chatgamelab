package gpt

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"time"
	"webapp-server/obj"
)

func newClient(apiKey string) *openai.Client {
	return openai.NewClient(apiKey)
}

func initAssistant(ctx context.Context, name, instructions, apiKey string) (assistantId string, threadId string, err error) {
	assistantCfg := openai.AssistantRequest{
		Model:        openai.GPT4TurboPreview,
		Instructions: &instructions,
		Name:         &name,
	}

	client := newClient(apiKey)

	var assistants openai.AssistantsList
	if assistants, err = client.ListAssistants(ctx, nil, nil, nil, nil); err != nil {
		return
	}

	for _, assistant := range assistants.Assistants {
		if *assistant.Name == name {
			assistantId = assistant.ID
			break
		}
	}

	var assistant openai.Assistant
	if assistantId == "" {
		assistant, err = client.CreateAssistant(context.Background(), assistantCfg)
		assistantId = assistant.ID
		log.Printf("Assistant '%s' created, id=%s\n", name, assistant.ID)
	} else {
		assistant, err = client.ModifyAssistant(context.Background(), assistantId, assistantCfg)
		log.Printf("Assistant '%s' updated, id=%s\n", name, assistant.ID)
	}
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
