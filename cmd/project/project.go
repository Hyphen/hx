package project

import (
	"github.com/Hyphen/cli/cmd/project/create"
	"github.com/Hyphen/cli/cmd/project/get"
	"github.com/Hyphen/cli/cmd/project/list"
	"github.com/spf13/cobra"
)

var ProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long: `
Manage projects within your organization.

This command allows you to interact with the project resources in your organization.
You can list all projects, get details of a specific project and create a new project.

Examples:
  hyphen project list
  hyphen project get <project_id>
  hyphen project create "New Project"
`,
	Run: func(cmd *cobra.Command, args []string) {
		// check if the subcommand is unsupported
		// If no subcommand is provided, default to 'list' command
		list.ProjectListCmd.Run(cmd, args)
	},
}

func init() {
	ProjectCmd.AddCommand(list.ProjectListCmd)
	ProjectCmd.AddCommand(get.ProjectGetCmd)
	ProjectCmd.AddCommand(create.ProjectCreateCmd)
}
