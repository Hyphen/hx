package appcloud

import (
	"fmt"
	"os"
	"time"

	"github.com/Hyphen/cli/internal/appcloud"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	deployName        string
	deployOwner       string
	deployDomains     []string
	deployKind        string
	deployArtifactRef string
	deployBatchSize   int
	deployNoActivate  bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy <dir>",
	Short: "Deploy a directory to AppCloud (create-or-find app, upload, activate)",
	Long: `
End-to-end deploy: find the app by its first --domain (or create one), register
a new revision, upload the directory's contents, and pin it active.

Examples:
  hx appcloud deploy ./dist --name my-site --owner me --domain my-site.app.hyphen.cloud
  hx appcloud deploy ./public --name docs --owner me --domain docs.acme.com --no-activate`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)
		dir := args[0]
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			return errors.Wrapf(errors.New("not a directory"), "%s", dir)
		}
		svc := newAppCloudService()

		app, err := resolveDeployApp(svc, printer)
		if err != nil {
			return err
		}
		return shipRevision(svc, printer, app, dir, deployKind, deployArtifactRef, deployBatchSize, deployNoActivate)
	},
}

// shipRevision registers a revision, uploads `dir`, and (unless noActivate)
// pins it active — the shared tail of `deploy` and `apps deploy`.
func shipRevision(svc appcloud.AppCloudServicer, printer *cprint.CPrinter, app appcloud.App, dir, kind, artifactRef string, batchSize int, noActivate bool) error {
	if artifactRef == "" {
		artifactRef = fmt.Sprintf("%s@%d", app.Name, time.Now().Unix())
	}
	rev, err := svc.CreateRevision(app.ID, kind, artifactRef)
	if err != nil {
		return err
	}
	printer.Info(fmt.Sprintf("registered revision %s (%s)", rev.Hex, rev.ArtifactRef))

	n, err := appcloud.UploadDirectory(svc, app.ID, rev.Hex, dir, batchSize, func(msg string) { printer.Print(msg) })
	if err != nil {
		return err
	}
	if n == 0 {
		printer.Warning(fmt.Sprintf("nothing to upload (no files under %s)", dir))
	}

	if noActivate {
		printer.Info("revision uploaded but NOT activated (--no-activate)")
		if rev.PreviewURL != "" {
			printer.Print("preview: " + rev.PreviewURL)
		}
		return nil
	}
	updated, err := svc.SetActiveRevision(app.ID, rev.Hex)
	if err != nil {
		return err
	}
	printer.Success("pinned active revision: " + rev.Hex)
	if rev.PreviewURL != "" {
		printer.Print("preview: " + rev.PreviewURL)
	}
	for _, d := range updated.Domains {
		printer.Print("live at: https://" + d + "/")
	}
	return nil
}

// resolveDeployApp finds the target app by its first domain, or creates one.
// When an existing app matches the domain, its owner/name must match the flags
// so a deploy can't silently ship into a foreign app that shares a domain.
func resolveDeployApp(svc appcloud.AppCloudServicer, printer *cprint.CPrinter) (appcloud.App, error) {
	if len(deployDomains) > 0 {
		existing, err := svc.FindAppByDomain(deployDomains[0])
		if err != nil {
			return appcloud.App{}, err
		}
		if existing != nil {
			if deployOwner != "" && existing.Owner != deployOwner ||
				deployName != "" && existing.Name != deployName {
				return appcloud.App{}, fmt.Errorf(
					"domain %s is already attached to app %s (owner=%s, name=%s); refusing to deploy with mismatched --owner/--name",
					deployDomains[0], existing.ID, existing.Owner, existing.Name)
			}
			printer.Info(fmt.Sprintf("found existing app %s for %s", existing.ID, deployDomains[0]))
			return *existing, nil
		}
	}
	return createDeployApp(svc, printer)
}

func createDeployApp(svc appcloud.AppCloudServicer, printer *cprint.CPrinter) (appcloud.App, error) {
	if deployName == "" {
		return appcloud.App{}, errors.New("creating a new app requires --name")
	}
	owner, err := resolveOwner(deployOwner)
	if err != nil {
		return appcloud.App{}, err
	}
	orgID, err := flags.GetOrganizationID()
	if err != nil {
		return appcloud.App{}, err
	}
	projID, err := flags.GetProjectID()
	if err != nil {
		return appcloud.App{}, err
	}
	app, err := svc.CreateApp(owner, deployName, orgID, projID, deployDomains)
	if err != nil {
		return appcloud.App{}, err
	}
	printer.Success(fmt.Sprintf("created app %s (%s)", app.ID, app.Name))
	return app, nil
}

func init() {
	deployCmd.Flags().StringVar(&deployName, "name", "", "App name (URL-safe slug); required when creating")
	deployCmd.Flags().StringVar(&deployOwner, "owner", "", "Owner label (default: your logged-in user)")
	deployCmd.Flags().StringArrayVar(&deployDomains, "domain", nil, "Domain to attach (repeatable); the first is used to find an existing app")
	deployCmd.Flags().StringVar(&deployKind, "kind", "static", "Revision kind")
	deployCmd.Flags().StringVar(&deployArtifactRef, "artifact-ref", "", "Client artifact reference (default: {name}@{unix_ts})")
	deployCmd.Flags().IntVar(&deployBatchSize, "batch-size", 100, "Max files per upload request")
	deployCmd.Flags().BoolVar(&deployNoActivate, "no-activate", false, "Upload the revision without pinning it active")
}
