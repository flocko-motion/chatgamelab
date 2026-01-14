package lang

import (
	langutil "cgl/lang"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all supported languages",
	Long:  `List all supported languages with their codes and native names.`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonOutput, _ := cmd.Flags().GetBool("json")

		languages := langutil.GetAllLanguages()

		// Sort by code for consistent output
		sort.Slice(languages, func(i, j int) bool {
			return languages[i].Code < languages[j].Code
		})

		if jsonOutput {
			// Output as JSON
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(languages); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Output as text table
			fmt.Println("Supported Languages:")
			fmt.Println("Code\tLanguage")
			fmt.Println("----\t--------")
			for _, lang := range languages {
				fmt.Printf("%s\t%s\n", lang.Code, lang.Label)
			}
			fmt.Printf("\nTotal: %d languages\n", len(languages))
		}
	},
}

func init() {
	Cmd.AddCommand(listCmd)
	listCmd.Flags().Bool("json", false, "Output as JSON")
}
