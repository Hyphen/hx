package show

import (
	"fmt"
	"github.com/spf13/cobra"
)

var ShowCmd = &cobra.Command{
	Use:     "show [ENVIRONMENT]",
	Aliases: []string{"s"},
	Short:   "Show an environmental variable file",
	Long:    `Displays the contents of an environmental variable file for the specified environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		environment := "default"
		if len(args) > 0 {
			environment = args[0]
		}
		fmt.Printf("Showing environment variable file for %s.\n", environment)
	},
}
