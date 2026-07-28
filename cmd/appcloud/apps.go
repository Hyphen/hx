package appcloud

import (
	"strings"

	"github.com/Hyphen/cli/internal/appcloud"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "List AppCloud sites in your organization",
	Long: `
List the AppCloud sites in your organization. If a default project is set (or
--project is passed) the list is narrowed to that project; otherwise every site
in the organization is shown.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)

		orgID, err := flags.GetOrganizationID()
		if err != nil {
			return err
		}
		// Project is an optional narrowing filter — no default is fine.
		projectID, _ := flags.GetProjectID()

		apps, err := newAppCloudService().ListApps(orgID, projectID)
		if err != nil {
			return err
		}

		if len(apps) == 0 {
			printer.Info("No AppCloud sites found.")
			return nil
		}

		for _, a := range apps {
			printAppSummary(printer, a)
		}
		return nil
	},
}

func printAppSummary(printer *cprint.CPrinter, a appcloud.App) {
	name := a.Name
	if name == "" {
		name = a.ID
	}
	printer.PrintHeader(name)
	printer.PrintDetail("ID", a.ID)
	if a.Owner != "" {
		printer.PrintDetail("Owner", a.Owner)
	}
	if len(a.Domains) > 0 {
		printer.PrintDetail("Domains", strings.Join(a.Domains, ", "))
	}
	if a.ActiveRevision != "" {
		printer.PrintDetail("Active revision", a.ActiveRevision)
	}
	printer.Print("")
}
