package apikey

import (
	"cgl/api/client"
	"cgl/api/routes"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var modifyCmd = &cobra.Command{
	Use:   "modify <share-id>",
	Short: "Modify an API key",
	Long:  "Modify an API key's properties (e.g., name) using its share ID.",
	Args:  cobra.ExactArgs(1),
	Run:   runModify,
}

var modifyName string

func init() {
	modifyCmd.Flags().StringVarP(&modifyName, "name", "n", "", "New name for the API key")
	Cmd.AddCommand(modifyCmd)
}

func runModify(cmd *cobra.Command, args []string) {
	apiKeyID := args[0]

	if modifyName == "" {
		log.Fatalf("At least one modification flag is required (e.g., --name)")
	}

	req := routes.UpdateApiKeyRequest{
		Name: &modifyName,
	}

	if err := client.ApiPatch("apikeys/"+apiKeyID, req, nil); err != nil {
		log.Fatalf("Failed to modify API key: %v", err)
	}

	fmt.Println("API key modified successfully")
}
