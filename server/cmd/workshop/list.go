package workshop

import (
	"cgl/api/client"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [--institution-id=<id>]",
	Short: "List workshops",
	Long:  "List all workshops or workshops, either for a given institution or for the institution of the user.",
	Run:   runList,
}

var institutionID string

func init() {
	Cmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&institutionID, "institution-id", "", "Pick institution ID")
}

func runList(cmd *cobra.Command, args []string) {
	endpoint := "workshops"
	if institutionID != "" {
		endpoint = fmt.Sprintf("workshops?institutionId=%s", institutionID)
	}

	var workshops []obj.Workshop
	if err := client.ApiGet(endpoint, &workshops); err != nil {
		log.Fatalf("Failed to list workshops: %v", err)
	}

	if len(workshops) == 0 {
		fmt.Println("No workshops found.")
		return
	}

	fmt.Printf("Found %d workshop(s):\n\n", len(workshops))
	for _, w := range workshops {
		fmt.Printf("  %s - %s (Active: %v, Public: %v)\n", w.ID, w.Name, w.Active, w.Public)
	}
}
