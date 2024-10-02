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
	Long: `
The project get command retrieves and displays details of a specific project within your Hyphen organization.

Usage:
  hyphen project get [project_id]

This command allows you to:
- Fetch detailed information about a specific project
- Use either the project's ID or alternate ID as the identifier

The command will display the following details about the project:
- Name: The project's full name
- ID: The unique identifier assigned by Hyphen
- AlternateID: The human-readable identifier generated from the project name

Examples:
  hyphen project get 12345abc-de67-89fg-hi01-jklmnopqrstu
  hyphen project get my-project-name

After execution, you'll see a summary of the project's details.

Note: Make sure you have the necessary permissions to access the project information.
`,
	Args: cobra.ExactArgs(1),
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
