package lang

import (
	"cgl/config"
	"cgl/functional"
	"cgl/game/ai"
	langutil "cgl/lang"
	"cgl/obj"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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
		threads, _ := cmd.Flags().GetInt("threads")

		// Validate required flags
		if inputPath == "" {
			fmt.Fprintf(os.Stderr, "Error: input path is required\n")
			os.Exit(1)
		}

		// Determine target languages
		var targetLangs []string
		if targetLang == "" {
			// No language specified - translate all supported languages except source languages (en, de)
			allLangs := langutil.GetAllLanguageCodes()
			for _, lang := range allLangs {
				if lang != "en" && lang != "de" {
					targetLangs = append(targetLangs, lang)
				}
			}
		} else {
			targetLangs = []string{targetLang}
		}

		// Determine output path
		if outputPath == "" {
			outputPath = "./lang/locales/"
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
		aiPlatform, err := ai.GetAiPlatform(platform)
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
			sourceFiles := []string{"en.json", "de.json"}
			patched1, patched2, added1, added2, err := functional.SyncJsonStructures(inputContents[0], inputContents[1], "TODO: TRANSLATE")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to compare original language files: %v\n", err)
				os.Exit(1)
			}

			if len(added1) > 0 || len(added2) > 0 {
				// Write patched files back
				for i, content := range []string{patched1, patched2} {
					filePath := filepath.Join(inputPath, sourceFiles[i])
					if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
						fmt.Fprintf(os.Stderr, "Error writing patched %s: %v\n", sourceFiles[i], err)
						os.Exit(1)
					}
				}

				// Report what was added
				fmt.Fprintf(os.Stderr, "Error: Original language files have different structures.\n")
				fmt.Fprintf(os.Stderr, "The missing fields have been added with \"TODO: TRANSLATE\" placeholders.\n\n")
				if len(added1) > 0 {
					fmt.Fprintf(os.Stderr, "Added to %s (found in %s):\n", sourceFiles[0], sourceFiles[1])
					for _, field := range added1 {
						fmt.Fprintf(os.Stderr, "  + %s\n", field)
					}
				}
				if len(added2) > 0 {
					fmt.Fprintf(os.Stderr, "Added to %s (found in %s):\n", sourceFiles[1], sourceFiles[0])
					for _, field := range added2 {
						fmt.Fprintf(os.Stderr, "  + %s\n", field)
					}
				}
				fmt.Fprintf(os.Stderr, "\nPlease translate the TODO fields and run again.\n")
				os.Exit(1)
			}

			fmt.Println("✓ Original files have matching structure")

			// Check for leftover TODO placeholders in source files
			hasPlaceholders := false
			for i, content := range inputContents {
				fields, err := functional.FindPlaceholders(content, "TODO: TRANSLATE")
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error scanning %s for placeholders: %v\n", sourceFiles[i], err)
					os.Exit(1)
				}
				if len(fields) > 0 {
					if !hasPlaceholders {
						fmt.Fprintf(os.Stderr, "Error: Source files contain untranslated TODO placeholders.\n\n")
						hasPlaceholders = true
					}
					fmt.Fprintf(os.Stderr, "%s:\n", sourceFiles[i])
					for _, field := range fields {
						fmt.Fprintf(os.Stderr, "  - %s\n", field)
					}
				}
			}
			if hasPlaceholders {
				fmt.Fprintf(os.Stderr, "\nPlease translate the TODO fields and run again.\n")
				os.Exit(1)
			}
			fmt.Println("✓ No TODO placeholders found")
		}

		// Compute source hash from en+de contents
		sourceHash := functional.ComputeHash(inputContents...)
		fmt.Printf("Source hash: %s\n", sourceHash)

		// Translate all target languages in parallel
		if threads > 0 {
			fmt.Printf("Translating to %d language(s) using %s platform in %d threads...\n", len(targetLangs), platform, threads)
		} else {
			fmt.Printf("Translating to %d language(s) using %s platform (unlimited parallelism)...\n", len(targetLangs), platform)
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		successCount := 0

		// Semaphore to limit concurrent translations (0 = unlimited)
		var semaphore chan struct{}
		if threads > 0 {
			semaphore = make(chan struct{}, threads)
		}

		// Pre-check which languages need translating
		var langsToTranslate []string
		for _, lang := range targetLangs {
			outputFile := filepath.Join(outputPath, lang+".json")
			if existingContent, err := os.ReadFile(outputFile); err == nil {
				if existingHash, err := functional.ReadJsonField(string(existingContent), "_sourceHash"); err == nil && existingHash == sourceHash {
					successCount++
					fmt.Printf("⏭ %s (%s): up-to-date, skipping\n", langutil.GetLanguageName(lang), lang)
					continue
				}
			}
			langsToTranslate = append(langsToTranslate, lang)
		}

		if len(langsToTranslate) == 0 {
			fmt.Printf("\n✓ All %d translations are up-to-date\n", len(targetLangs))
			return
		}
		fmt.Printf("\n⏳ Translating %d language(s)...\n\n", len(langsToTranslate))

		var totalUsage obj.TokenUsage

		for _, currentLang := range langsToTranslate {
			wg.Add(1)

			// Launch goroutine for each translation
			go func(lang string) {
				defer wg.Done()

				// Acquire semaphore slot if throttling is enabled
				if semaphore != nil {
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
				}

				langName := langutil.GetLanguageName(lang)
				outputFile := filepath.Join(outputPath, lang+".json")

				// Retry logic: up to 3 attempts
				var translatedJSON string
				var err error
				maxRetries := 3

				for attempt := 1; attempt <= maxRetries; attempt++ {
					// Call the AI platform's Translate method
					var usage obj.TokenUsage
					translatedJSON, usage, err = aiPlatform.Translate(cmd.Context(), apiKey, inputContents, lang)
					if err == nil {
						mu.Lock()
						totalUsage = totalUsage.Add(usage)
						mu.Unlock()
						break // Success, exit retry loop
					}

					// Failed - print error and retry if attempts remain
					mu.Lock()
					if attempt < maxRetries {
						fmt.Fprintf(os.Stderr, "⚠ %s (%s): Attempt %d/%d failed: %v - retrying...\n", langName, lang, attempt, maxRetries, err)
					} else {
						fmt.Fprintf(os.Stderr, "✗ %s (%s): All %d attempts failed: %v\n", langName, lang, maxRetries, err)
					}
					mu.Unlock()
				}

				// If all retries failed, exit
				if err != nil {
					return
				}

				// Validate that translation has the same structure as original
				if err := functional.IsSameJsonStructure(inputContents[0], translatedJSON); err != nil {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "⚠ %s (%s): Translation has different structure than original: %v\n", langName, lang, err)
					mu.Unlock()
				}

				// Inject source hash into translated JSON
				translatedJSON, err = functional.InjectJsonField(translatedJSON, "_sourceHash", sourceHash)
				if err != nil {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "✗ %s (%s): Failed to inject source hash: %v\n", langName, lang, err)
					mu.Unlock()
					return
				}

				// Write to file
				if err := os.WriteFile(outputFile, []byte(translatedJSON), 0644); err != nil {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "✗ %s (%s): Failed to write output file: %v\n", langName, lang, err)
					mu.Unlock()
					return
				}

				// Success - print with lock
				mu.Lock()
				successCount++
				fmt.Printf("✓ %s (%s) → %s\n", langName, lang, outputFile)
				mu.Unlock()
			}(currentLang)
		}

		// Wait for all translations to complete
		wg.Wait()

		fmt.Printf("\n✓ Translation complete: %d/%d successful (tokens: %d in, %d out, %d total)\n",
			successCount, len(targetLangs), totalUsage.InputTokens, totalUsage.OutputTokens, totalUsage.TotalTokens)
	},
}

func init() {
	Cmd.AddCommand(translateCmd)

	translateCmd.Flags().String("api-key", "", "API key for the platform")
	translateCmd.Flags().String("input", "../web/src/i18n/locales", "Path containing original language files (en.json, de.json)")
	translateCmd.Flags().String("output", "./lang/locales/", "Output directory for generated translation files")
	translateCmd.Flags().String("lang", "", "Target language code (e.g., 'fr', 'es', 'it'). If not specified, translates to all supported languages")
	translateCmd.Flags().String("platform", "openai", "AI platform to use (openai, mock, mistral)")
	translateCmd.Flags().Int("threads", 0, "Number of parallel translation threads (0 = unlimited)")
}
