package gpt

import (
	"bufio"
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"os"
	"time"
)

func newClient() *openai.Client {
	return openai.NewClient(OPENAI_API_KEY)
}

func init() {
	client := newClient()

	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "you are a helpful chatbot",
			},
		},
	}
	fmt.Println("Conversation")
	fmt.Println("---------------------")
	fmt.Print("> ")
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: s.Text(),
		})
		resp, err := client.CreateChatCompletion(context.Background(), req)
		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)
			continue
		}
		fmt.Printf("%s\n\n", resp.Choices[0].Message.Content)
		req.Messages = append(req.Messages, resp.Choices[0].Message)
		fmt.Print("> ")
	}
}

func initAssistant(ctx context.Context, name, instructions string) (assistantId string, threadId string, err error) {
	assistantCfg := openai.AssistantRequest{
		Model:        openai.GPT3Dot5Turbo1106,
		Instructions: &instructions,
		Name:         &name,
	}

	client := newClient()

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

func SendUserMessage(ctx context.Context, threadId, assistantId, message string) (response string, err error) {
	client := newClient()

	var messageObject openai.Message
	if messageObject, err = client.CreateMessage(ctx, threadId, openai.MessageRequest{
		Role:    openai.ChatMessageRoleUser,
		Content: "I look around the room",
	}); err != nil {
		return
	}
	log.Printf("Message created: %s\n", messageObject.ID)

	var run openai.Run
	if run, err = client.CreateRun(ctx, threadId, openai.RunRequest{
		AssistantID: assistantId,
	}); err != nil {
		return
	}
	log.Printf("Run %s created", run.ID)

	for run.Status == openai.RunStatusQueued || run.Status == openai.RunStatusInProgress {
		if run, err = client.RetrieveRun(ctx, threadId, run.ID); err != nil {
			return
		}
		time.Sleep(1 * time.Second)
	}
	log.Printf("Run %s completed", run.ID)

	limit := 1
	var msgList openai.MessagesList
	if msgList, err = client.ListMessage(ctx, threadId, &limit, nil, nil, nil); err != nil {
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
