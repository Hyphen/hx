package project

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
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
		projectListCmd.Run(cmd, args)
	},
}

var projectListCmd = &cobra.Command{
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

var projectGetCmd = &cobra.Command{
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

		cprint.PrintHeader("Project")
		cprint.PrintDetail("Name", project.Name)
		cprint.PrintDetail("ID", *project.ID)
		cprint.PrintDetail("AlternateID", project.AlternateID)
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project with the provided name",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		orgId, err := flags.GetOrganizationID()
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to get organization ID: %w", err))
			return
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
		project := projects.Project{
			Name:        name,
			AlternateID: alternateId,
		}

		// Call the service to create the project
		newProject, err := service.CreateProject(project)
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to create project: %w", err))
			return
		}

		cprint.GreenPrint(fmt.Sprintf("Project '%s' created successfully!", name))

		cprint.PrintHeader("Project Details")
		cprint.PrintDetail("Name", newProject.Name)
		cprint.PrintDetail("ID", *newProject.ID)
		cprint.PrintDetail("AlternateID", newProject.AlternateID)
	},
}

func init() {
	ProjectCmd.AddCommand(projectListCmd)
	ProjectCmd.AddCommand(projectGetCmd)
	ProjectCmd.AddCommand(projectCreateCmd)
}
