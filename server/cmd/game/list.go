package game

import (
	"cgl/api/client"
	"cgl/functional"
	"cgl/obj"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var gameListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all games",
	Long:  "Fetch and display all games from the API.",
	Run:   runGameList,
}

func init() {
	Cmd.AddCommand(gameListCmd)
}

func runGameList(cmd *cobra.Command, args []string) {
	var games []obj.Game
	if err := client.ApiGet("games", &games); err != nil {
		log.Fatalf("Failed to fetch games: %v", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"ID", "Name", "Owner", "Created", "Public", "Description"})

	for _, g := range games {
		owner := "n/a"
		if g.Meta.CreatedBy.Valid {
			owner = g.Meta.CreatedBy.UUID.String()[:8] + "..."
		}

		created := "n/a"
		if g.Meta.CreatedAt != nil {
			created = g.Meta.CreatedAt.Format("2006-01-02")
		}

		public := "no"
		if g.Public {
			public = "yes"
		}

		table.Append([]string{
			g.ID.String(),
			functional.Shorten(g.Name, 30),
			owner,
			created,
			public,
			functional.Shorten(g.Description, 40),
		})
	}
	table.Render()
}
