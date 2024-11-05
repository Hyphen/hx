package list

import (
	"fmt"

	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var printer *cprint.CPrinter

var ProjectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Args:  cobra.NoArgs,
	Long: `
The project list command retrieves and displays all projects associated with your Hyphen organization.

Usage:
  hyphen project list

This command allows you to:
- View all projects in your current organization
- See key details of each project at a glance

For each project, the command will display:
- Name: The project's full name
- ID: The unique identifier assigned by Hyphen
- AlternateID: The human-readable identifier generated from the project name

If no projects are found in your organization, you'll be informed accordingly.

The projects are displayed in a list format, with each project's details separated by an empty line for better readability.

Example:
  hyphen project list

This command does not accept any arguments. Use it to get a quick overview of all your projects within the organization.

Note: The list is fetched based on your current organization context. Ensure you're in the correct organization before running this command.
`,
	Run: func(cmd *cobra.Command, args []string) {
		orgId, err := flags.GetOrganizationID()
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to get organization ID: %w", err))
			return
		}

		service := projects.NewService(orgId)
		projects, err := service.ListProjects()
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to list projects: %w", err))
			return
		}

		if len(projects) == 0 {
			printer.YellowPrint("No projects found")
			return
		}

		for _, project := range projects {
			printer.PrintDetail("Name", project.Name)
			printer.PrintDetail("ID", *project.ID)
			printer.PrintDetail("AlternateID", project.AlternateID)
			printer.Print("")
		}

	},
}

func init() {
	printer = cprint.NewCPrinter(flags.VerboseFlag)
}
