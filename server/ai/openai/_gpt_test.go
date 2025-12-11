package gpt

import (
	"context"
	"log"
	"testing"
	"cgl/constants"
	"cgl/obj"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestCreateThread(t *testing.T) {
	ctx := context.Background()

	instructions := "You are a text-adventure engine. The player is a wizard. The player writes, what he wants to do. You are the game master and you write, what happens. You decide, what's possible and what's not possible - not the player."
	assistantId, assistantModel, threadId, err := initAssistant(ctx, constants.ProjectName+"_test", instructions, apiKey())
	assert.NoError(t, err)
	assert.NotEmpty(t, assistantId)
	assert.NotEmpty(t, assistantModel)
	assert.NotEmpty(t, threadId)

	var response string
	response, err = AddMessageToThread(ctx, obj.Session{ThreadID: threadId, AssistantID: assistantId}, openai.ChatMessageRoleUser, "I look around the room", apiKey())
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	log.Printf("Message response: %s\n", response)
}
