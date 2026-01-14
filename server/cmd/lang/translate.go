package lang

import (
	"cgl/config"
	"cgl/functional"
	"cgl/game/ai"
	langutil "cgl/lang"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

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

		// Determine target languages
		var targetLangs []string
		if targetLang == "" {
			// No language specified - translate all supported languages
			targetLangs = langutil.GetAllLanguageCodes()
		} else {
			targetLangs = []string{targetLang}
		}

		// Determine output path
		if outputPath == "" {
			outputPath = "./assets/locales/"
			// Check if output directory exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error: Output directory does not exist: %s\n", outputPath)
				fmt.Fprintf(os.Stderr, "Please create it first: mkdir -p %s\n", outputPath)
				os.Exit(1)
			}
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

		// Translate all target languages
		fmt.Printf("Translating to %d language(s) using %s platform...\n", len(targetLangs), platform)

		for _, currentLang := range targetLangs {
			langName := langutil.GetLanguageName(currentLang)
			fmt.Printf("\n→ Translating to %s (%s)...\n", langName, currentLang)

			// Call the AI platform's Translate method
			translatedJSON, err := aiPlatform.Translate(cmd.Context(), apiKey, inputContents, currentLang)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ Translation failed: %v\n", err)
				continue
			}

			// Validate that translation has the same structure as original
			if err := functional.IsSameJsonStructure(inputContents[0], translatedJSON); err != nil {
				fmt.Fprintf(os.Stderr, "  WARNING: Translation has different structure than original: %v\n", err)
			} else {
				fmt.Println("  ✓ Translation structure validated")
			}

			// Write to file
			outputFile := filepath.Join(outputPath, currentLang+".json")
			if err := os.WriteFile(outputFile, []byte(translatedJSON), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ Failed to write output file: %v\n", err)
				continue
			}
			fmt.Printf("  ✓ Output written to: %s\n", outputFile)
		}

		fmt.Printf("\n✓ Translation complete for %d language(s)\n", len(targetLangs))
	},
}

func init() {
	Cmd.AddCommand(translateCmd)

	translateCmd.Flags().String("api-key", "", "API key for the platform")
	translateCmd.Flags().String("input", "../web/src/i18n/locales", "Path containing original language files (en.json, de.json)")
	translateCmd.Flags().String("output", "./assets/locales/", "Output directory for generated translation files")
	translateCmd.Flags().String("lang", "", "Target language code (e.g., 'fr', 'es', 'it'). If not specified, translates to all supported languages")
	translateCmd.Flags().String("platform", "mistral", "AI platform to use (openai, mock, mistral)")
	translateCmd.Flags().String("model", "mistral-large-latest", "AI model to use (e.g., 'mistral-large-latest', 'gpt-4o-mini')")
}
