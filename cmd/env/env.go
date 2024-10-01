package env

import (
	"github.com/Hyphen/cli/cmd/env/pull"
	"github.com/Hyphen/cli/cmd/env/push"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var EnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment .env secrets",
	Long:  `Manage environment .env secrets for different environments.`,
}

func init() {
	EnvCmd.PersistentFlags().StringVarP(&flags.EnvironmentFlag, "environment", "e", "", "Project Environment ID (e.g., pevr_12345)")
	EnvCmd.PersistentFlags().BoolVar(&flags.AllFlag, "all", false, "push/pull secrets for all environments")

	pull.PullCmd.Flags().AddFlagSet(EnvCmd.PersistentFlags())
	push.PushCmd.Flags().AddFlagSet(EnvCmd.PersistentFlags())

	EnvCmd.AddCommand(pull.PullCmd)
	EnvCmd.AddCommand(push.PushCmd)
}
