package appcloud

import (
	"fmt"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/appcloud"
	"github.com/Hyphen/cli/internal/config"
	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/spf13/cobra"
)

// requireDir errors clearly when `dir` is missing or is not a directory. A
// plain `os.Stat` check that returns the (nil) error for a non-directory path
// would let a command exit 0 without doing anything.
func requireDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	return nil
}

// newAppCloudService is an injection seam: commands call it to obtain the
// service, and tests swap it for a mock.
var newAppCloudService = func() appcloud.AppCloudServicer { return appcloud.NewService() }

// getExecutionContext resolves the caller's Hyphen identity; a seam for tests.
var getExecutionContext = func() (models.ExecutionContext, error) {
	return user.NewService().GetExecutionContext()
}

// resolveOwner returns the explicit --owner if given, otherwise defaults to
// the logged-in user's email (falling back to their member/user id). `owner`
// is a display/attribution label on the app, not a security principal.
func resolveOwner(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	ctx, err := getExecutionContext()
	if err != nil {
		return "", errors.Wrap(err, "could not determine your Hyphen identity for the app owner; pass --owner explicitly")
	}
	switch {
	case ctx.Member.Email != "":
		return ctx.Member.Email, nil
	case ctx.Member.ID != "":
		return ctx.Member.ID, nil
	case ctx.User.ID != "":
		return ctx.User.ID, nil
	default:
		return "", errors.New("could not determine an owner from your account; pass --owner explicitly")
	}
}

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
		return appsListCmd.RunE(cmd, args)
	},
}

func init() {
	AppCloudCmd.AddCommand(deployCmd)
	AppCloudCmd.AddCommand(appsCmd)
	AppCloudCmd.AddCommand(appCmd)
	AppCloudCmd.AddCommand(metricsCmd)
}

// resolveAppID returns the explicit id if given, otherwise the default app
// persisted in ~/.hx via `hx appcloud app <id>`.
func resolveAppID(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	mc, err := config.RestoreConfig()
	if err != nil {
		return "", err
	}
	if mc.AppCloudAppId != nil && *mc.AppCloudAppId != "" {
		return *mc.AppCloudAppId, nil
	}
	return "", errors.New("no app id given and no default app selected; pass an <app-id> or run `hx appcloud app <id>` to select one")
}

// printAppSummary renders one app in the human-readable list/detail format.
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
