package game

import (
	"cgl/api/client"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var apiKeyID string

var gamePlayCmd = &cobra.Command{
	Use:   "play <game-id>",
	Short: "Start a new game session",
	Long:  "Create a new session for a game and start playing.",
	Args:  cobra.ExactArgs(1),
	Run:   runGamePlay,
}

func init() {
	gamePlayCmd.Flags().StringVarP(&apiKeyID, "api-key", "k", "", "API key ID to use (optional, uses game's public key if not provided)")
	Cmd.AddCommand(gamePlayCmd)
}

type createSessionRequest struct {
	ApiKeyID uuid.UUID `json:"apiKeyId"`
}

type createSessionResponse struct {
	SessionID uuid.UUID `json:"sessionId"`
}

func runGamePlay(cmd *cobra.Command, args []string) {
	gameID := args[0]

	var req createSessionRequest
	if apiKeyID != "" {
		keyID, err := uuid.Parse(apiKeyID)
		if err != nil {
			log.Fatalf("Invalid API key ID: %v", err)
		}
		req.ApiKeyID = keyID
	}
	var resp createSessionResponse

	if err := client.ApiPost("games/"+gameID+"/sessions", req, &resp); err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	log.Printf("Session created: %s", resp.SessionID)
	log.Printf("Game ID: %s", gameID)

	// TODO: Start interactive game loop
}
