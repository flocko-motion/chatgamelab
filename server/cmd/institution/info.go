package institution

import (
	"cgl/api/client"
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

	var institution obj.Institution
	if err := client.ApiGet(fmt.Sprintf("institutions/%s", institutionID), &institution); err != nil {
		log.Fatalf("Failed to fetch institution: %v", err)
	}

	// Display institution information
	fmt.Println("Institution Information:")
	fmt.Printf("  ID:   %s\n", institution.ID)
	fmt.Printf("  Name: %s\n", institution.Name)

	// Display members (only included if user has permission)
	if len(institution.Members) > 0 {
		// Group members by role
		var heads []obj.InstitutionMember
		var staff []obj.InstitutionMember
		for _, member := range institution.Members {
			if member.Role == obj.RoleHead {
				heads = append(heads, member)
			} else if member.Role == obj.RoleStaff {
				staff = append(staff, member)
			}
		}

		// Display heads
		fmt.Printf("\nHeads (%d):\n", len(heads))
		if len(heads) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, member := range heads {
				email := "n/a"
				if member.Email != nil {
					email = *member.Email
				}
				fmt.Printf("  - %s (%s) - %s\n", member.Name, member.UserID, email)
			}
		}

		// Display staff
		fmt.Printf("\nStaff (%d):\n", len(staff))
		if len(staff) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, member := range staff {
				email := "n/a"
				if member.Email != nil {
					email = *member.Email
				}
				fmt.Printf("  - %s (%s) - %s\n", member.Name, member.UserID, email)
			}
		}
	}

	if institution.Meta.CreatedAt != nil {
		fmt.Printf("\nCreated: %s\n", institution.Meta.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	if institution.Meta.ModifiedAt != nil {
		fmt.Printf("Modified: %s\n", institution.Meta.ModifiedAt.Format("2006-01-02 15:04:05"))
	}
}
