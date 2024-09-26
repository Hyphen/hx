package create

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var ProjectCreateCmd = &cobra.Command{
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
