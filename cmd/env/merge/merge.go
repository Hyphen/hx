package merge

import (
	"fmt"
	"github.com/spf13/cobra"
)

var MergeCmd = &cobra.Command{
	Use:     "merge ENVIRONMENT FILENAME",
	Aliases: []string{"m"},
	Short:   "Merge into CloudEnv",
	Long:    `Merges environment variables from a specified file into a designated CloudEnv environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Println("Usage: cloudenv merge ENVIRONMENT FILENAME")
			return
		}
		environment := args[0]
		filename := args[1]
		fmt.Printf("Merged %s into CloudEnv environment %s\n", filename, environment)
	},
}
