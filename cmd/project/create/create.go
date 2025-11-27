package create

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var printer *cprint.CPrinter

var ProjectCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project in your organization",
	Long: `
The project create command sets up a new project within your Hyphen organization.

Usage:
  hyphen project create [name]

This command allows you to:
- Create a new project with a specified name
- Automatically generate an alternate ID based on the project name

The project name:
- Can include spaces and special characters
- Will be trimmed of leading/trailing spaces and quotes

The alternate ID:
- Is automatically generated from the project name
- Contains only alphanumeric characters and hyphens
- Replaces spaces with hyphens and removes other special characters

After creation, you'll receive a summary of the new project, including its:
- Name
- ID (assigned by Hyphen)
- Alternate ID (generated from the name)

Example:
  hyphen project create "My New Project"

This will create a project named "My New Project" with an alternate ID like "my-new-project".
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)

		orgId, err := flags.GetOrganizationID()
		if err != nil {
			return fmt.Errorf("failed to get organization ID: %w", err)
		}

		rawName := args[0]
		name := strings.TrimSpace(rawName)
		name = strings.Trim(name, "\"")
		name = strings.Trim(name, "'")

		// Ensure the alternate ID is alphanumeric and replaces spaces with hyphens
		alternateId := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			} else if r == ' ' {
				return '-'
			}
			return -1 // Strip out any other non-alphanumeric characters
		}, name)

		service := projects.NewService(orgId)
		project := models.Project{
			Name:        name,
			AlternateID: alternateId,
		}

		// Call the service to create the project
		newProject, err := service.CreateProject(project)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		printer.GreenPrint(fmt.Sprintf("Project '%s' created successfully!", name))

		printer.PrintDetail("Name", newProject.Name)
		printer.PrintDetail("ID", *newProject.ID)
		printer.PrintDetail("AlternateID", newProject.AlternateID)
		return nil
	},
}
