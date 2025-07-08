package get

import (
	"fmt"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var printer *cprint.CPrinter

var GetCmd = &cobra.Command{
	Use:   "get <app name or id>",
	Short: "Get an app",
	Long: `
The get command retrieves details of an application within your organization.

Usage:
  hyphen app get <app name or id>

This command allows you to:
- Fetch detailed information about a specific application
- Use either the application name or ID as the identifier

The command will display various details about the application, including:
- Project information (ID and name)
- Application details (name, alternate ID, and ID)
- Organization information (ID and name)

Example:
  hyphen app get my-app
  hyphen app get app-123456

After execution, you'll see a summary of the application's details.
`,
	Args: cobra.ExactArgs(1),
	Run:  runGet,
}

func runGet(cmd *cobra.Command, args []string) {
	printer = cprint.NewCPrinter(flags.VerboseFlag)

	appService := app.NewService()
	orgID, err := flags.GetOrganizationID()
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	appIdentifier := args[0]
	if appIdentifier == "" {
		printer.Error(cmd, fmt.Errorf("app name or id is required"))
		return
	}

	retrievedApp, err := appService.GetApp(orgID, appIdentifier)
	if err != nil {
		printer.Error(cmd, err)
		return
	}

	printAppDetails(retrievedApp)
}

func printAppDetails(app models.App) {
	if app.Project != nil {
		printer.PrintDetail("Project ID", app.Project.ID)
		printer.PrintDetail("Project Name", app.Project.Name)
	}
	printer.PrintDetail("App Name", app.Name)
	printer.PrintDetail("App AlternateId", app.AlternateId)
	printer.PrintDetail("App ID", app.ID)
	printer.PrintDetail("Organization ID", app.Organization.ID)
	printer.PrintDetail("Organization Name", app.Organization.Name)
}
