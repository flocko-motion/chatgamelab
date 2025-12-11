package user

import (
	"cgl/api/client"
	"cgl/api/endpoints"
	"cgl/obj"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var userAddCmd = &cobra.Command{
	Use:   "add <name> [email]",
	Short: "Add a new user without Auth0",
	Long:  "Create a new user in the database without Auth0 authentication. Useful for development.",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runUserAdd,
}

func init() {
	Cmd.AddCommand(userAddCmd)
}

func runUserAdd(cmd *cobra.Command, args []string) {
	req := endpoints.UserAddRequest{Name: args[0]}
	if len(args) > 1 {
		req.Email = &args[1]
	}

	var user obj.User
	if err := client.ApiPost("user/add", req, &user); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("Created user:\n")
	fmt.Printf("  ID:    %s\n", user.ID)
	fmt.Printf("  Name:  %s\n", user.Name)
	if user.Email != nil {
		fmt.Printf("  Email: %s\n", *user.Email)
	}
	fmt.Printf("\nTo generate a JWT for this user:\n")
	fmt.Printf("  go run . user jwt %s\n", user.ID)
}
