package initialize

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var projectIDFlag string

var InitCmd = &cobra.Command{
	Use:   "init <project name>",
	Short: "init a project",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectService := app.NewService()
		orgID, err := utils.GetOrganizationID()
		if err != nil {
			cmd.PrintErrf("Error: %s\n", err)
			return
		}

		projectName := args[0]
		if projectName == "" {
			cmd.PrintErrf("Project name is required.\n")
			return
		}

		defaultProjectAlternateId := generateDefaultProjectId(projectName)
		projectAlternateId := projectIDFlag
		if projectAlternateId == "" {
			projectAlternateId = defaultProjectAlternateId
		}

		err = app.CheckProjectId(projectAlternateId)
		if err != nil {
			suggestedID := strings.TrimSpace(strings.Split(err.Error(), ":")[1])
			yesFlag, _ := cmd.Flags().GetBool("yes")
			if yesFlag {
				projectAlternateId = suggestedID
				cmd.Printf("Using suggested project ID: %s\n", suggestedID)
			} else {
				if !promptForSuggestedID(os.Stdin, suggestedID) {
					cmd.Println("Operation cancelled.")
					return
				}
				projectAlternateId = suggestedID
			}
		}

		if manifest.Exists() {
			yesFlag, _ := cmd.Flags().GetBool("yes")
			if !yesFlag {
				if !promptForOverwrite(os.Stdin) {
					cmd.Println("Operation cancelled.")
					return
				}
			}
		}

		newProject, err := projectService.CreateApp(orgID, projectAlternateId, projectName)
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

func init() {
	InitCmd.Flags().StringVarP(&projectIDFlag, "id", "i", "", "Project ID (optional)")
}

func generateDefaultProjectId(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}

func promptForOverwrite(reader io.Reader) bool {
	r := bufio.NewReader(reader)
	for {
		fmt.Print("Manifest file exists. Do you want to overwrite it? (y/N): ")
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

func promptForSuggestedID(reader io.Reader, suggestedID string) bool {
	r := bufio.NewReader(reader)
	for {
		fmt.Printf("Invalid project ID. Do you want to use the suggested ID [%s]? (Y/n): ", suggestedID)
		response, err := r.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
			os.Exit(1)
		}
		response = strings.TrimSpace(response)
		switch strings.ToLower(response) {
		case "y", "yes", "":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Invalid response. Please enter 'y' or 'n'.")
		}
	}
}
