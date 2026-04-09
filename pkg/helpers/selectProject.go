package helpers

import (
	"fmt"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
)

func SelectProject(organizationID, promptMessage string) (models.Project, error) {
	if promptMessage == "" {
		promptMessage = "Select a project:"
	}
	projectService := projects.NewService(organizationID)
	// TODO: handle pagination
	projects, err := projectService.ListProjects()
	if err != nil {
		return models.Project{}, err
	}
	if len(projects) == 0 {
		return models.Project{}, fmt.Errorf("no projects found")
	}
	if len(projects) == 1 {
		project := projects[0]
		printer := cprint.NewCPrinter(flags.VerboseFlag)
		printer.Print(fmt.Sprintf("You only have access to one project, automatically choosing %s", project.Name))
		return project, nil
	}
	choices := make([]prompt.Choice, len(projects))
	for i, project := range projects {
		choices[i] = prompt.Choice{
			Id:           *project.ID,
			Display:      fmt.Sprintf("%s (%s)", project.Name, *project.ID),
			OriginalData: project,
		}
	}
	choice, err := prompt.PromptSelection(choices, promptMessage)
	if err != nil {
		return models.Project{}, err
	}
	return choice.OriginalData.(models.Project), nil
}
