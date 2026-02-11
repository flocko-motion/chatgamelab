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
	table.Header([]string{"ID", "Owner", "Created", "Public", "Name", "Description"})

	for _, g := range games {

		public := "no"
		if g.Public {
			public = "yes"
		}

		table.Append([]string{
			g.ID.String(),
			g.Meta.CreatedBy.UUID.String(),
			g.Meta.CreatedAt.Format("2006-01-02"),
			public,
			functional.Shorten(g.Name, 30),
			functional.Shorten(g.Description, 60),
		})
	}
	table.Render()
}
