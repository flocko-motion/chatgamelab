package game

import (
	"cgl/api/client"
	"cgl/api/endpoints"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var gameActionCmd = &cobra.Command{
	Use:   "action <session-id> <message>",
	Short: "Send an action to an existing game session",
	Long:  "Send a player action/message to an existing game session and stream the response.",
	Args:  cobra.ExactArgs(2),
	Run:   runGameAction,
}

func init() {
	Cmd.AddCommand(gameActionCmd)
}

func runGameAction(cmd *cobra.Command, args []string) {
	sessionID, err := uuid.Parse(args[0])
	if err != nil {
		log.Fatalf("Invalid session ID: %v", err)
	}
	message := args[1]

	req := endpoints.SessionActionRequest{Message: message}
	var resp endpoints.SessionActionResponse

	if err := client.ApiPost(fmt.Sprintf("sessions/%s", sessionID), req, &resp); err != nil {
		log.Fatalf("Failed to send action: %v", err)
	}

	// Print full JSON response
	respJSON, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("=== Action Sent ===\n%s\n\n", respJSON)

	// Stream the response (reuse function from play.go)
	if err := streamMessageResponse(resp.MessageID); err != nil {
		log.Fatalf("Failed to stream response: %v", err)
	}
}
