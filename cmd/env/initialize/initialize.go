package initialize

import (
	"fmt"
	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"i"},
	Short:   "Initialize the environment",
	Long:    `This command initializes the environment with necessary configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Environment initialized")
	},
}
