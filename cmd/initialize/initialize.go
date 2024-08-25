package initialize

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Hyphen/cli/config"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/internal/project"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/spf13/cobra"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "init a project",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if manifest.Exists() {
			if !promptForOverwrite(os.Stdin) {
				cmd.Println("Operation cancelled.")
				return
			}
		}

		projectName, err := promptForProjectName(os.Stdin)
		if err != nil {
			cmd.PrintErrf("%s\n", err)
			return
		}

		defaultProjectId := generateDefaultProjectId(projectName)
		projectId, err := promptForProjectId(os.Stdin, defaultProjectId)
		if err != nil {
			cmd.PrintErrf("%s\n", err)
			return
		}

		credentials, err := config.LoadCredentials()
		if err != nil {
			cmd.PrintErrf("%s\n", err)
			return
		}

		_, err = manifest.Initialize(credentials.Default.OrganizationId, projectName, projectId)
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

func ensureGitignore(manifestFileName string) error {
	const gitDirPath = ".git"
	const gitignorePath = ".gitignore"

	if _, err := os.Stat(gitDirPath); os.IsNotExist(err) {
		return nil
	}

	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		file, err := os.Create(gitignorePath)
		if err != nil {
			return errors.Wrap(err, "Error creating .gitignore")
		}
		defer file.Close()

		_, err = file.WriteString(manifestFileName + "\n")
		if err != nil {
			return errors.Wrap(err, "Error writing to .gitignore")
		}
		return nil
	}

	file, err := os.OpenFile(gitignorePath, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return errors.Wrap(err, "Error opening .gitignore")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == manifestFileName {
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "Error reading .gitignore")
	}

	_, err = file.WriteString(manifestFileName + "\n")
	if err != nil {
		return errors.Wrap(err, "Error writing to .gitignore")
	}

	return nil
}
