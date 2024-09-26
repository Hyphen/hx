package env

import (
	"github.com/Hyphen/cli/cmd/env/pull"
	"github.com/Hyphen/cli/cmd/env/push"
	"github.com/spf13/cobra"
)

var EnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables",
	Long:  `Manage environment variables for different environments.`,
}

func init() {
	EnvCmd.AddCommand(push.PushCmd)
	EnvCmd.AddCommand(pull.PullCmd)
}
