package create

import (
	"fmt"
	"strings"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/Hyphen/cli/pkg/prompt"
	"github.com/spf13/cobra"
)

var (
	appIDFlag string
	printer   *cprint.CPrinter
)

var CreateCmd = &cobra.Command{
	Use:   "create <app name>",
	Short: "Create a new application in Hyphen",
	Long: `
The create command sets up a new application in Hyphen.

Usage:
  hyphen create <app name> [flags]

This command allows you to:
- Create a new application with a specified name
- Optionally provide a custom app ID or use an automatically generated one
- Associate the new app with your current organization and project

The app ID will be automatically generated based on the app name if not provided. 
If the generated ID is invalid, you'll be prompted with a suggested valid ID.

Example:
  hyphen create "app"
  hyphen create "app" --id my-custom-app-id

After creation, you'll receive a summary of the new application, including its name, 
ID, and associated organization.
`,
	Args: cobra.ExactArgs(1),
	Run:  runCreate,
}

func init() {
	CreateCmd.Flags().StringVarP(&appIDFlag, "id", "i", "", "App ID (optional)")
}

func runCreate(cmd *cobra.Command, args []string) {
	printer = cprint.NewCPrinter(flags.VerboseFlag)
	appService := app.NewService()
	orgID, err := flags.GetOrganizationID()
	if err != nil {
		printer.Error(cmd, err)
		return
	}
	projID, err := flags.GetProjectID()
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	appName := args[0]
	if appName == "" {
		printer.Error(cmd, fmt.Errorf("app name is required"))
		return
	}

	appAlternateId := getAppID(cmd, appName)
	if appAlternateId == "" {
		return
	}

	newApp, err := appService.CreateApp(orgID, projID, appAlternateId, appName)
	if err != nil {
		printer.Error(cmd, err)
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
		response := prompt.PromptYesNo(cmd, fmt.Sprintf("Invalid app ID. Do you want to use the suggested ID [%s]?", suggestedID), true)

		if response.IsFlag && !response.Confirmed {
			printer.Info("Operation cancelled due to --no flag.")
			return ""
		}

		if !response.Confirmed {
			var customID string
			for {
				var err error
				customID, err = prompt.PromptString(cmd, "Enter a custom app ID:")
				if err != nil {
					printer.Error(cmd, err)
					return ""
				}

				err = app.CheckAppId(customID)
				if err == nil {
					return customID
				}
				printer.Warning("Invalid app ID. Please try again.")
			}
		}
		appAlternateId = suggestedID
	}
	return appAlternateId
}

func generateDefaultAppId(appName string) string {
	return strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
}

func printCreationSummary(appName, appAlternateId, appID, orgID string) {
	printer.Success("App successfully created")
	printer.Print("") // Print an empty line for spacing
	printer.PrintDetail("App Name", appName)
	printer.PrintDetail("App AlternateId", appAlternateId)
	printer.PrintDetail("App ID", appID)
	printer.PrintDetail("Organization ID", orgID)
}
