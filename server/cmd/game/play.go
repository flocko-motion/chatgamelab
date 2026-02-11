package game

import (
	"bufio"
	"cgl/api/client"
	"cgl/api/routes"
	"cgl/obj"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const imageOutputDir = "/tmp/cgl"

var gamePlayCmd = &cobra.Command{
	Use:   "play <game-id|session-id>",
	Short: "Play a game (new or existing session)",
	Long:  "Start a new game session or continue an existing one. Auto-detects whether the ID is a game or session.",
	Args:  cobra.ExactArgs(1),
	Run:   runGamePlay,
}

func init() {
	Cmd.AddCommand(gamePlayCmd)
}

func runGamePlay(cmd *cobra.Command, args []string) {
	id := args[0]

	// Try to detect if this is a session ID or game ID by checking the session endpoint first
	sessionID, err := tryGetSession(id)
	if err != nil {
		// Not a session, try creating a new session for this game
		sessionID = createNewSession(id)
	} else {
		fmt.Printf("=== Resuming Session %s ===\n\n", sessionID)
	}

	// Enter game loop
	gameLoop(sessionID)
}

// tryGetSession checks if the ID is a valid session ID and prints the latest message
func tryGetSession(id string) (uuid.UUID, error) {
	var resp routes.SessionResponse
	err := client.ApiGet("sessions/"+id+"?messages=latest", &resp)
	if err != nil {
		return uuid.Nil, err
	}

	// Print latest message for context
	if len(resp.Messages) > 0 {
		printMessageDetails(resp.Messages[0], time.Now())
	} else {
		fmt.Println("[No messages found]")
	}

	return resp.ID, nil
}

// createNewSession creates a new session for the given game ID
func createNewSession(gameID string) uuid.UUID {
	startTime := time.Now()
	var resp obj.GameSessionMessage
	if err := client.ApiPost("games/"+gameID+"/sessions", struct{}{}, &resp); err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	// Print message details (summary, status, image prompt)
	fmt.Println("=== Session Created ===")
	printMessageDetails(resp, startTime)

	// Stream the initial response
	if err := streamMessageResponse(resp.ID, startTime); err != nil {
		log.Fatalf("Failed to stream response: %v", err)
	}

	return resp.GameSessionID
}

// gameLoop runs the interactive game loop
func gameLoop(sessionID uuid.UUID) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nWhat next?> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("\nGoodbye!")
			return
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Send action to session
		if err := sendAction(sessionID, input); err != nil {
			fmt.Printf("Error: %v\n", err)
			if strings.Contains(err.Error(), string(rune(http.StatusUnauthorized))) {
				fmt.Println("Session may have expired. Please start a new game.")
				return
			}
		}
	}
}

// sendAction sends a player action to the session and streams the response
func sendAction(sessionID uuid.UUID, message string) error {
	startTime := time.Now()
	req := routes.SessionActionRequest{Message: message}
	var resp obj.GameSessionMessage

	if err := client.ApiPost(fmt.Sprintf("sessions/%s", sessionID), req, &resp); err != nil {
		return fmt.Errorf("failed to send action: %v", err)
	}

	// Print the initial response (plot outline, status, image prompt)
	printMessageDetails(resp, startTime)

	// Stream the expanded text and image
	return streamMessageResponse(resp.ID, startTime)
}

// printMessageDetails prints the initial message details (plot outline, status, image prompt)
func printMessageDetails(msg obj.GameSessionMessage, startTime time.Time) {
	// Print message ID with elapsed time
	fmt.Printf("\n[Message ID: %s @ %.1fs]\n", msg.ID, time.Since(startTime).Seconds())

	// Print plot outline (initial message before streaming expands it)
	if msg.Message != "" {
		fmt.Println("\n=== Plot Outline ===")
		fmt.Println(msg.Message)
	}

	// Print status fields
	if len(msg.StatusFields) > 0 {
		fmt.Println("\n=== Status ===")
		for _, sf := range msg.StatusFields {
			fmt.Printf("  %s: %s\n", sf.Name, sf.Value)
		}
	}

	// Print image prompt
	if msg.ImagePrompt != nil && *msg.ImagePrompt != "" {
		fmt.Println("\n=== Image Prompt ===")
		fmt.Println(*msg.ImagePrompt)
	}
	fmt.Println()
}

// streamMessageResponse connects to SSE and streams text/image content
// This function is reusable for both session creation and game actions
func streamMessageResponse(messageID uuid.UUID, startTime time.Time) error {
	// Ensure output directory exists
	if err := os.MkdirAll(imageOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create image output dir: %v", err)
	}

	elapsed := func() string {
		return fmt.Sprintf("%.1fs", time.Since(startTime).Seconds())
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
			fmt.Printf("\n[TEXT DONE @ %s]\n", elapsed())
		}

		// Handle image chunks
		if len(chunk.ImageData) > 0 {
			imageCount++
			filename := fmt.Sprintf("%s/%s_%d.png", imageOutputDir, messageID, imageCount)
			if err := os.WriteFile(filename, chunk.ImageData, 0644); err != nil {
				fmt.Printf("\n[IMAGE ERROR: %v]\n", err)
			} else {
				fmt.Printf("\n[IMAGE %d: %d bytes -> %s @ %s]\n", imageCount, len(chunk.ImageData), filename, elapsed())
			}
		}
		if chunk.ImageDone {
			fmt.Printf("[IMAGE DONE @ %s]\n", elapsed())
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println("\n=== Stream Complete ===")
	fmt.Printf("Total text: %d chars\n", textBuilder.Len())
	fmt.Printf("Total images: %d\n", imageCount)
	fmt.Printf("Total time: %s\n", elapsed())

	return nil
}
