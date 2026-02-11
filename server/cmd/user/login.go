package user

import (
	"cgl/api/client"
	"cgl/api/routes"
	"cgl/config"
	"cgl/functional"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

var (
	remoteURL   string
	remoteToken string
	localMode   bool
	aliasName   string
)

var loginCmd = &cobra.Command{
	Use:   "login [alias]",
	Short: "Configure server connection",
	Long: `Configure the CLI to connect to a ChatGameLab server.

Modes:
  1. Remote mode: Connect to a remote server with URL and JWT token
     Example: user login --url https://api.example.com --token <jwt> --alias prod
  
  2. Local mode: Generate JWT for local development server
     Example: user login --local [user-id] --alias dev
  
  3. Quick login: Use a saved server alias
     Example: user login prod
  
  4. Update JWT for existing alias: Provide alias and new JWT token
     Example: user login prod --token <new-jwt>`,
	Args: cobra.MaximumNArgs(1),
	Run:  runLogin,
}

func init() {
	Cmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&remoteURL, "url", "", "Remote server URL (e.g., http://localhost:8080)")
	loginCmd.Flags().StringVar(&remoteToken, "token", "", "JWT token for remote server")
	loginCmd.Flags().BoolVar(&localMode, "local", false, "Use local mode (generate JWT from local server)")
	loginCmd.Flags().StringVar(&aliasName, "alias", "", "Alias name for this server (saved for quick login)")
}

func runLogin(cmd *cobra.Command, args []string) {
	// Mode 1: Update JWT for existing alias (alias + --token)
	if len(args) > 0 && remoteToken != "" && remoteURL == "" && !localMode {
		runAliasUpdateJWT(args[0], remoteToken)
		return
	}

	// Mode 2: Quick login with alias
	if len(args) > 0 && remoteURL == "" && remoteToken == "" && !localMode {
		runAliasLogin(args[0])
		return
	}

	// Mode 3: Local mode
	if localMode {
		runLocalLogin(args)
		return
	}

	// Mode 4: Remote mode
	if remoteURL != "" && remoteToken != "" {
		runRemoteLogin()
		return
	}

	log.Fatal("Either provide an alias, use --local for local mode, or provide both --url and --token for remote mode")
}

func runAliasLogin(alias string) {
	server, err := config.GetKnownServerByAlias(alias)
	if err != nil {
		log.Fatalf("Failed to find server: %v", err)
	}

	if err := config.SetServerConfig(server.URL, server.JWT); err != nil {
		log.Fatalf("Failed to save configuration: %v", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("✓ Switched to server '%s'\n", alias)
	fmt.Printf("  Config: %s\n", configPath)
	fmt.Printf("  Server URL: %s\n", server.URL)
	fmt.Printf("  JWT: %s...\n", truncateToken(server.JWT))
}

func runAliasUpdateJWT(alias string, newToken string) {
	// Get existing server by alias
	server, err := config.GetKnownServerByAlias(alias)
	if err != nil {
		log.Fatalf("Failed to find server with alias '%s': %v", alias, err)
	}

	// Strip "Bearer " prefix if user included it
	token := strings.TrimSpace(strings.TrimPrefix(newToken, "Bearer "))

	// Update the server config with new JWT, keeping the same URL and alias
	if err := config.SetServerConfigWithAlias(server.URL, token, alias); err != nil {
		log.Fatalf("Failed to update configuration: %v", err)
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("✓ Updated JWT for server '%s'\n", alias)
	fmt.Printf("  Config: %s\n", configPath)
	fmt.Printf("  Server URL: %s\n", server.URL)
	fmt.Printf("  JWT: %s...\n", truncateToken(token))
}

func runRemoteLogin() {
	// Strip "Bearer " prefix if user included it
	token := strings.TrimSpace(strings.TrimPrefix(remoteToken, "Bearer "))
	url := strings.TrimSuffix(strings.TrimSuffix(remoteURL, "/"), "/api")

	// Save with alias if provided
	if aliasName != "" {
		if err := config.SetServerConfigWithAlias(url, token, aliasName); err != nil {
			log.Fatalf("Failed to save configuration: %v", err)
		}
	} else {
		if err := config.SetServerConfig(url, token); err != nil {
			log.Fatalf("Failed to save configuration: %v", err)
		}
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("✓ Configuration saved to %s\n", configPath)
	fmt.Printf("  Server URL: %s\n", url)
	if aliasName != "" {
		fmt.Printf("  Alias: %s\n", aliasName)
	}
	fmt.Printf("  JWT: %s...\n", truncateToken(token))
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

	// Save with alias if provided
	if aliasName != "" {
		if err := config.SetServerConfigWithAlias(localURL, resp.Token, aliasName); err != nil {
			log.Fatalf("Failed to save JWT: %v", err)
		}
	} else {
		if err := config.SetServerConfig(localURL, resp.Token); err != nil {
			log.Fatalf("Failed to save JWT: %v", err)
		}
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("✓ Configuration saved to %s\n", configPath)
	fmt.Printf("  Server URL: %s\n", localURL)
	if aliasName != "" {
		fmt.Printf("  Alias: %s\n", aliasName)
	}
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
