package raw

import (
	"fmt"
	"github.com/spf13/cobra"
)

var RawCmd = &cobra.Command{
	Use:     "raw [ENVIRONMENT]",
	Aliases: []string{"r"},
	Short:   "Show raw encrypted data for an environment",
	Long:    `Shows the raw, encrypted data of environment variables for the specified environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		environment := "default"
		if len(args) > 0 {
			environment = args[0]
		}
		fmt.Printf("Raw encrypted data for environment %s.\n", environment)
	},
}
