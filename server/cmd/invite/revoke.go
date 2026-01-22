package invite

import (
	"cgl/api/client"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var revokeCmd = &cobra.Command{
	Use:   "revoke <inviteId>",
	Short: "Revoke an invite",
	Long:  "Revoke a pending invite (creator or admin only).",
	Args:  cobra.ExactArgs(1),
	Run:   runRevokeInvite,
}

func init() {
	Cmd.AddCommand(revokeCmd)
}

func runRevokeInvite(cmd *cobra.Command, args []string) {
	inviteID := args[0]

	if err := client.ApiDelete(fmt.Sprintf("invites/%s", inviteID)); err != nil {
		log.Fatalf("Failed to revoke invite: %v", err)
	}

	fmt.Printf("✓ Invite revoked successfully\n")
	fmt.Printf("  Invite ID: %s\n", inviteID)
}
