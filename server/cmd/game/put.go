package game

import (
	"cgl/api/client"
	"cgl/functional"
	"cgl/obj"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:   "put <yaml-file-or-directory> [game-id]",
	Short: "Create or update a game from YAML",
	Long:  "Create a new game or update an existing game from YAML file or directory. If a directory is provided, all .yaml files in it will be uploaded. If game-id is provided, updates that game. Otherwise creates new games. Use --stdin to read from stdin instead of file.",
	Args:  cobra.RangeArgs(1, 2),
	Run:   runPut,
}

var useStdin bool

func init() {
	putCmd.Flags().BoolVar(&useStdin, "stdin", false, "Read YAML from stdin instead of file")
	Cmd.AddCommand(putCmd)
}

func runPut(cmd *cobra.Command, args []string) {
	if useStdin {
		// Read from stdin
		yamlContent := functional.MustReturn(io.ReadAll(os.Stdin))
		uploadSingleGame(yamlContent, args)
		return
	}

	path := args[0]
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot access path: %v\n", err)
		os.Exit(1)
	}

	if fileInfo.IsDir() {
		// Upload all YAML files in directory
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "Error: Cannot specify game-id when uploading a directory\n")
			os.Exit(1)
		}
		uploadDirectory(path)
	} else {
		// Upload single file
		yamlContent := functional.MustReturn(os.ReadFile(path))
		uploadSingleGame(yamlContent, args)
	}
}

func uploadDirectory(dirPath string) {
	// Find all .yaml and .yml files
	yamlFiles, err := filepath.Glob(filepath.Join(dirPath, "*.yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding YAML files: %v\n", err)
		os.Exit(1)
	}

	ymlFiles, err := filepath.Glob(filepath.Join(dirPath, "*.yml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding YML files: %v\n", err)
		os.Exit(1)
	}

	allFiles := append(yamlFiles, ymlFiles...)

	if len(allFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No YAML files found in directory: %s\n", dirPath)
		os.Exit(1)
	}

	fmt.Printf("Found %d YAML file(s) in %s\n", len(allFiles), dirPath)

	successCount := 0
	failCount := 0

	for i, filePath := range allFiles {
		fileName := filepath.Base(filePath)
		fmt.Printf("\n[%d/%d] Uploading %s...\n", i+1, len(allFiles), fileName)

		yamlContent, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Failed to read %s: %v\n", fileName, err)
			failCount++
			continue
		}

		var resp obj.Game
		err = client.ApiPostRaw("games/new", string(yamlContent), &resp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Failed to upload %s: %v\n", fileName, err)
			failCount++
			continue
		}

		fmt.Printf("✓ Created game: %s (ID: %s)\n", resp.Name, resp.ID.String())
		successCount++
	}

	fmt.Printf("\n=== Upload Summary ===\n")
	fmt.Printf("Total files: %d\n", len(allFiles))
	fmt.Printf("✓ Successful: %d\n", successCount)
	if failCount > 0 {
		fmt.Printf("✗ Failed: %d\n", failCount)
	}
}

func uploadSingleGame(yamlContent []byte, args []string) {
	if len(args) > 1 {
		// Update existing game
		gameID := args[1]
		functional.Must(client.ApiPutRaw(fmt.Sprintf("games/%s/yaml", gameID), string(yamlContent)), "failed to update game")
		fmt.Println("Game updated successfully")
		printGameInfo(gameID)
	} else {
		// Create new game
		var resp obj.Game
		functional.Must(client.ApiPostRaw("games/new", string(yamlContent), &resp), "failed to create game")
		fmt.Println("Game created successfully")
		printGameInfo(resp.ID.String())
	}
}
