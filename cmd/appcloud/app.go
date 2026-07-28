package appcloud

import (
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app <app-id>",
	Short: "Show details for a single AppCloud site",
	Long: `
Show the details for one AppCloud site by its ID, including its owner, attached
domains, and the currently active revision.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)

		orgID, err := flags.GetOrganizationID()
		if err != nil {
			return err
		}

		app, err := newAppCloudService().GetApp(orgID, args[0])
		if err != nil {
			return err
		}

		printAppSummary(printer, app)
		return nil
	},
}
