package edit

import (
	"fmt"
	"github.com/spf13/cobra"
)

var EditCmd = &cobra.Command{
	Use:     "edit [ENVIRONMENT]",
	Aliases: []string{"e"},
	Short:   "Edit an environmental variable file",
	Long:    `Opens an environmental variable file for the specific environment in a text editor, allowing modifications.`,
	Run: func(cmd *cobra.Command, args []string) {
		environment := "default"
		if len(args) > 0 {
			environment = args[0]
		}
		fmt.Printf("Opened environment variable file for %s in a text editor.\n", environment)
	},
}
