package env

import (
	"github.com/Hyphen/cli/cmd/env/decrypt"
	"github.com/Hyphen/cli/cmd/env/encrypt"
	"github.com/Hyphen/cli/cmd/env/initialize"
	"github.com/Hyphen/cli/cmd/env/merge"
	"github.com/Hyphen/cli/cmd/env/pull"
	"github.com/Hyphen/cli/cmd/env/push"
	"github.com/Hyphen/cli/cmd/env/run"
	"github.com/spf13/cobra"
)

var EnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Environment related commands",
	Long:  `Commands to manage environments.`,
}

func init() {
	EnvCmd.AddCommand(initialize.InitCmd)
	EnvCmd.AddCommand(decrypt.DecryptCmd)
	EnvCmd.AddCommand(merge.MergeCmd)
	EnvCmd.AddCommand(push.PushCmd)
	EnvCmd.AddCommand(run.RunCmd)
	EnvCmd.AddCommand(encrypt.EncryptCmd)
	EnvCmd.AddCommand(pull.PullCmd)

}
