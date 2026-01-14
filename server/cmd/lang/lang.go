package lang

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Cmd is the lang subcommand
var Cmd = &cobra.Command{
	Use:   "lang",
	Short: "Language translation helper commands",
	Long:  "Commands for translating language files using OpenAI API.",
}

var translateCmd = &cobra.Command{
	Use:   "translate",
	Short: "Translate language files using OpenAI API",
	Long: `Translate language files from English and German to other languages using OpenAI API.
	
The API key can be provided either as an environment variable (OPENAI_API_KEY) 
or as a file path containing the key.`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKey, _ := cmd.Flags().GetString("api-key")
		apiKeyFile, _ := cmd.Flags().GetString("api-key-file")
		inputPath, _ := cmd.Flags().GetString("input")
		outputPath, _ := cmd.Flags().GetString("output")
		targetLang, _ := cmd.Flags().GetString("lang")

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

		// Get API key from flag, env var, or file
		var key string
		if apiKey != "" {
			key = apiKey
		} else if apiKeyFile != "" {
			fileKey, err := os.ReadFile(apiKeyFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading API key file: %v\n", err)
				os.Exit(1)
			}
			key = string(fileKey)
		} else if key = os.Getenv("OPENAI_API_KEY"); key == "" {
			fmt.Fprintf(os.Stderr, "Error: OpenAI API key is required. Use --api-key, --api-key-file, or OPENAI_API_KEY environment variable\n")
			os.Exit(1)
		}

		// TODO: Implement the actual translation logic
		fmt.Printf("Translation scaffolding:\n")
		fmt.Printf("  API Key: %s...\n", key[:min(len(key), 10)])
		fmt.Printf("  Input Path: %s\n", inputPath)
		fmt.Printf("  Output Path: %s\n", outputPath)
		fmt.Printf("  Target Language: %s\n", targetLang)
		fmt.Printf("\nTODO: Implement actual translation logic\n")
	},
}

func init() {
	// Add translate subcommand
	Cmd.AddCommand(translateCmd)

	// Add flags for translate command
	translateCmd.Flags().String("api-key", "", "OpenAI API key")
	translateCmd.Flags().String("api-key-file", "", "Path to file containing OpenAI API key")
	translateCmd.Flags().String("input", "", "Path containing original language files (en.json, de.json)")
	translateCmd.Flags().String("output", "", "Output path for generated translation files")
	translateCmd.Flags().String("lang", "", "Target language code (e.g., 'fr', 'es', 'it')")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
