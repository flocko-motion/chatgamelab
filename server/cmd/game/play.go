package game

import (
	"cgl/api/client"
	"cgl/api/endpoints"
	"cgl/obj"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var shareID string
var modelID string

const imageOutputDir = "/tmp/cgl"

var gamePlayCmd = &cobra.Command{
	Use:   "play <game-id>",
	Short: "Start a new game session",
	Long:  "Create a new session for a game and start playing.",
	Args:  cobra.ExactArgs(1),
	Run:   runGamePlay,
}

func init() {
	gamePlayCmd.Flags().StringVarP(&shareID, "share", "s", "", "API key share ID to use (optional, uses default if not provided)")
	gamePlayCmd.Flags().StringVarP(&modelID, "model", "m", "", "AI model to use (optional, uses platform default if not provided)")
	Cmd.AddCommand(gamePlayCmd)
}

func runGamePlay(cmd *cobra.Command, args []string) {
	gameID := args[0]

	var req endpoints.CreateSessionRequest
	if shareID != "" {
		id, err := uuid.Parse(shareID)
		if err != nil {
			log.Fatalf("Invalid share ID: %v", err)
		}
		req.ShareID = id
	}
	req.Model = modelID
	var resp endpoints.CreateSessionResponse

	if err := client.ApiPost("games/"+gameID+"/sessions", req, &resp); err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	// Print full JSON response
	respJSON, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("=== Session Created ===\n%s\n\n", respJSON)

	// Stream the response
	if err := streamMessageResponse(resp.MessageID); err != nil {
		log.Fatalf("Failed to stream response: %v", err)
	}
	fmt.Printf("Use 'game action %s' to continue playing\n", resp.SessionID)

}

// streamMessageResponse connects to SSE and streams text/image content
// This function is reusable for both session creation and game actions
func streamMessageResponse(messageID uuid.UUID) error {
	// Ensure output directory exists
	if err := os.MkdirAll(imageOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create image output dir: %v", err)
	}

	var textBuilder strings.Builder
	imageCount := 0

	fmt.Println("=== Streaming Response ===")

	err := client.StreamSSE(fmt.Sprintf("messages/%s/stream", messageID), func(chunk obj.GameSessionMessageChunk) error {
		// Handle text chunks
		if chunk.Text != "" {
			fmt.Print(chunk.Text)
			textBuilder.WriteString(chunk.Text)
		}
		if chunk.TextDone {
			fmt.Println("\n[TEXT DONE]")
		}

		// Handle image chunks
		if len(chunk.ImageData) > 0 {
			imageCount++
			filename := fmt.Sprintf("%s/%s_%d.png", imageOutputDir, messageID, imageCount)
			if err := os.WriteFile(filename, chunk.ImageData, 0644); err != nil {
				fmt.Printf("\n[IMAGE ERROR: %v]\n", err)
			} else {
				fmt.Printf("\n[IMAGE: %d bytes -> %s]\n", len(chunk.ImageData), filename)
			}
		}
		if chunk.ImageDone {
			fmt.Println("[IMAGE DONE]")
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println("\n=== Stream Complete ===")
	fmt.Printf("Total text: %d chars\n", textBuilder.Len())
	fmt.Printf("Total images: %d\n", imageCount)

	return nil
}
