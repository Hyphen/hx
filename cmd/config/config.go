package config

import (
	"github.com/Hyphen/cli/cmd/config/set"
	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Hyphen CLI configuration",
	Long:  `The config command allows you to manage your Hyphen CLI configuration.`,
}

func init() {
	ConfigCmd.AddCommand(set.SetCmd)
}
