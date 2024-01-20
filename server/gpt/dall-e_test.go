package gpt

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func apiKey() string {
	// load ".apikey-for-unittests" to string
	key, err := os.ReadFile(".apikey-for-unittests")
	if err != nil {
		panic(err)
	}
	return string(key)
}

func TestDalleImageGeneration(t *testing.T) {
	ctx := context.Background()

	image, err := GenerateImage(ctx, apiKey(), "Parrot on a skateboard performs a trick, cartoon style, natural light, high detail")
	assert.NoError(t, err)
	assert.NotEmpty(t, image)
}
