package apikey

import (
	"cgl/api/client"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <share-id>",
	Short: "Delete an API key and all its shares",
	Long:  "Delete an API key by its share ID. This removes the key and all associated shares.",
	Args:  cobra.ExactArgs(1),
	Run:   runDelete,
}

func init() {
	Cmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) {
	shareID := args[0]

	if err := client.ApiDelete("apikeys/" + shareID + "?cascade=true"); err != nil {
		log.Fatalf("Failed to delete API key: %v", err)
	}

	fmt.Println("API key deleted successfully")
}
