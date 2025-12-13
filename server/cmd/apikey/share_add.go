package apikey

import (
	"cgl/api/client"
	"cgl/api/endpoints"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var shareAddCmd = &cobra.Command{
	Use:   "add <share-id>",
	Short: "Share an API key with others",
	Long:  "Share an API key (using your share ID) with a user, workshop, or institution.",
	Args:  cobra.ExactArgs(1),
	Run:   runShareAdd,
}

var (
	shareUserID        string
	shareWorkshopID    string
	shareInstitutionID string
	shareAllowPublic   bool
)

func init() {
	shareAddCmd.Flags().StringVar(&shareUserID, "user-id", "", "User ID to share with")
	shareAddCmd.Flags().StringVar(&shareWorkshopID, "workshop-id", "", "Workshop ID to share with")
	shareAddCmd.Flags().StringVar(&shareInstitutionID, "institution-id", "", "Institution ID to share with")
	shareAddCmd.Flags().BoolVar(&shareAllowPublic, "allow-public", false, "Allow public sponsored plays")
	shareCmd.AddCommand(shareAddCmd)
}

func runShareAdd(cmd *cobra.Command, args []string) {
	apiKeyID := args[0]

	if shareUserID == "" && shareWorkshopID == "" && shareInstitutionID == "" {
		log.Fatalf("At least one of --user-id, --workshop-id, or --institution-id is required")
	}

	req := endpoints.ShareRequest{
		AllowPublic: shareAllowPublic,
	}

	if shareUserID != "" {
		id, err := uuid.Parse(shareUserID)
		if err != nil {
			log.Fatalf("Invalid user ID: %v", err)
		}
		req.UserID = &id
	}
	if shareWorkshopID != "" {
		id, err := uuid.Parse(shareWorkshopID)
		if err != nil {
			log.Fatalf("Invalid workshop ID: %v", err)
		}
		req.WorkshopID = &id
	}
	if shareInstitutionID != "" {
		id, err := uuid.Parse(shareInstitutionID)
		if err != nil {
			log.Fatalf("Invalid institution ID: %v", err)
		}
		req.InstitutionID = &id
	}

	var resp endpoints.ShareResponse
	if err := client.ApiPost("apikeys/"+apiKeyID+"/shares", req, &resp); err != nil {
		log.Fatalf("Failed to add share: %v", err)
	}

	fmt.Printf("Share created: %s\n", resp.ID)
}
