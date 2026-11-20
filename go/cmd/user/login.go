// package: user / user management CLI commands
// type:    cli
// job:     configure the CLI's server connection via remote, alias, or JWT-update modes
// limits:  writes only local CLI config; does not manage server-side accounts (-> config)
package user

import (
	"cgl/config"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

var (
	remoteURL   string
	remoteToken string
	aliasName   string
)

var loginCmd = &cobra.Command{
	Use:   "login [alias]",
	Short: "Configure server connection",
	Long: `Configure the CLI to connect to a ChatGameLab server.

Modes:
  1. Remote mode: Connect to a server with URL and token
     Example: user login --url https://api.example.com --token <token> --alias prod
     In development, mint the token with: ./schemaf.sh auth token <user-id>
  
  2. Quick login: Use a saved server alias
     Example: user login prod
  
  3. Update JWT for existing alias: Provide alias and new JWT token
     Example: user login prod --token <new-jwt>`,
	Args: cobra.MaximumNArgs(1),
	Run:  runLogin,
}

func init() {
	Cmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&remoteURL, "url", "", "Remote server URL (e.g., http://localhost:8080)")
	loginCmd.Flags().StringVar(&remoteToken, "token", "", "JWT token for remote server")
	loginCmd.Flags().StringVar(&aliasName, "alias", "", "Alias name for this server (saved for quick login)")
}

func runLogin(cmd *cobra.Command, args []string) {
	// Update the token of a saved alias (alias + --token)
	if len(args) > 0 && remoteToken != "" && remoteURL == "" {
		runAliasUpdateJWT(args[0], remoteToken)
		return
	}

	// Quick login with a saved alias
	if len(args) > 0 && remoteURL == "" && remoteToken == "" {
		runAliasLogin(args[0])
		return
	}

	// Remote mode
	if remoteURL != "" && remoteToken != "" {
		runRemoteLogin()
		return
	}

	log.Fatal("Either provide an alias, or both --url and --token")
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

func truncateToken(token string) string {
	if len(token) > 20 {
		return token[:20]
	}
	return token
}
