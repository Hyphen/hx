package list

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/app"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/aquasecurity/table"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	pageSize  int
	page      int
	showTable bool
	printer   *cprint.CPrinter
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications in your organization and project",
	Long: `
The list command retrieves and displays all applications associated with your organization and project.

This command allows you to:
- View all applications in your current organization and project
- Control the number of results per page and which page to view
- Choose between a detailed list view or a compact table view

The command will display various details about each application, including:
- Application details (name, alternate ID, and ID)
- Organization information (ID and name)

Examples:
  hyphen app list
  hyphen app list --page-size 20 --page 2
  hyphen app list --table

If no applications are found, you'll be informed accordingly.

Use 'hyphen app list --help' for more information about available flags.
`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		orgId, err := flags.GetOrganizationID()
		if err != nil {
			printer.Error(cmd, err)
			return
		}
		projectId, err := flags.GetProjectID()
		if err != nil {
			printer.Error(cmd, err)
			return
		}
		service := newService(app.NewService())

		apps, err := service.ListApps(orgId, projectId, pageSize, page)
		if err != nil {
			printer.Error(cmd, fmt.Errorf("failed to list apps: %w", err))
			return
		}

		if len(apps) == 0 {
			printer.Info("No applications found for the specified organization.")
			return
		}

		if showTable {
			displayTable(apps)
		} else {
			displayList(apps)
		}
	},
}

func displayTable(apps []app.App) {
	// Define color functions
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	t := table.New(os.Stdout)
	t.SetHeaders(
		cyan("App ID"),
		cyan("Alternate ID"),
		cyan("Name"),
		cyan("Organization ID"),
		cyan("Organization Name"),
	)

	for _, app := range apps {
		t.AddRow(
			green(app.ID),
			yellow(app.AlternateId),
			magenta(app.Name),
			blue(app.Organization.ID),
			red(app.Organization.Name),
		)
	}

	t.Render()
}

func displayList(apps []app.App) {
	for _, app := range apps {
		printer.PrintDetail("App Name", app.Name)
		printer.PrintDetail("App AlternateId", app.AlternateId)
		printer.PrintDetail("App ID", app.ID)
		printer.PrintDetail("Organization ID", app.Organization.ID)
		printer.PrintDetail("Organization Name", app.Organization.Name)
		fmt.Println()
	}
}

type service struct {
	appService app.AppServicer
}

func newService(appService app.AppServicer) *service {
	return &service{
		appService,
	}
}

func (s *service) ListApps(organizationId, projectId string, limit, page int) ([]app.App, error) {
	return s.appService.GetListApps(organizationId, projectId, limit, page)
}

func init() {
	ListCmd.Flags().IntVar(&pageSize, "page-size", 10, "Number of results per page")
	ListCmd.Flags().IntVar(&page, "page", 1, "Page number")
	ListCmd.Flags().BoolVar(&showTable, "table", false, "Display results in a table format")
}
