package gpt

import (
	"context"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"webapp-server/constants"
)

func TestCreateThread(t *testing.T) {
	ctx := context.Background()

	instructions := "You are a text-adventure engine. The player is a wizard. The player writes, what he wants to do. You are the game master and you write, what happens. You decide, what's possible and what's not possible - not the player."
	assistantId, threadId, err := initAssistant(ctx, constants.ProjectName+"_test", instructions)
	assert.NoError(t, err)
	assert.NotEmpty(t, assistantId)
	assert.NotEmpty(t, threadId)

	var response string
	response, err = SendUserMessage(ctx, threadId, assistantId, "I look around the room")
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	log.Printf("Message response: %s\n", response)
}
