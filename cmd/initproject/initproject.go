package initproject

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var projectIDFlag string
var printer *cprint.CPrinter

var InitProjectCmd = &cobra.Command{
	Use:   "init-project <project name>",
	Short: "Initialize a new Hyphen project in the current directory",
	Long: `
The init-project command sets up a new Hyphen project in your current directory.

This command will:
- Create a new project in Hyphen
- Generate a local configuration file
- Update .gitignore to exclude sensitive files

If no project name is provided, it will prompt to use the current directory name.

The command will guide you through:
- Confirming or entering a project name
- Generating or providing a project ID
- Creating necessary local files

After initialization, you'll receive a summary of the new project, including its name, 
ID, and associated organization.

Examples:
  hyphen init-project
  hyphen init-project "My New Project"
  hyphen init-project "My New Project" --id my-custom-project-id
`,
	Args: cobra.MaximumNArgs(1),
	Run:  runInitProject,
}

func init() {
	InitProjectCmd.Flags().StringVarP(&projectIDFlag, "id", "i", "", "Project ID (optional)")
}

func runInitProject(cmd *cobra.Command, args []string) {
	printer = cprint.NewCPrinter(flags.VerboseFlag)

	if err := isValidDirectory(cmd); err != nil {
		printer.Error(cmd, err)
		printer.Info("Please change to a project directory and try again.")
		return
	}

	orgID, err := flags.GetOrganizationID()
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	projectService := projects.NewService(orgID)

	projectName := ""
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			printer.Error(cmd, err)
			return
		}

		projectName = filepath.Base(cwd)
		response := prompt.PromptYesNo(cmd, fmt.Sprintf("Use the current directory name '%s' as the project name?", projectName), true)
		if !response.Confirmed {
			if response.IsFlag {
				printer.Info("Operation cancelled due to --no flag.")
				return
			} else {
				projectName, err = prompt.PromptString(cmd, "What would you like the project name to be?")
				if err != nil {
					printer.Error(cmd, err)
					return
				}
			}
		}
	} else {
		projectName = args[0]
	}

	projectAlternateId := getProjectID(cmd, projectName)
	if projectAlternateId == "" {
		return
	}

	if manifest.ExistsLocal() {
		response := prompt.PromptYesNo(cmd, "Config file exists. Do you want to overwrite it?", false)
		if !response.Confirmed {
			if response.IsFlag {
				printer.Info("Operation cancelled due to --no flag.")
			} else {
				printer.Info("Operation cancelled.")
			}
			return
		}
	}

	newProject := projects.Project{
		Name:        projectName,
		AlternateID: projectAlternateId,
	}

	createdProject, err := projectService.CreateProject(newProject)
	if err != nil {
		if !errors.Is(err, errors.ErrConflict) {
			printer.Error(cmd, err)
			return
		}

		existingProject, handleErr := handleExistingProject(cmd, projectService, projectAlternateId)
		if handleErr != nil {
			printer.Error(cmd, handleErr)
			return
		}
		if existingProject == nil {
			printer.Info("Operation cancelled.")
			return
		}

		createdProject = *existingProject
	}

	mcl := manifest.Config{
		ProjectId:          createdProject.ID,
		ProjectAlternateId: &createdProject.AlternateID,
		ProjectName:        &createdProject.Name,
		OrganizationId:     orgID,
	}

	_, err = manifest.LocalInitialize(mcl)
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	printInitializationSummary(createdProject.Name, createdProject.AlternateID, *createdProject.ID, orgID)
}

func getProjectID(cmd *cobra.Command, projectName string) string {
	defaultProjectAlternateId := generateDefaultProjectId(projectName)
	projectAlternateId := projectIDFlag
	if projectAlternateId == "" {
		projectAlternateId = defaultProjectAlternateId
	}

	err := projects.CheckProjectId(projectAlternateId)
	if err != nil {
		suggestedID := strings.TrimSpace(strings.Split(err.Error(), ":")[1])
		response := prompt.PromptYesNo(cmd, fmt.Sprintf("Invalid app ID. Do you want to use the suggested ID [%s]?", suggestedID), true)

		if !response.Confirmed {
			if response.IsFlag {
				printer.Info("Operation cancelled due to --no flag.")
				return ""
			} else {
				// Prompt for a custom project ID
				for {
					var err error
					projectAlternateId, err = prompt.PromptString(cmd, "Enter a custom app ID:")
					if err != nil {
						printer.Error(cmd, err)
						return ""
					}

					err = projects.CheckProjectId(projectAlternateId)
					if err == nil {
						return projectAlternateId
					}
					printer.Warning("Invalid app ID. Please try again.")
				}
			}
		} else {
			projectAlternateId = suggestedID
		}
	}
	return projectAlternateId
}

func generateDefaultProjectId(projectName string) string {
	return strings.ToLower(strings.ReplaceAll(projectName, " ", "-"))
}

func printInitializationSummary(projectName, projectAlternateId, projectID, orgID string) {
	printer.Success("Project successfully initialized")
	printer.Print("") // Print an empty line for spacing
	printer.PrintDetail("Project Name", projectName)
	printer.PrintDetail("Project AlternateId", projectAlternateId)
	printer.PrintDetail("Project ID", projectID)
	printer.PrintDetail("Organization ID", orgID)
}

func isValidDirectory(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if cwd == homeDir {
		return fmt.Errorf("initialization in home directory not allowed")
	}

	return nil
}

func handleExistingProject(cmd *cobra.Command, projectService projects.ProjectService, projectAlternateId string) (*projects.Project, error) {
	response := prompt.PromptYesNo(cmd, fmt.Sprintf("A project with ID '%s' already exists. Do you want to use this existing project?", projectAlternateId), true)
	if !response.Confirmed {
		return nil, nil
	}

	existingProject, err := projectService.GetProject(projectAlternateId)
	if err != nil {
		return nil, err
	}

	printer.Info(fmt.Sprintf("Using existing project '%s' (%s)", existingProject.Name, existingProject.AlternateID))
	return &existingProject, nil
}
