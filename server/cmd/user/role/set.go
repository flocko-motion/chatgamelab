package role

import (
	"cgl/api/client"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var (
	institutionFlag string
	workshopFlag    string
)

var setCmd = &cobra.Command{
	Use:   "set <user-id> <role>",
	Short: "Set a user's role",
	Long: `Set a user's role in the system.

Role types:
  - admin: System administrator (no institution/workshop required)
  - head: Institution head (requires --institution)
  - staff: Institution staff (requires --institution)
  - participant: Workshop participant (requires --workshop)

Examples:
  # Make user an admin
  user role set <user-id> admin

  # Make user a head of an institution
  user role set <user-id> head --institution <institution-id>

  # Make user staff of an institution
  user role set <user-id> staff --institution <institution-id>

  # Make user a participant in a workshop
  user role set <user-id> participant --workshop <workshop-id>`,
	Args: cobra.ExactArgs(2),
	Run:  runSet,
}

func init() {
	setCmd.Flags().StringVar(&institutionFlag, "institution", "", "Institution ID (required for head/staff)")
	setCmd.Flags().StringVar(&workshopFlag, "workshop", "", "Workshop ID (required for participant)")

	Cmd.AddCommand(setCmd)
}

func runSet(cmd *cobra.Command, args []string) {
	userID, err := uuid.Parse(args[0])
	if err != nil {
		log.Fatalf("Invalid user ID: %v", err)
	}

	role := args[1]

	// Validate role
	validRoles := map[string]bool{
		"admin":       true,
		"head":        true,
		"staff":       true,
		"participant": true,
	}
	if !validRoles[role] {
		log.Fatalf("Invalid role: %s (must be admin, head, staff, or participant)", role)
	}

	// Validate required flags based on role
	if role == "head" || role == "staff" {
		if institutionFlag == "" {
			log.Fatalf("--institution is required for %s role", role)
		}
		if _, err := uuid.Parse(institutionFlag); err != nil {
			log.Fatalf("Invalid institution ID: %v", err)
		}
	}

	if role == "participant" {
		if workshopFlag == "" {
			log.Fatalf("--workshop is required for participant role")
		}
		if _, err := uuid.Parse(workshopFlag); err != nil {
			log.Fatalf("Invalid workshop ID: %v", err)
		}
	}

	// Build request body
	body := map[string]interface{}{
		"role": role,
	}
	if institutionFlag != "" {
		body["institutionId"] = institutionFlag
	}
	if workshopFlag != "" {
		body["workshopId"] = workshopFlag
	}

	// Make API request
	if err := client.ApiPost(fmt.Sprintf("users/%s/role", userID), body, nil); err != nil {
		log.Fatalf("Failed to set user role: %v", err)
	}

	fmt.Printf("✓ Role set successfully\n")
	fmt.Printf("  User ID: %s\n", userID)
	fmt.Printf("  Role:    %s\n", role)
	if institutionFlag != "" {
		fmt.Printf("  Institution: %s\n", institutionFlag)
	}
	if workshopFlag != "" {
		fmt.Printf("  Workshop: %s\n", workshopFlag)
	}
}
