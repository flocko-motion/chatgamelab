// package: invite / invite management CLI commands
// type:    cli
// job:     create a targeted institution invite for a user (by id or email) with a role
// limits:  only posts the invite request and prints the result; server issues the token (-> api/routes)
package invite

import (
	"cgl/api/client"
	"cgl/api/routes"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var institutionCmd = &cobra.Command{
	Use:   "institution <institutionId> <role> [--user-id=<userId>] [--email=<email>]",
	Short: "Create an institution invite",
	Long:  "Create a targeted invite for a user to join an institution as head or staff.",
	Args:  cobra.ExactArgs(2),
	Run:   runInstitutionInvite,
}

var (
	invitedUserID string
	invitedEmail  string
)

func init() {
	Cmd.AddCommand(institutionCmd)
	institutionCmd.Flags().StringVar(&invitedUserID, "user-id", "", "User ID to invite")
	institutionCmd.Flags().StringVar(&invitedEmail, "email", "", "Email address to invite")
}

func runInstitutionInvite(cmd *cobra.Command, args []string) {
	institutionID := args[0]
	role := args[1]

	if invitedUserID == "" && invitedEmail == "" {
		log.Fatal("Either --user-id or --email must be provided")
	}

	req := routes.CreateInstitutionInviteRequest{
		InstitutionID: institutionID,
		Role:          role,
	}

	if invitedUserID != "" {
		req.InvitedUserID = &invitedUserID
	}
	if invitedEmail != "" {
		req.InvitedEmail = &invitedEmail
	}

	var result map[string]interface{}
	if err := client.ApiPost("invites/institution", req, &result); err != nil {
		log.Fatalf("Failed to create institution invite: %v", err)
	}

	fmt.Printf("✓ Institution invite created successfully\n")
	fmt.Printf("  Invite ID:      %v\n", result["id"])
	fmt.Printf("  Institution ID: %s\n", institutionID)
	fmt.Printf("  Role:           %s\n", role)
	if invitedUserID != "" {
		fmt.Printf("  Invited User:   %s\n", invitedUserID)
	}
	if invitedEmail != "" {
		fmt.Printf("  Invited Email:  %s\n", invitedEmail)
	}
	if token, ok := result["invite_token"].(map[string]interface{}); ok {
		if tokenStr, ok := token["String"].(string); ok && tokenStr != "" {
			fmt.Printf("  Token:          %s\n", tokenStr)
			fmt.Printf("\nAccept URL: /api/invites/institution/%s/accept\n", tokenStr)
		}
	}
}
