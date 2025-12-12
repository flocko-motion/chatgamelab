package apikey

import (
	"cgl/api/client"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <api-key-id>",
	Short: "Delete an API key",
	Long:  "Delete an API key by its ID.",
	Args:  cobra.ExactArgs(1),
	Run:   runDelete,
}

func init() {
	Cmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) {
	apiKeyID := args[0]

	if err := client.ApiDelete("apikeys/" + apiKeyID); err != nil {
		log.Fatalf("Failed to delete API key: %v", err)
	}

	fmt.Println("API key deleted successfully")
}
