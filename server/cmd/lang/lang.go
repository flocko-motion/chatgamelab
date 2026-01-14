package lang

import (
	"cgl/config"
	"cgl/functional"
	"cgl/game/ai"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Cmd is the lang subcommand
var Cmd = &cobra.Command{
	Use:   "lang",
	Short: "Language translation helper commands",
	Long:  "Commands for translating language files using AI.",
}

var translateCmd = &cobra.Command{
	Use:   "translate",
	Short: "Translate language files using AI",
	Long: `Translate language files from English and German to other languages using AI.
	
The API key will be read from ~/.chatgamelab/config.yaml unless --api-key <key>is provided.
The --api-key flag accepts only the key itself, NOT a path to a file containing the key.

Supported platforms:
- openai: Requires OpenAI API key
- mock: Testing platform that generates mock translations (no API key needed)
- mistral: Requires Mistral API key`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyFlag, _ := cmd.Flags().GetString("api-key")
		inputPath, _ := cmd.Flags().GetString("input")
		outputPath, _ := cmd.Flags().GetString("output")
		targetLang, _ := cmd.Flags().GetString("lang")
		platform, _ := cmd.Flags().GetString("platform")
		model, _ := cmd.Flags().GetString("model")

		// Validate required flags
		if inputPath == "" {
			fmt.Fprintf(os.Stderr, "Error: input path is required\n")
			os.Exit(1)
		}
		if outputPath == "" {
			fmt.Fprintf(os.Stderr, "Error: output path is required\n")
			os.Exit(1)
		}
		if targetLang == "" {
			fmt.Fprintf(os.Stderr, "Error: target language is required\n")
			os.Exit(1)
		}

		// Get API key from flag or config
		apiKey, err := config.GetApiKey(platform, apiKeyFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Get AI platform
		aiPlatform, _, err := ai.GetAiPlatform(platform, model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting AI platform: %v\n", err)
			os.Exit(1)
		}

		// Find and read input files (en.json and de.json)
		inputContents := []string{}
		for _, filename := range []string{"en.json", "de.json"} {
			filePath := filepath.Join(inputPath, filename)
			if !fileExists(filePath) {
				continue
			}
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filename, err)
				os.Exit(1)
			}
			inputContents = append(inputContents, string(content))
		}

		if len(inputContents) == 0 {
			fmt.Fprintf(os.Stderr, "Error: No language files found in input path. Expected en.json and/or de.json\n")
			os.Exit(1)
		}

		// Validate that original files have the same structure
		if len(inputContents) == 2 {
			if err := functional.IsSameJsonStructure(inputContents[0], inputContents[1]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Original language files have different structures: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Original files have matching structure")
		}

		// Ensure output directory exists
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
			os.Exit(1)
		}

		// Call the AI platform's Translate method
		fmt.Printf("Translating using %s platform...\n", platform)
		translatedJSON, err := aiPlatform.Translate(cmd.Context(), apiKey, inputContents, targetLang)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Translation failed: %v\n", err)
			os.Exit(1)
		}

		// Write translated JSON to output file
		if err := os.WriteFile(outputPath, []byte(translatedJSON), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write output file: %v\n", err)
			os.Exit(1)
		}

		// Validate that translation has the same structure as original
		if err := functional.IsSameJsonStructure(inputContents[0], translatedJSON); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Translation has different structure than original: %v\n", err)
		}
		fmt.Println("✓ Translation structure validated")

		fmt.Printf("Output written to: %s\n", outputPath)
	},
}

func init() {
	// Add translate subcommand
	Cmd.AddCommand(translateCmd)

	// Add flags for translate command
	translateCmd.Flags().String("api-key", "", "API key for the platform")
	translateCmd.Flags().String("input", "", "Path containing original language files (en.json, de.json)")
	translateCmd.Flags().String("output", "", "Output path for generated translation files")
	translateCmd.Flags().String("lang", "", "Target language code (e.g., 'fr', 'es', 'it')")
	translateCmd.Flags().String("platform", "openai", "AI platform to use (openai, mock, mistral)")
	translateCmd.Flags().String("model", "", "AI model to use (e.g., 'gpt-4o-mini', 'gpt-4o'). Defaults to platform default")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
