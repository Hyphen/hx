package env

import (
	"github.com/Hyphen/cli/cmd/env/list"
	"github.com/Hyphen/cli/cmd/env/listversions"
	"github.com/Hyphen/cli/cmd/env/pull"
	"github.com/Hyphen/cli/cmd/env/push"
	"github.com/Hyphen/cli/cmd/env/rotatekey"
	"github.com/Hyphen/cli/cmd/env/run"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var EnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment .env secrets",
	Long:  `Manage environment .env secrets for different environments.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
}

func init() {
	EnvCmd.PersistentFlags().StringVarP(&flags.EnvironmentFlag, "environment", "e", "", "Project Environment ID (e.g., pevr_12345)")

	pull.PullCmd.Flags().AddFlagSet(EnvCmd.PersistentFlags())
	push.PushCmd.Flags().AddFlagSet(EnvCmd.PersistentFlags())

	EnvCmd.AddCommand(pull.PullCmd)
	EnvCmd.AddCommand(push.PushCmd)
	EnvCmd.AddCommand(run.RunCmd)
	EnvCmd.AddCommand(list.ListCmd)
	EnvCmd.AddCommand(listversions.ListVersionsCmd)
	EnvCmd.AddCommand(rotatekey.RotateCmd)
}
