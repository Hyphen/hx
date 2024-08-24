package initialize

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/project"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "init a project",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		projectService := project.NewService()
		orgID, err := utils.GetOrganizationID()
		if err != nil {
			cmd.PrintErrf("Error: %s\n", err)
		}

		if manifest.Exists() {
			if !utils.YesFlag && !promptForOverwrite(os.Stdin) {
				cmd.Println("Operation cancelled.")
				return
			}
		}

		projectName, err := promptForProjectName(os.Stdin)
		if err != nil {
			cmd.PrintErrf("%s\n", err)
			return
		}

		defaultProjectAlternateId := generateDefaultProjectId(projectName)
		var projectAlternateId string
		if utils.YesFlag {
			projectAlternateId = defaultProjectAlternateId
		} else {
			projectAlternateId, err = promptForProjectId(os.Stdin, defaultProjectAlternateId)
			if err != nil {
				cmd.PrintErrf("%s\n", err)
				return
			}
		}

		newProject, err := projectService.CreateProject(orgID, projectAlternateId, projectName)
		if err != nil {
			cmd.PrintErrf("%s\n", err)
			os.Exit(1)
		}

		_, err = manifest.Initialize(orgID, newProject.Name, newProject.ID, newProject.AlternateId)
		if err != nil {
			cmd.PrintErrf("%s\n", err)
			os.Exit(1)
		}
		cmd.Println("Project initialized")

		if err := ensureGitignore(manifest.ManifestConfigFile); err != nil {
			cmd.PrintErrf("Error checking/updating .gitignore: %s\n", err)
			os.Exit(1)
		}
	},
}

func promptForProjectName(reader io.Reader) (string, error) {
	r := bufio.NewReader(reader)
	fmt.Print("Name of Project: ")
	projectName, err := r.ReadString('\n')
	if err != nil {
		return "", errors.Wrap(err, "Error reading input")
	}
	projectName = strings.TrimSpace(projectName)
	if projectName == "" {
		return "", errors.New("Project name cannot be empty")
	}
	return projectName, nil
}

func promptForOverwrite(reader io.Reader) bool {
	r := bufio.NewReader(reader)
	for {
		fmt.Print("Are you sure you want to overwrite the ManifestConfigFile? (y/N): ")
		response, err := r.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
			os.Exit(1)
		}
		response = strings.TrimSpace(response)
		switch strings.ToLower(response) {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Println("Invalid response. Please enter 'y' or 'n'.")
		}
	}
}

func promptForProjectId(reader io.Reader, defaultAppId string) (string, error) {
	r := bufio.NewReader(reader)
	for {
		fmt.Printf("Project ID [%s]: ", defaultAppId)
		appId, err := r.ReadString('\n')
		if err != nil {
			return "", errors.Wrap(err, "Error reading input")
		}
		appId = strings.TrimSpace(appId)
		if appId == "" {
			appId = defaultAppId
		}

		if err := project.CheckProjectId(appId); err == nil {
			return appId, nil
		}

		suggestedAppId := strings.ToLower(strings.ReplaceAll(appId, " ", "-"))
		fmt.Printf("Do you want to use [%s]? (Y/n): ", suggestedAppId)
		response, err := r.ReadString('\n')
		if err != nil {
			return "", errors.Wrap(err, "Error reading input")
		}
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" || response == "" {
			return suggestedAppId, nil
		} else if response == "n" || response == "no" {
			fmt.Println("Please enter a new Project ID (lowercase with hyphens).")
		} else {
			fmt.Println("Invalid response. Please enter 'y' or 'n'.")
		}
	}
}

func generateDefaultProjectId(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}
