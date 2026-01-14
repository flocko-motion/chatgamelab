package lang

import (
	"cgl/functional"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify language files have matching structure",
	Long:  "Verifies that all language files in a directory have the same JSON structure as the English reference (en.json).",
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")

		if path == "" {
			fmt.Fprintf(os.Stderr, "Error: --path is required\n")
			os.Exit(1)
		}

		// Check if path exists
		if !fileExists(path) {
			fmt.Fprintf(os.Stderr, "Error: Path does not exist: %s\n", path)
			os.Exit(1)
		}

		// Read English reference file
		enPath := filepath.Join(path, "en.json")
		if !fileExists(enPath) {
			fmt.Fprintf(os.Stderr, "Error: English reference file not found: %s\n", enPath)
			os.Exit(1)
		}

		enContent, err := os.ReadFile(enPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading en.json: %v\n", err)
			os.Exit(1)
		}

		// Find all JSON files in directory
		files, err := filepath.Glob(filepath.Join(path, "*.json"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding JSON files: %v\n", err)
			os.Exit(1)
		}

		if len(files) == 0 {
			fmt.Fprintf(os.Stderr, "Error: No JSON files found in %s\n", path)
			os.Exit(1)
		}

		fmt.Printf("Verifying language files in: %s\n", path)
		fmt.Printf("Reference: en.json\n\n")

		allValid := true
		checkedCount := 0

		for _, file := range files {
			filename := filepath.Base(file)

			// Skip English file (it's the reference)
			if filename == "en.json" {
				continue
			}

			content, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("✗ %s: Failed to read file: %v\n", filename, err)
				allValid = false
				continue
			}

			if err := functional.IsSameJsonStructure(string(enContent), string(content)); err != nil {
				fmt.Printf("✗ %s: Structure mismatch - %v\n", filename, err)
				allValid = false
			} else {
				fmt.Printf("✓ %s: Structure matches\n", filename)
			}
			checkedCount++
		}

		fmt.Printf("\nChecked %d language file(s)\n", checkedCount)

		if allValid {
			fmt.Println("✓ All language files have matching structure")
		} else {
			fmt.Println("✗ Some language files have structural differences")
			os.Exit(1)
		}
	},
}

func init() {
	Cmd.AddCommand(verifyCmd)
	verifyCmd.Flags().String("path", "", "Path to directory containing language files")
}
