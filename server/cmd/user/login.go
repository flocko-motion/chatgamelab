package user

import (
	"cgl/api/client"
	"cgl/api/routes"
	"cgl/config"
	"cgl/functional"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var (
	remoteURL   string
	remoteToken string
	localMode   bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure server connection",
	Long: `Configure the CLI to connect to a ChatGameLab server.

Two modes are available:
  1. Remote mode: Connect to a remote server with URL and JWT token
     Example: user login --url https://api.example.com --token <jwt>
  
  2. Local mode: Generate JWT for local development server
     Example: user login --local [user-id]`,
	Run: runLogin,
}

func init() {
	Cmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&remoteURL, "url", "", "Remote server URL (e.g., http://localhost:8080)")
	loginCmd.Flags().StringVar(&remoteToken, "token", "", "JWT token for remote server")
	loginCmd.Flags().BoolVar(&localMode, "local", false, "Use local mode (generate JWT from local server)")
}

func runLogin(cmd *cobra.Command, args []string) {
	if localMode {
		runLocalLogin(args)
	} else if remoteURL != "" && remoteToken != "" {
		runRemoteLogin()
	} else {
		log.Fatal("Either use --local for local mode, or provide both --url and --token for remote mode")
	}
}

func runRemoteLogin() {
	if err := config.SetServerConfig(remoteURL, remoteToken); err != nil {
		log.Fatalf("Failed to save configuration: %v", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("✓ Configuration saved to %s\n", configPath)
	fmt.Printf("  Server URL: %s\n", remoteURL)
	fmt.Printf("  JWT: %s...\n", truncateToken(remoteToken))
}

func runLocalLogin(args []string) {
	var userId string
	if len(args) > 0 {
		userId = args[0]
	} else {
		userId = "00000000-0000-0000-0000-000000000000"
		fmt.Println("Using dev user (00000000-0000-0000-0000-000000000000)")
	}

	port := functional.EnvOrDefault("PORT_BACKEND", "")
	if port == "" {
		log.Fatal("PORT_BACKEND environment variable is not set")
	}
	localURL := fmt.Sprintf("http://127.0.0.1:%s", port)

	if err := config.SetServerConfig(localURL, ""); err != nil {
		log.Fatalf("Failed to save configuration: %v", err)
	}

	var resp routes.UsersJwtResponse
	if err := client.ApiGet("users/"+userId+"/jwt", &resp); err != nil {
		log.Fatalf("Failed to generate JWT from local server: %v\nMake sure the server is running at %s", err, localURL)
	}

	if err := config.SetServerConfig(localURL, resp.Token); err != nil {
		log.Fatalf("Failed to save JWT: %v", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("✓ Configuration saved to %s\n", configPath)
	fmt.Printf("  Server URL: %s\n", localURL)
	fmt.Printf("  User ID: %s\n", resp.UserID)
	fmt.Printf("  Auth0 ID: %s\n", resp.Auth0ID)
	fmt.Printf("  JWT: %s...\n", truncateToken(resp.Token))
}

func truncateToken(token string) string {
	if len(token) > 20 {
		return token[:20]
	}
	return token
}
