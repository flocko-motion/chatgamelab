package apikey

import (
	"cgl/api/client"
	"cgl/functional"
	"cgl/obj"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	Long:  "Fetch and display all API keys for the current user.",
	Run:   runList,
}

func init() {
	Cmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	var keys []obj.ApiKeyShare
	if err := client.ApiGet("apikeys", &keys); err != nil {
		log.Fatalf("Failed to fetch API keys: %v", err)
	}

	if len(keys) == 0 {
		log.Println("No API keys found.")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Share ID", "Key ID", "Platform", "Name", "Owner", "Created", "Key"})

	for _, k := range keys {
		if k.ApiKey == nil {
			continue
		}

		created := "n/a"
		if k.Meta.CreatedAt != nil {
			created = k.Meta.CreatedAt.Format("2006-01-02")
		}

		table.Append([]string{
			k.ID.String(),
			k.ApiKey.ID.String(),
			k.ApiKey.Platform,
			functional.Shorten(k.ApiKey.Name, 15),
			functional.Shorten(k.ApiKey.UserName, 15),
			created,
			k.ApiKey.KeyShortened,
		})
	}
	table.Render()
}
