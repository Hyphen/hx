package push

import (
	"fmt"
	"github.com/spf13/cobra"
)

var PushCmd = &cobra.Command{
	Use:     "push [ENVIRONMENT] [FILE]",
	Aliases: []string{"p"},
	Short:   "Push an existing environmental variable file",
	Long:    `Pushes the contents of an existing environmental variable file to CloudEnv for the specified environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			fmt.Println("Usage: cloudenv push [ENVIRONMENT] [FILE]")
			return
		}
		environment := args[0]
		filename := args[1]
		fmt.Printf("Pushed %s to CloudEnv environment %s\n", filename, environment)
	},
}
