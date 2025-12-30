package apikey

import (
	"cgl/api/client"
	"cgl/api/routes"
	"cgl/obj"
	"fmt"
	"log"
	"strings"

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

func runAdd(cmd *cobra.Command, args []string) {
	platform := args[0]
	key := strings.TrimSpace(args[1])

	req := routes.CreateApiKeyRequest{
		Name:     keyName,
		Platform: platform,
		Key:      key,
	}

	var resp obj.ApiKeyShare
	if err := client.ApiPost("apikeys/new", req, &resp); err != nil {
		log.Fatalf("Failed to add API key: %v", err)
	}

	fmt.Printf("API key added (share id): %s\n", resp.ID)
}
