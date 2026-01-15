package institution

import (
	"cgl/api/client"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all institutions",
	Long:  "List all institutions in the system.",
	Run:   runList,
}

func init() {
	Cmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	var institutions []obj.Institution
	if err := client.ApiGet("institutions", &institutions); err != nil {
		log.Fatalf("Failed to list institutions: %v", err)
	}

	if len(institutions) == 0 {
		fmt.Println("No institutions found.")
		return
	}

	fmt.Printf("Found %d institution(s):\n\n", len(institutions))
	for _, inst := range institutions {
		fmt.Printf("  %s - %s\n", inst.ID, inst.Name)
	}
}
