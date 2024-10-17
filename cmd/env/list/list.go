package list

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
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environment variables in Hyphen",
	Long: `
The list command displays environment variables stored in Hyphen for your current project.

This command allows you to:
- View all environments associated with your project
- See key details about each environment, including ID, version, secret count, size, and publish date
- Display results in either a detailed list format or a concise table format

You can customize the output using the following flags:
--page-size: Specify the number of results to display per page (default: 10)
--page: Specify which page of results to display (default: 1)
--table: Display results in a table format for a more compact view

The information displayed for each environment includes:
- ID: The unique identifier for the environment (default for the main environment)
- Version: The current version number of the environment
- Secrets Count: The number of secret variables stored in the environment
- Size: The total size of the environment data
- Published: The date and time when the environment was last published

Examples:
  hyphen list
  hyphen list --page-size 20
  hyphen list --page 2
  hyphen list --table
    `,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := RunList(args); err != nil {
			cprint.Error(cmd, err)
		}
	},
}

func RunList(args []string) error {
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
	envs, err = service.ListEnvs(orgId, appId, pageSize, page)
	if err != nil {
		return err
	}

	if len(envs) == 0 {
		cprint.Info("No environments found.")
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
	if len(envs) == 0 {
		fmt.Println("No environments to display.")
		return
	}

	for _, e := range envs {
		id := "default"
		if e.ProjectEnv != nil {
			id = e.ProjectEnv.AlternateID
		}
		cprint.PrintHeader(fmt.Sprintf("ID: %s", id))

		version := "-"
		if e.Version != nil {
			version = fmt.Sprintf("%d", *e.Version)
		}
		cprint.PrintDetail("Version", version)

		cprint.PrintDetail("Secrets Count", fmt.Sprintf("%d", e.CountVariables))
		cprint.PrintDetail("Size", e.Size)

		publishedTime := "-"
		if e.Published != nil {
			publishedTime = e.Published.Format("01/02/2006 3:04:05 PM")
		}
		cprint.PrintDetail("Published", publishedTime)

		fmt.Println()
	}
}

func init() {
	ListCmd.Flags().IntVar(&pageSize, "page-size", 10, "Number of results per page")
	ListCmd.Flags().IntVar(&page, "page", 1, "Page number")
	ListCmd.Flags().BoolVar(&showTable, "table", false, "Display results in a table format")
}
