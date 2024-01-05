package api

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsValidOpenaiKey(t *testing.T) {
	assert.True(t, isOpenaiApiKey("sk-cLoIdjNmT0qkZizSrzZ222BlbkFJKFSqlHsO9AsRO6LU88Qy"))
	assert.False(t, isOpenaiApiKey("sk-..88Qy"))
}
