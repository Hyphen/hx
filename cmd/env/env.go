package env

import (
	"github.com/Hyphen/cli/cmd/env/decrypt"
	"github.com/Hyphen/cli/cmd/env/edit"
	"github.com/Hyphen/cli/cmd/env/initialize"
	"github.com/Hyphen/cli/cmd/env/merge"
	"github.com/Hyphen/cli/cmd/env/push"
	"github.com/Hyphen/cli/cmd/env/raw"
	"github.com/Hyphen/cli/cmd/env/run"
	"github.com/Hyphen/cli/cmd/env/show"
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
	EnvCmd.AddCommand(edit.EditCmd)
	EnvCmd.AddCommand(merge.MergeCmd)
	EnvCmd.AddCommand(push.PushCmd)
	EnvCmd.AddCommand(raw.RawCmd)
	EnvCmd.AddCommand(run.RunCmd)
	EnvCmd.AddCommand(show.ShowCmd)
}
