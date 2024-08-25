package project

import (
	"github.com/Hyphen/cli/cmd/project/list"
	"github.com/spf13/cobra"
)

var ProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long:  `Project command allows you to manage your Hyphen projects.`,
}

func init() {
	ProjectCmd.AddCommand(list.ListCmd)
}
