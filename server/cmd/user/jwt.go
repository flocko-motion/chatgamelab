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
	Args:  cobra.ExactArgs(1),
	Run:   runUserJwt,
}

func init() {
	Cmd.AddCommand(userJwtCmd)
}

func runUserJwt(cmd *cobra.Command, args []string) {
	var resp endpoints.UserJwtResponse
	if err := client.ApiGet("user/jwt?id="+args[0], &resp); err != nil {
		log.Fatalf("Failed to generate JWT: %v", err)
	}

	fmt.Printf("User ID: %s\n", resp.UserID)
	fmt.Printf("Auth0 ID: %s\n", resp.Auth0ID)
	fmt.Printf("\nJWT Token (valid for 24h):\n%s\n", resp.Token)
}
