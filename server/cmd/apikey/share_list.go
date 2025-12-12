package apikey

import (
	"cgl/api/client"
	"cgl/obj"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var shareListCmd = &cobra.Command{
	Use:   "list <api-key-id>",
	Short: "List all shares for an API key",
	Long:  "List all shares (users, workshops, institutions) for a specific API key.",
	Args:  cobra.ExactArgs(1),
	Run:   runShareList,
}

func init() {
	shareCmd.AddCommand(shareListCmd)
}

func runShareList(cmd *cobra.Command, args []string) {
	apiKeyID := args[0]

	var shares []obj.ApiKeyShare
	if err := client.ApiGet("apikeys/"+apiKeyID, &shares); err != nil {
		log.Fatalf("Failed to fetch shares: %v", err)
	}

	if len(shares) == 0 {
		log.Println("No shares found.")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Share ID", "Type", "Target", "Public Plays", "Created"})

	for _, s := range shares {
		shareType := ""
		target := ""

		if s.User != nil {
			shareType = "user"
			target = s.User.Name
		} else if s.Workshop != nil {
			shareType = "workshop"
			target = s.Workshop.Name
		} else if s.Institution != nil {
			shareType = "institution"
			target = s.Institution.Name
		}

		created := "n/a"
		if s.Meta.CreatedAt != nil {
			created = s.Meta.CreatedAt.Format("2006-01-02")
		}

		allowPublic := "no"
		if s.AllowPublicSponsoredPlays {
			allowPublic = "yes"
		}

		table.Append([]string{
			s.ID.String(),
			shareType,
			target,
			allowPublic,
			created,
		})
	}
	table.Render()
}
