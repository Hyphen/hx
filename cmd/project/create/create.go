package create

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/cmd/initproject"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	printer       *cprint.CPrinter
	isMonorepo    bool
	projectIDFlag string
)

var ProjectCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project in your organization",
	Long: `
The project create command sets up a new project within your Hyphen organization.

Usage:
  hyphen project create [name] [--monorepo] [--id project-id]

This command allows you to:
- Create a new project with a specified name
- Create a monorepo project using the --monorepo flag
- Specify a custom project ID using the --id flag

Example:
  hyphen project create "My New Project"
  hyphen project create "My Monorepo Project" --monorepo
  hyphen project create "Custom ID Project" --id my-custom-id
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)

		orgId, err := flags.GetOrganizationID()
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to get organization ID: %w", err))
			return
		}

		rawName := args[0]
		name := strings.TrimSpace(rawName)
		name = strings.Trim(name, "\"")
		name = strings.Trim(name, "'")

		alternateId := initproject.GetProjectID(cmd, name)
		if alternateId == "" {
			return
		}

		service := projects.NewService(orgId)

		// Check if project exists first
		_, err = service.GetProject(alternateId)
		if err == nil {
			printer.Error(cmd, fmt.Errorf("project with ID '%s' already exists", alternateId))
			return
		}

		project := projects.Project{
			Name:        name,
			AlternateID: alternateId,
			IsMonorepo:  isMonorepo,
		}

		newProject, err := service.CreateProject(project)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to create project: %w", err))
			return
		}

		printer.GreenPrint(fmt.Sprintf("Project '%s' created successfully!", name))
		printer.PrintDetail("Name", newProject.Name)
		printer.PrintDetail("ID", *newProject.ID)
		printer.PrintDetail("AlternateID", newProject.AlternateID)
		printer.PrintDetail("Monorepo", fmt.Sprintf("%t", newProject.IsMonorepo))
	},
}

func init() {
	ProjectCreateCmd.Flags().BoolVarP(&isMonorepo, "monorepo", "m", false, "Create the project as a monorepo")
	ProjectCreateCmd.Flags().StringVarP(&projectIDFlag, "id", "i", "", "Specify a custom project ID")
}
