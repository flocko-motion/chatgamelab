package user

import (
	"cgl/api/client"
	"cgl/api/endpoints"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var userJwtCmd = &cobra.Command{
	Use:   "jwt <id>",
	Short: "Generate a JWT token for a user",
	Long:  "Generate a JWT token for a user by UUID or Auth0 ID. Useful for development/testing.",
	Args:  cobra.RangeArgs(0, 1),
	Run:   runUserJwt,
}

func init() {
	Cmd.AddCommand(userJwtCmd)
}

func runUserJwt(cmd *cobra.Command, args []string) {
	var resp endpoints.UserJwtResponse

	var userId string
	if len(args) > 0 {
		userId = args[0]
	} else {
		userId = "00000000-0000-0000-0000-000000000000" // dev user
		fmt.Println("Creating JWT for dev user")
	}

	if err := client.ApiGet("user/jwt?id="+userId, &resp); err != nil {
		log.Fatalf("Failed to generate JWT: %v", err)
	}

	if err := client.SaveJwt(resp.Token); err != nil {
		log.Fatalf("Failed to save JWT: %v", err)
	}

	fmt.Printf("User ID: %s\n", resp.UserID)
	fmt.Printf("Auth0 ID: %s\n", resp.Auth0ID)
	fmt.Printf("JWT: %s\n", resp.Token)
	fmt.Printf("\nJWT Token saved to %s\n", client.GetJwtPath())
}
