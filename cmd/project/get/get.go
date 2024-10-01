package get

import (
	"fmt"

	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var ProjectGetCmd = &cobra.Command{
	Use:   "get [project_id]",
	Short: "Get a project by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID := args[0]
		orgId, err := flags.GetOrganizationID()
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to get organization ID: %w", err))
			return
		}

		service := projects.NewService(orgId)
		project, err := service.GetProject(projectID)
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to get project: %w", err))
			return
		}

		cprint.PrintDetail("Name", project.Name)
		cprint.PrintDetail("ID", *project.ID)
		cprint.PrintDetail("AlternateID", project.AlternateID)
	},
}
