package appcloud

import (
	"os"

	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Manage AppCloud apps (list, get, create, delete, deploy, config)",
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List AppCloud sites in your organization",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)
		orgID, err := flags.GetOrganizationID()
		if err != nil {
			return err
		}
		projectID, _ := flags.GetProjectID() // optional narrowing

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

var appsGetCmd = &cobra.Command{
	Use:   "get <app-id>",
	Short: "Show a single AppCloud app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)
		app, err := newAppCloudService().GetApp(args[0])
		if err != nil {
			return err
		}
		printAppSummary(printer, app)
		return nil
	},
}

var (
	createOwner   string
	createName    string
	createDomains []string
)

var appsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new AppCloud app",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)
		orgID, err := flags.GetOrganizationID()
		if err != nil {
			return err
		}
		projID, err := flags.GetProjectID()
		if err != nil {
			return err
		}
		app, err := newAppCloudService().CreateApp(createOwner, createName, orgID, projID, createDomains)
		if err != nil {
			return err
		}
		printer.Success("Created app " + app.ID)
		printAppSummary(printer, app)
		return nil
	},
}

var appsDeleteCmd = &cobra.Command{
	Use:   "delete <app-id>",
	Short: "Delete an AppCloud app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)
		if err := newAppCloudService().DeleteApp(args[0]); err != nil {
			return err
		}
		printer.Success("Deleted app " + args[0])
		return nil
	},
}

var (
	appsDeployKind        string
	appsDeployArtifactRef string
	appsDeployBatchSize   int
	appsDeployNoActivate  bool
)

var appsDeployCmd = &cobra.Command{
	Use:   "deploy <app-id> <dir>",
	Short: "Upload a directory as a new revision under an existing app",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)
		appID, dir := args[0], args[1]
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return err
		}
		svc := newAppCloudService()
		app, err := svc.GetApp(appID)
		if err != nil {
			return err
		}
		return shipRevision(svc, printer, app, dir, appsDeployKind, appsDeployArtifactRef, appsDeployBatchSize, appsDeployNoActivate)
	},
}

func init() {
	appsCreateCmd.Flags().StringVar(&createOwner, "owner", "", "Owner label")
	appsCreateCmd.Flags().StringVar(&createName, "name", "", "App name (URL-safe slug)")
	appsCreateCmd.Flags().StringArrayVar(&createDomains, "domain", nil, "Domain to attach (repeatable)")

	appsDeployCmd.Flags().StringVar(&appsDeployKind, "kind", "static", "Revision kind")
	appsDeployCmd.Flags().StringVar(&appsDeployArtifactRef, "artifact-ref", "", "Client artifact reference (default: {name}@{unix_ts})")
	appsDeployCmd.Flags().IntVar(&appsDeployBatchSize, "batch-size", 100, "Max files per upload request")
	appsDeployCmd.Flags().BoolVar(&appsDeployNoActivate, "no-activate", false, "Upload the revision without pinning it active")

	appsCmd.AddCommand(appsListCmd)
	appsCmd.AddCommand(appsGetCmd)
	appsCmd.AddCommand(appsCreateCmd)
	appsCmd.AddCommand(appsDeleteCmd)
	appsCmd.AddCommand(appsDeployCmd)
	appsCmd.AddCommand(newConfigCmd(false)) // `apps config <get|set> <app-id>`
}
