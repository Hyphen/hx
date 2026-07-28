package appcloud

import (
	"github.com/Hyphen/cli/internal/appcloud"
	"github.com/Hyphen/cli/internal/user"
	"github.com/spf13/cobra"
)

// newAppCloudService is an injection seam: commands call it to obtain the
// service, and tests swap it for a mock.
var newAppCloudService = func() appcloud.AppCloudServicer { return appcloud.NewService() }

var AppCloudCmd = &cobra.Command{
	Use:     "appcloud",
	Aliases: []string{"ac"},
	Short:   "Manage AppCloud sites and deployments",
	Long: `
Manage AppCloud — Hyphen's static-site platform.

AppCloud serves sites at platform-issued names ({site}.app.hyphen.cloud) and
your own verified domains. These commands talk to the AppCloud management API
using your Hyphen login; pass --dev (or set HYPHEN_DEV=true) to target the
development environment.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	// With no subcommand, list the org's sites.
	RunE: func(cmd *cobra.Command, args []string) error {
		return appsCmd.RunE(cmd, args)
	},
}

func init() {
	AppCloudCmd.AddCommand(appsCmd)
	AppCloudCmd.AddCommand(appCmd)
}
