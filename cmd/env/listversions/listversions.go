package listversions

import (
	"fmt"
	"os"

	"github.com/Hyphen/cli/internal/env"
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

var ListVersionsCmd = &cobra.Command{
	Use:   "list-versions [environment-id]",
	Short: "List versions of an environment in Hyphen",
	Long: `
The list-versions command displays different versions of a specific environment stored in Hyphen for your current project.

This command allows you to:
- View all versions of a specific environment
- See key details about each version, including ID, version number, secret count, size, and publish date
- Display results in either a detailed list format or a concise table format

You must provide the environment ID as an argument.

You can customize the output using the following flags:
--page-size: Specify the number of results to display per page (default: 10)
--page: Specify which page of results to display (default: 1)
--table: Display results in a table format for a more compact view

The information displayed for each version includes:
- ID: The unique identifier for the environment
- Version: The version number of the environment
- Secrets Count: The number of secret variables stored in this version
- Size: The total size of the environment data for this version
- Published: The date and time when this version was published

Examples:
  hyphen list-versions my-env-id
  hyphen list-versions my-env-id --page-size 20
  hyphen list-versions my-env-id --page 2
  hyphen list-versions my-env-id --table
    `,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		if err := RunListVersions(args[0]); err != nil {
			printer.Error(cmd, err)
		}
	},
}

func RunListVersions(environmentId string) error {
	orgId, err := flags.GetOrganizationID()
	if err != nil {
		return err
	}

	appId, err := flags.GetApplicationID()
	if err != nil {
		return err
	}

	service := env.NewService()

	var envs []env.Env
	envs, err = service.ListEnvVersions(orgId, appId, environmentId, pageSize, page)
	if err != nil {
		return err
	}

	if len(envs) == 0 {
		printer.Info("No versions found for the specified environment.")
		return nil
	}

	if showTable {
		displayTable(envs)
	} else {
		displayList(envs)
	}

	return nil
}

func displayTable(envs []env.Env) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	t := table.New(os.Stdout)
	t.SetHeaders(
		cyan("ID"),
		cyan("Version"),
		cyan("Secrets Count"),
		cyan("Size"),
		cyan("Published"),
	)

	for _, e := range envs {
		id := "default"
		if e.ProjectEnv != nil {
			id = e.ProjectEnv.AlternateID
		}

		version := "-"
		if e.Version != nil {
			version = fmt.Sprintf("%d", *e.Version)
		}

		publishedTime := "-"
		if e.Published != nil {
			publishedTime = e.Published.Format("01/02/2006 3:04:05 PM")
		}

		t.AddRow(
			blue(id),
			green(version),
			yellow(fmt.Sprintf("%d", e.CountVariables)),
			magenta(e.Size),
			cyan(publishedTime),
		)
	}

	t.Render()
}

func displayList(envs []env.Env) {
	for _, e := range envs {
		id := "default"
		if e.ProjectEnv != nil {
			id = e.ProjectEnv.AlternateID
		}
		printer.PrintHeader(fmt.Sprintf("ID: %s", id))

		version := "-"
		if e.Version != nil {
			version = fmt.Sprintf("%d", *e.Version)
		}
		printer.PrintDetail("Version", version)

		printer.PrintDetail("Secrets Count", fmt.Sprintf("%d", e.CountVariables))
		printer.PrintDetail("Size", e.Size)

		publishedTime := "-"
		if e.Published != nil {
			publishedTime = e.Published.Format("01/02/2006 3:04:05 PM")
		}
		printer.PrintDetail("Published", publishedTime)

		fmt.Println()
	}
}

func init() {
	ListVersionsCmd.Flags().IntVar(&pageSize, "page-size", 10, "Number of results per page")
	ListVersionsCmd.Flags().IntVar(&page, "page", 1, "Page number")
	ListVersionsCmd.Flags().BoolVar(&showTable, "table", false, "Display results in a table format")
}
