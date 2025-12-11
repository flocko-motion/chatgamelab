package user

import (
	"cgl/api/client"
	"cgl/functional"
	"cgl/obj"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Long:  "Fetch and display all users from the API.",
	Run:   runUserList,
}

func init() {
	Cmd.AddCommand(userListCmd)
}

func runUserList(cmd *cobra.Command, args []string) {
	var users []obj.User
	if err := client.ApiGet("users", &users); err != nil {
		log.Fatalf("Failed to fetch users: %v", err)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"ID", "Name", "Auth0 ID"})

	for _, u := range users {
		table.Append([]string{
			functional.MaybeToString(u.ID, "n/a"),
			functional.MaybeToString(u.Name, "n/a"),
			functional.MaybeToString(u.Auth0Id, "n/a"),
		})
	}
	table.Render()
}
