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

var gameSessionsCmd = &cobra.Command{
	Use:   "sessions <game-id>",
	Short: "List all sessions for a game",
	Long:  "Fetch and display all sessions for a specific game from the API.",
	Args:  cobra.ExactArgs(1),
	Run:   runGameSessions,
}

func init() {
	Cmd.AddCommand(gameSessionsCmd)
}

func runGameSessions(cmd *cobra.Command, args []string) {
	gameID := args[0]

	var sessions []obj.GameSession
	if err := client.ApiGet("games/"+gameID+"/sessions", &sessions); err != nil {
		log.Fatalf("Failed to fetch sessions: %v", err)
	}

	if len(sessions) == 0 {
		log.Println("No sessions found for this game.")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"ID", "User", "API Key ID", "Model", "Created"})

	for _, s := range sessions {
		userID := s.UserID.String()[:8] + "..."
		apiKeyID := s.ApiKeyID.String()[:8] + "..."

		created := "n/a"
		if s.Meta.CreatedAt != nil {
			created = s.Meta.CreatedAt.Format("2006-01-02 15:04")
		}

		table.Append([]string{
			s.ID.String(),
			userID,
			apiKeyID,
			functional.Shorten(s.Model, 20),
			created,
		})
	}
	table.Render()
}
