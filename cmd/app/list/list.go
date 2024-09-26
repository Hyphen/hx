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
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all applications associated with the organization and project",
	Long:  `Retrieve and display a list of all applications associated with a specified organization ID.`,
	Run: func(cmd *cobra.Command, args []string) {
		orgId, err := flags.GetOrganizationID()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}
		projectId, err := flags.GetProjectID()
		if err != nil {
			cprint.Error(cmd, err)
			return
		}
		service := newService(app.NewService())

		cprint.PrintHeader("Listing Applications")

		apps, err := service.ListApps(orgId, projectId, pageSize, page)
		if err != nil {
			cprint.Error(cmd, fmt.Errorf("failed to list apps: %w", err))
			return
		}

		if len(apps) == 0 {
			cprint.Info("No applications found for the specified organization.")
			return
		}

		if showTable {
			displayTable(apps)
		} else {
			displayList(apps)
		}

		cprint.Success("Applications listed successfully")
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
		cprint.PrintHeader("--- App Details ---")
		cprint.PrintDetail("App Name", app.Name)
		cprint.PrintDetail("App AlternateId", app.AlternateId)
		cprint.PrintDetail("App ID", app.ID)
		cprint.PrintDetail("Organization ID", app.Organization.ID)
		cprint.PrintDetail("Organization Name", app.Organization.Name)
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
