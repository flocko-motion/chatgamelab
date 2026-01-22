package institution

import (
	"cgl/api/client"
	"cgl/api/routes"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new institution",
	Long:  "Create a new institution with the specified name.",
	Args:  cobra.ExactArgs(1),
	Run:   runCreate,
}

func init() {
	Cmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) {
	name := args[0]

	req := routes.CreateInstitutionRequest{
		Name: name,
	}

	var institution obj.Institution
	if err := client.ApiPost("institutions", req, &institution); err != nil {
		log.Fatalf("Failed to create institution: %v", err)
	}

	fmt.Printf("✓ Institution created successfully\n")
	fmt.Printf("  ID:   %s\n", institution.ID)
	fmt.Printf("  Name: %s\n", institution.Name)
}
