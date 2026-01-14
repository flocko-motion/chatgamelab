package ai

import (
	"fmt"
	"os"

	"cgl/config"
	"cgl/game/ai"

	"github.com/spf13/cobra"
)

// Cmd is the ai subcommand
var Cmd = &cobra.Command{
	Use:   "ai",
	Short: "AI platform management commands",
	Long:  "Commands for managing AI platforms and models.",
}

var modelsCmd = &cobra.Command{
	Use:   "models [platform]",
	Short: "List available models for a platform",
	Long: `List all available models for a specific AI platform.
If platform is not specified, lists models for all platforms.

The API key will be read from ~/.chatgamelab/config.yaml unless --api-key is provided.`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyFlag, _ := cmd.Flags().GetString("api-key")
		platform, _ := cmd.Flags().GetString("platform")

		// If no platform specified, list all
		if platform == "" {
			if len(args) > 0 {
				platform = args[0]
			} else {
				// List all platforms
				platformInfos := ai.GetAiPlatformInfos()
				for _, info := range platformInfos {
					fmt.Printf("\n=== %s ===\n", info.Name)
					for _, model := range info.Models {
						fmt.Printf("  %s: %s\n", model.ID, model.Description)
					}
				}
				fmt.Printf("\nUse 'cgl ai models <platform>' to query live models from API\n")
				return
			}
		}

		// Get API key from flag or config
		apiKey, err := config.GetApiKey(platform, apiKeyFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Get AI platform
		aiPlatform, _, err := ai.GetAiPlatform(platform, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting AI platform: %v\n", err)
			os.Exit(1)
		}

		// List models
		fmt.Printf("Querying models for %s...\n\n", platform)
		models, err := aiPlatform.ListModels(cmd.Context(), apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing models: %v\n", err)
			os.Exit(1)
		}

		if len(models) == 0 {
			fmt.Printf("No models found for platform %s\n", platform)
			return
		}

		for _, model := range models {
			fmt.Printf("  %s: %s\n", model.ID, model.Description)
		}
		fmt.Printf("\nFound %d models\n", len(models))
	},
}

func init() {
	// Add models subcommand
	Cmd.AddCommand(modelsCmd)

	// Add flags
	modelsCmd.Flags().String("api-key", "", "API key for the platform")
	modelsCmd.Flags().String("platform", "", "AI platform (openai, mistral, mock)")
}
