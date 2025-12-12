package apikey

import (
	"cgl/api/client"
	"cgl/obj"
	"fmt"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <share-id>",
	Short: "Show API key info and linked shares",
	Long:  "Display detailed information about an API key share, including all linked shares if you are the owner.",
	Args:  cobra.ExactArgs(1),
	Run:   runInfo,
}

func init() {
	Cmd.AddCommand(infoCmd)
}

type infoResponse struct {
	Share        *obj.ApiKeyShare  `json:"share"`
	LinkedShares []obj.ApiKeyShare `json:"linkedShares"`
}

func runInfo(cmd *cobra.Command, args []string) {
	shareID := args[0]

	var resp infoResponse
	if err := client.ApiGet("apikeys/"+shareID, &resp); err != nil {
		log.Fatalf("Failed to fetch API key info: %v", err)
	}

	if resp.Share == nil || resp.Share.ApiKey == nil {
		log.Fatalf("Share not found")
	}

	k := resp.Share.ApiKey
	fmt.Printf("API Key Info:\n")
	fmt.Printf("  Name:     %s\n", k.Name)
	fmt.Printf("  Platform: %s\n", k.Platform)
	fmt.Printf("  Key:      %s\n", k.KeyShortened)
	fmt.Printf("  Owner:    %s\n", k.UserName)

	if len(resp.LinkedShares) > 0 {
		fmt.Printf("\nLinked Shares:\n")
		table := tablewriter.NewWriter(os.Stdout)
		table.Header([]string{"Share ID", "Type", "Target", "Public Plays"})

		for _, s := range resp.LinkedShares {
			shareType := ""
			target := ""

			if s.User != nil {
				shareType = "user"
				target = s.User.Name
				if s.User.ID == k.UserID {
					target += " (owner)"
				}
			} else if s.Workshop != nil {
				shareType = "workshop"
				target = s.Workshop.Name
			} else if s.Institution != nil {
				shareType = "institution"
				target = s.Institution.Name
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
			})
		}
		table.Render()
	}
}
