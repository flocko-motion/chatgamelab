package invite

import (
	"cgl/api/client"
	"cgl/api/routes"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var workshopCmd = &cobra.Command{
	Use:   "workshop <workshopId> [--max-uses=<n>] [--expires-at=<RFC3339>]",
	Short: "Create a workshop invite",
	Long:  "Create an open invite for users to join a workshop as participants.",
	Args:  cobra.ExactArgs(1),
	Run:   runWorkshopInvite,
}

var (
	maxUses   int32
	expiresAt string
)

func init() {
	Cmd.AddCommand(workshopCmd)
	workshopCmd.Flags().Int32Var(&maxUses, "max-uses", 0, "Maximum number of uses (0 = unlimited)")
	workshopCmd.Flags().StringVar(&expiresAt, "expires-at", "", "Expiration date (RFC3339 format)")
}

func runWorkshopInvite(cmd *cobra.Command, args []string) {
	workshopID := args[0]

	req := routes.CreateWorkshopInviteRequest{
		WorkshopID: workshopID,
	}

	if maxUses > 0 {
		req.MaxUses = &maxUses
	}
	if expiresAt != "" {
		req.ExpiresAt = &expiresAt
	}

	var result map[string]interface{}
	if err := client.ApiPost("invites/workshop", req, &result); err != nil {
		log.Fatalf("Failed to create workshop invite: %v", err)
	}

	fmt.Printf("✓ Workshop invite created successfully\n")
	fmt.Printf("  Invite ID:   %v\n", result["id"])
	fmt.Printf("  Workshop ID: %s\n", workshopID)
	fmt.Printf("  Role:        participant\n")
	if maxUses > 0 {
		fmt.Printf("  Max Uses:    %d\n", maxUses)
	} else {
		fmt.Printf("  Max Uses:    unlimited\n")
	}
	if expiresAt != "" {
		fmt.Printf("  Expires At:  %s\n", expiresAt)
	}
	if token, ok := result["invite_token"].(map[string]interface{}); ok {
		if tokenStr, ok := token["String"].(string); ok && tokenStr != "" {
			fmt.Printf("  Token:       %s\n", tokenStr)
			fmt.Printf("\nAccept URL: /api/invites/workshop/%s/accept\n", tokenStr)
		}
	}
}
