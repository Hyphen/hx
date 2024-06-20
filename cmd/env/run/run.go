package run

import (
	"fmt"
	"github.com/spf13/cobra"
)

var RunCmd = &cobra.Command{
	Use:   "run [ENVIRONMENT] [COMMAND]",
	Short: "Run a command using some environmental variables",
	Long:  `Executes the specified command with the environment variables sourced from CloudEnv for the specified environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Println("Usage: cloudenv run [ENVIRONMENT] [COMMAND]")
			return
		}
		environment := args[0]
		command := args[1]
		fmt.Printf("Running command '%s' with environment variables from %s.\n", command, environment)
	},
}
