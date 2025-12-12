package apikey

import (
	"cgl/api/client"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var shareDeleteCmd = &cobra.Command{
	Use:   "delete <share-id>",
	Short: "Delete a share",
	Long:  "Delete an API key share by its ID.",
	Args:  cobra.ExactArgs(1),
	Run:   runShareDelete,
}

func init() {
	shareCmd.AddCommand(shareDeleteCmd)
}

func runShareDelete(cmd *cobra.Command, args []string) {
	shareID := args[0]

	if err := client.ApiDelete("apikeys/shares/" + shareID); err != nil {
		log.Fatalf("Failed to delete share: %v", err)
	}

	fmt.Println("Share deleted successfully")
}
