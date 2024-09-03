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

var appIDFlag string

var InitCmd = &cobra.Command{
	Use:   "init <app name>",
	Short: "Initialize a new app",
	Long: `The 'hyphen init' command initializes a new app within your organization.

You need to provide an app name as a positional argument. Optionally, you can specify a custom app ID using the '--id' flag. If no app ID is provided, a default ID will be generated based on the app name.

If a manifest file already exists, you will be prompted to confirm if you want to overwrite it, unless the '--yes' flag is provided, in which case the manifest file will be overwritten automatically.

Example usages:
  hyphen init MyApp
  hyphen init MyApp --id custom-app-id --yes

Flags:
  --id, -i   Specify a custom app ID (optional)
  --yes, -y  Automatically confirm prompts and overwrite files if necessary`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		appService := app.NewService()
		orgID, err := utils.GetOrganizationID()
		if err != nil {
			cmd.PrintErrf("Error: %s\n", err)
			return
		}

		appName := args[0]
		if appName == "" {
			cmd.PrintErrf("App name is required.\n")
			return
		}

		defaultAppAlternateId := generateDefaultAppId(appName)
		appAlternateId := appIDFlag
		if appAlternateId == "" {
			appAlternateId = defaultAppAlternateId
		}

		err = app.CheckAppId(appAlternateId)
		if err != nil {
			suggestedID := strings.TrimSpace(strings.Split(err.Error(), ":")[1])
			yesFlag, _ := cmd.Flags().GetBool("yes")
			if yesFlag {
				appAlternateId = suggestedID
				cmd.Printf("Using suggested app ID: %s\n", suggestedID)
			} else {
				if !promptForSuggestedID(os.Stdin, suggestedID) {
					cmd.Println("Operation cancelled.")
					return
				}
				appAlternateId = suggestedID
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

		newApp, err := appService.CreateApp(orgID, appAlternateId, appName)
		if err != nil {
			cmd.PrintErrf("%s\n", err)
			os.Exit(1)
		}

		_, err = manifest.Initialize(orgID, newApp.Name, newApp.ID, newApp.AlternateId)
		if err != nil {
			cmd.PrintErrf("%s\n", err)
			os.Exit(1)
		}
		cmd.Println("App initialized")

		if err := ensureGitignore(manifest.ManifestConfigFile); err != nil {
			cmd.PrintErrf("Error checking/updating .gitignore: %s\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	InitCmd.Flags().StringVarP(&appIDFlag, "id", "i", "", "App ID (optional)")
}

func generateDefaultAppId(appName string) string {
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
		fmt.Printf("Invalid app ID. Do you want to use the suggested ID [%s]? (Y/n): ", suggestedID)
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
