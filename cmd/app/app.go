package app

import (
	"github.com/Hyphen/cli/cmd/app/create"
	"github.com/Hyphen/cli/cmd/app/get"
	"github.com/Hyphen/cli/cmd/app/list"
	"github.com/spf13/cobra"
)

var AppCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage applications",
	Long:  `The 'hx app' command allows you to manage applications within your organization. You can list, create, and delete applications using the available subcommands.`,
	Run: func(cmd *cobra.Command, args []string) {
		list.ListCmd.Run(cmd, args)
	},
}

func init() {
	AppCmd.AddCommand(list.ListCmd)
	AppCmd.AddCommand(create.CreateCmd)
	AppCmd.AddCommand(get.GetCmd)
}
