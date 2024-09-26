package get

import (
	"fmt"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var GetCmd = &cobra.Command{
	Use:   "get <app name or id>",
	Short: "Get an app",
	Long:  `The 'hyphen get' command retrieves details of an app within your organization.`,
	Args:  cobra.ExactArgs(1),
	Run:   runGet,
}

func init() {
}

func runGet(cmd *cobra.Command, args []string) {
	appService := app.NewService()
	orgID, err := flags.GetOrganizationID()
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	appIdentifier := args[0]
	if appIdentifier == "" {
		cprint.Error(cmd, fmt.Errorf("app name or id is required"))
		return
	}

	retrievedApp, err := appService.GetApp(orgID, appIdentifier)
	if err != nil {
		cprint.Error(cmd, err)
		return
	}

	printAppDetails(retrievedApp)
}

func printAppDetails(app app.App) {
	cprint.PrintHeader("--- App Details ---")
	cprint.PrintDetail("App Name", app.Name)
	cprint.PrintDetail("App AlternateId", app.AlternateId)
	cprint.PrintDetail("App ID", app.ID)
	cprint.PrintDetail("Organization ID", app.Organization.ID)
	cprint.PrintDetail("Organization Name", app.Organization.Name)
}
