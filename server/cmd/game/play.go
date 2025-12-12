package game

import (
	"cgl/api/client"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var shareID string

var gamePlayCmd = &cobra.Command{
	Use:   "play <game-id>",
	Short: "Start a new game session",
	Long:  "Create a new session for a game and start playing.",
	Args:  cobra.ExactArgs(1),
	Run:   runGamePlay,
}

func init() {
	gamePlayCmd.Flags().StringVarP(&shareID, "share", "s", "", "API key share ID to use (optional, uses default if not provided)")
	Cmd.AddCommand(gamePlayCmd)
}

type createSessionRequest struct {
	ShareID uuid.UUID `json:"shareId"`
}

type createSessionResponse struct {
	SessionID uuid.UUID `json:"sessionId"`
}

func runGamePlay(cmd *cobra.Command, args []string) {
	gameID := args[0]

	var req createSessionRequest
	if shareID != "" {
		id, err := uuid.Parse(shareID)
		if err != nil {
			log.Fatalf("Invalid share ID: %v", err)
		}
		req.ShareID = id
	}
	var resp createSessionResponse

	if err := client.ApiPost("games/"+gameID+"/sessions", req, &resp); err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	log.Printf("Session created: %s", resp.SessionID)
	log.Printf("Game ID: %s", gameID)

	// TODO: Start interactive game loop
}
