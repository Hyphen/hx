package initproject

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/projects"
	"github.com/Hyphen/cli/internal/secret"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/gitutil"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
	"go.uber.org/thriftrw/ptr"
)

var projectIDFlag string
var IsMonorepo bool
var printer *cprint.CPrinter

var InitProjectCmd = &cobra.Command{
	Use:   "init-project <project name>",
	Short: "Initialize a new Hyphen project in the current directory",
	Long: `
The init-project command sets up a new Hyphen project in your current directory.

This command will:
- Create a new project in Hyphen
- Generate a local configuration file

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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	Run: func(cmd *cobra.Command, args []string) {
		RunInitProject(cmd, args)
	},
}

func init() {
	InitProjectCmd.Flags().StringVarP(&projectIDFlag, "id", "i", "", "Project ID (optional)")
	InitProjectCmd.Flags().BoolVarP(&IsMonorepo, "monorepo", "m", false, "Initialize a monorepo project")
}

func RunInitProject(cmd *cobra.Command, args []string) {
	printer = cprint.NewCPrinter(flags.VerboseFlag)

	if err := isValidDirectory(cmd); err != nil {
		printer.Error(cmd, err)
		printer.Info("Please change to a project directory and try again.")
		os.Exit(1)
	}

	orgID, err := flags.GetOrganizationID()
	if err != nil {
		printer.Error(cmd, err)
		os.Exit(1)
	}

	projectService := projects.NewService(orgID)

	projectName, shouldContinue, err := GetProjectName(cmd, args)
	if err != nil {
		printer.Error(cmd, err)
		os.Exit(1)
	}
	if !shouldContinue {
		os.Exit(0)
	}

	projectAlternateId := GetProjectID(cmd, projectName)
	if projectAlternateId == "" {
		os.Exit(0)
	}

	if config.ExistsLocal() {
		response := prompt.PromptYesNo(cmd, "Config file exists. Do you want to overwrite it?", false)
		if !response.Confirmed {
			if response.IsFlag {
				printer.Info("Operation cancelled due to --no flag.")
			} else {
				printer.Info("Operation cancelled.")
			}
			os.Exit(0)
		}
	}

	newProject := projects.Project{
		Name:        projectName,
		AlternateID: projectAlternateId,
		IsMonorepo:  IsMonorepo,
	}

	createdProject, err := projectService.CreateProject(newProject)
	if err != nil {
		if !errors.Is(err, errors.ErrConflict) {
			printer.Error(cmd, err)
			os.Exit(1)
		}

		existingProject, handleErr := HandleExistingProject(cmd, projectService, projectAlternateId)
		if handleErr != nil {
			printer.Error(cmd, handleErr)
			os.Exit(1)
		}
		if existingProject == nil {
			printer.Info("Operation cancelled.")
			os.Exit(0)
		}

		createdProject = *existingProject
	}

	mcl := config.Config{
		ProjectId:          createdProject.ID,
		ProjectAlternateId: &createdProject.AlternateID,
		ProjectName:        &createdProject.Name,
		OrganizationId:     orgID,
	}
	mcl.IsMonorepo = ptr.Bool(IsMonorepo)

	err = config.InitializeConfig(mcl, config.ManifestConfigFile)
	if err != nil {
		printer.Error(cmd, err)
		os.Exit(1)
	}
	_, err = secret.LoadSecret(orgID, *createdProject.ID, true)
	if err != nil {
		printer.Error(cmd, err)
		os.Exit(1)
	}

	if err := gitutil.EnsureGitignore(secret.ManifestSecretFile); err != nil {
		printer.Error(cmd, fmt.Errorf("error adding .hxkey to .gitignore: %w. Please do this manually if you wish", err))
	}
	PrintInitializationSummary(createdProject.Name, createdProject.AlternateID, *createdProject.ID, orgID)
}

func GetProjectID(cmd *cobra.Command, projectName string) string {
	defaultProjectAlternateId := GenerateDefaultProjectId(projectName)
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

func GenerateDefaultProjectId(projectName string) string {
	return strings.ToLower(strings.ReplaceAll(projectName, " ", "-"))
}

func PrintInitializationSummary(projectName, projectAlternateId, projectID, orgID string) {
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

func HandleExistingProject(cmd *cobra.Command, projectService projects.ProjectService, projectAlternateId string) (*projects.Project, error) {
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

func GetProjectName(cmd *cobra.Command, args []string) (string, bool, error) {
	if len(args) > 0 {
		return args[0], true, nil
	}

	// Get the local directory name
	cwd, err := os.Getwd()
	if err != nil {
		return "", false, err
	}

	dirName := filepath.Base(cwd)
	response := prompt.PromptYesNo(cmd, fmt.Sprintf("Use the current directory name '%s' as the project name?", dirName), true)

	if !response.Confirmed {
		if response.IsFlag {
			printer.Info("Operation cancelled due to --no flag.")
			return "", false, nil
		}

		// Prompt for a new project name
		projectName, err := prompt.PromptString(cmd, "What would you like the project name to be?")
		if err != nil {
			return "", false, err
		}
		return projectName, true, nil
	}

	return dirName, true, nil
}
