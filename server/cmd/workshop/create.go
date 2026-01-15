package workshop

import (
	"cgl/api/client"
	"cgl/api/routes"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name> [institutionId]",
	Short: "Create a new workshop",
	Long:  "Create a new workshop. Institution ID is auto-resolved from your role unless you're an admin.",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runCreate,
}

var (
	active bool
	public bool
)

func init() {
	Cmd.AddCommand(createCmd)
	createCmd.Flags().BoolVar(&active, "active", true, "Workshop is active")
	createCmd.Flags().BoolVar(&public, "public", false, "Workshop is public")
}

func runCreate(cmd *cobra.Command, args []string) {
	name := args[0]

	var institutionID *string
	if len(args) > 1 {
		institutionID = &args[1]
	}

	req := routes.CreateWorkshopRequest{
		InstitutionID: institutionID,
		Name:          name,
		Active:        active,
		Public:        public,
	}

	var workshop obj.Workshop
	if err := client.ApiPost("workshops", req, &workshop); err != nil {
		log.Fatalf("Failed to create workshop: %v", err)
	}

	fmt.Printf("✓ Workshop created successfully\n")
	fmt.Printf("  ID:     %s\n", workshop.ID)
	fmt.Printf("  Name:   %s\n", workshop.Name)
	fmt.Printf("  Active: %v\n", workshop.Active)
	fmt.Printf("  Public: %v\n", workshop.Public)
}
