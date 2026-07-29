package appcloud

import (
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app [app-id]",
	Short: "Show or set the default app used when <app-id> is omitted",
	Long: `
With an <app-id>, remember it as the default app for other ` + "`hx appcloud`" + ` commands
(persisted in ~/.hx). With no argument, show the current default app.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)

		// `app <id>` — set the default app.
		if len(args) == 1 {
			mc, err := config.RestoreGlobalConfig()
			if err != nil {
				return err
			}
			id := args[0]
			mc.AppCloudAppId = &id
			if err := config.UpsertGlobalConfig(mc); err != nil {
				return err
			}
			printer.Success("Default AppCloud app set to " + id)
			return nil
		}

		// `app` — show the current default app.
		appID, err := resolveAppID("")
		if err != nil {
			return err
		}
		app, err := newAppCloudService().GetApp(appID)
		if err != nil {
			return err
		}
		printAppSummary(printer, app)
		return nil
	},
}

func init() {
	appCmd.AddCommand(newConfigCmd(true)) // `app config <get|set>` on the default app
}
