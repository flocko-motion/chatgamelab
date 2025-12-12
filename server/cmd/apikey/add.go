package apikey

import (
	"cgl/api/client"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <platform> <key>",
	Short: "Add a new API key",
	Long:  "Add a new API key for the specified platform (e.g., openai, mock).",
	Args:  cobra.ExactArgs(2),
	Run:   runAdd,
}

var keyName string

func init() {
	addCmd.Flags().StringVarP(&keyName, "name", "n", "", "Name for the API key")
	Cmd.AddCommand(addCmd)
}

type addApiKeyRequest struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Key      string `json:"key"`
}

type addApiKeyResponse struct {
	ID uuid.UUID `json:"id"`
}

func runAdd(cmd *cobra.Command, args []string) {
	platform := args[0]
	key := strings.TrimSpace(args[1])

	req := addApiKeyRequest{
		Name:     keyName,
		Platform: platform,
		Key:      key,
	}

	var resp addApiKeyResponse
	if err := client.ApiPost("apikeys/new", req, &resp); err != nil {
		log.Fatalf("Failed to add API key: %v", err)
	}

	fmt.Printf("API key added: %s\n", resp.ID)
}
