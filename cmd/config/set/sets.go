package set

import (
	"github.com/spf13/cobra"
)

var SetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration values",
	Long:  `Set various configuration values for the Hyphen CLI.`,
}

func init() {
	SetCmd.AddCommand(organizationIDCmd)
}
