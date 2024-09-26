package create

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	appIDFlag string
)

var CreateCmd = &cobra.Command{
	Use:   "create <app name>",
	Short: "Create a new app",
	Long: `The 'hyphen create' command creates a new app within your organization.

You need to provide an app name as a positional argument. Optionally, you can specify a custom app ID using the '--id' flag. If no app ID is provided, a default ID will be generated based on the app name.

Example usages:
  hyphen create MyApp
  hyphen create MyApp --id custom-app-id

Flags:
  --id, -i   Specify a custom app ID (optional)`,
	Args: cobra.ExactArgs(1),
	Run:  runCreate,
}

func init() {
	CreateCmd.Flags().StringVarP(&appIDFlag, "id", "i", "", "App ID (optional)")
}

func runCreate(cmd *cobra.Command, args []string) {
	appService := app.NewService()
	orgID, err := flags.GetOrganizationID()
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	appName := args[0]
	if appName == "" {
		cprint.Error(cmd, fmt.Errorf("app name is required"))
		return
	}

	appAlternateId := getAppID(cmd, appName)
	if appAlternateId == "" {
		return
	}

	newApp, err := appService.CreateApp(orgID, appAlternateId, appName)
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	printCreationSummary(newApp.Name, newApp.AlternateId, newApp.ID, orgID)
}

func getAppID(cmd *cobra.Command, appName string) string {
	defaultAppAlternateId := generateDefaultAppId(appName)
	appAlternateId := appIDFlag
	if appAlternateId == "" {
		appAlternateId = defaultAppAlternateId
	}

	err := app.CheckAppId(appAlternateId)
	if err != nil {
		suggestedID := strings.TrimSpace(strings.Split(err.Error(), ":")[1])
		yesFlag, _ := cmd.Flags().GetBool("yes")
		if yesFlag {
			appAlternateId = suggestedID
			cprint.Info(fmt.Sprintf("Using suggested app ID: %s", suggestedID))
		} else {
			if !promptForSuggestedID(suggestedID) {
				cprint.Info("Operation cancelled.")
				return ""
			}
			appAlternateId = suggestedID
		}
	}
	return appAlternateId
}

func generateDefaultAppId(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}

func promptForSuggestedID(suggestedID string) bool {
	var response string
	fmt.Printf("Invalid app ID. Do you want to use the suggested ID [%s]? (Y/n): ", suggestedID)
	fmt.Scanln(&response)
	switch strings.ToLower(strings.TrimSpace(response)) {
	case "y", "yes", "":
		return true
	case "n", "no":
		return false
	default:
		cprint.Warning("Invalid response. Please enter 'y' or 'n'.")
		return promptForSuggestedID(suggestedID)
	}
}

func printCreationSummary(appName, appAlternateId, appID, orgID string) {
	cprint.PrintHeader("--- App Creation Summary ---")
	cprint.Success("App successfully created")
	cprint.PrintDetail("App Name", appName)
	cprint.PrintDetail("App AlternateId", appAlternateId)
	cprint.PrintDetail("App ID", appID)
	cprint.PrintDetail("Organization ID", orgID)
}
