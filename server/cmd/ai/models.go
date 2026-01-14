package ai

import (
	"cgl/config"
	"cgl/game/ai"
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models [platform]",
	Short: "List available models for a platform",
	Long: `List all available models for a specific AI platform.
If platform is not specified, lists models for all platforms.

The API key will be read from ~/.chatgamelab/config.yaml unless --api-key is provided.`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyFlag, _ := cmd.Flags().GetString("api-key")
		platform, _ := cmd.Flags().GetString("platform")
		showAll, _ := cmd.Flags().GetBool("all")

		// If no platform specified, list all
		if platform == "" {
			if len(args) > 0 {
				platform = args[0]
			} else {
				// List all platforms
				fmt.Println("List of hardcoded models configured for production:")
				fmt.Println()
				platformInfos := ai.GetAiPlatformInfos()
				for _, info := range platformInfos {
					fmt.Printf("=== %s ===\n", info.Name)
					for _, model := range info.Models {
						fmt.Printf("  %s: %s\n", model.ID, model.Description)
					}
					fmt.Println()
				}
				fmt.Printf("Use 'cgl ai models <platform>' to query live models from API\n")
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
		if showAll {
			fmt.Printf("Querying ALL models for %s (no filtering)...\n\n", platform)
		} else {
			fmt.Printf("Querying models for %s...\n\n", platform)
		}

		// Store showAll in context for the platform to use
		ctx := cmd.Context()
		if showAll {
			ctx = context.WithValue(ctx, "showAll", true)
		}

		models, err := aiPlatform.ListModels(ctx, apiKey)
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

		if showAll {
			fmt.Printf("\nFound %d models (unfiltered)\n", len(models))
		} else {
			fmt.Printf("\nFound %d relevant models (removed non-chat, legacy, preview, ..)\n", len(models))
		}
	},
}

func init() {
	Cmd.AddCommand(modelsCmd)

	modelsCmd.Flags().String("api-key", "", "API key for the platform")
	modelsCmd.Flags().String("platform", "", "AI platform (openai, mistral, mock)")
	modelsCmd.Flags().Bool("all", false, "Show all models without filtering")
}
