// package: institution / institution management CLI commands
// type:    cli
// job:     fetch an institution by UUID and print its details and grouped members
// limits:  read-only display; does not modify institutions or filter member visibility (-> api)
package institution

import (
	"cgl/api/client"
	"cgl/api/wire"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <uuid>",
	Short: "Get detailed information about an institution",
	Long:  "Fetch and display detailed information about an institution by UUID.",
	Args:  cobra.ExactArgs(1),
	Run:   runInfo,
}

func init() {
	Cmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) {
	institutionID, err := uuid.Parse(args[0])
	if err != nil {
		log.Fatalf("Invalid UUID: %v", err)
	}

	var institution wire.OrganisationView
	if err := client.ApiGet(fmt.Sprintf("institutions/%s", institutionID), &institution); err != nil {
		log.Fatalf("Failed to fetch institution: %v", err)
	}

	// Display institution information
	fmt.Println("Institution Information:")
	fmt.Printf("  ID:   %s\n", institution.ID)
	fmt.Printf("  Name: %s\n", institution.Name)

	// Display members (only those the caller may list)
	if len(institution.Members) > 0 {
		var heads, staff []wire.UserView
		for _, member := range institution.Members {
			switch roleOf(member) {
			case string(obj.RoleHead):
				heads = append(heads, member)
			case string(obj.RoleStaff):
				staff = append(staff, member)
			}
		}
		printMembers("Heads", heads)
		printMembers("Staff", staff)
	}

	fmt.Printf("\nCreated: %s\n", institution.Meta.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Modified: %s\n", institution.Meta.ModifiedAt.Format("2006-01-02 15:04:05"))
}

// roleOf is a member's role, empty where the caller does not see it.
func roleOf(member wire.UserView) string {
	if member.Role == nil {
		return ""
	}
	return member.Role.Role
}

// printMembers prints a group of members under its title.
func printMembers(title string, members []wire.UserView) {
	fmt.Printf("\n%s (%d):\n", title, len(members))
	if len(members) == 0 {
		fmt.Println("  (none)")
		return
	}
	for _, member := range members {
		email := "n/a"
		if member.Email != nil {
			email = *member.Email
		}
		fmt.Printf("  - %s (%s) - %s\n", member.Name, member.ID, email)
	}
}
