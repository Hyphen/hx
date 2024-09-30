package list

import (
	"fmt"

	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var ProjectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Run: func(cmd *cobra.Command, args []string) {
		orgId, err := flags.GetOrganizationID()
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to get organization ID: %w", err))
			return
		}

		service := projects.NewService(orgId)
		projects, err := service.ListProjects()
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to list projects: %w", err))
			return
		}

		if len(projects) == 0 {
			cprint.YellowPrint("No projects found")
			return
		}

		cprint.PrintHeader("Projects")
		for _, project := range projects {
			cprint.PrintDetail("Name", project.Name)
			cprint.PrintDetail("ID", *project.ID)
			cprint.PrintDetail("AlternateID", project.AlternateID)
			cprint.Print("")
		}
	},
}
