package build

import (
	"github.com/Hyphen/cli/internal/build"
	hyphenapp "github.com/Hyphen/cli/internal/hyphenApp"
	"github.com/Hyphen/cli/internal/user"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	printer *cprint.CPrinter
)

var BuildCmd = &cobra.Command{
	Use:   "build ",
	Short: "Run a build and post it to hyphen",
	Long: `
The build command runs a build and uploads it to hyphen without deploying.

Usage:
	hyphen build [flags]

Examples:
hyphen build

Use 'hyphen build --help' for more information about available flags.
`,
	Args: cobra.RangeArgs(0, 1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return user.ErrorIfNotAuthenticated()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		service := build.NewService()
		build, err := service.RunBuild(cmd, printer, "", flags.VerboseFlag, flags.DockerfileFlag, flags.PreviewFlag)

		if err != nil {
			return err
		}

		url := hyphenapp.ApplicationBuildLink(build.Organization.ID, build.Project.ID, build.App.ID, build.Id)

		printer.Info("Build successful: " + url)
		return nil
	},
}

func init() {
	BuildCmd.Flags().StringVarP(&flags.DockerfileFlag, "dockerfile", "f", "", "Path to Dockerfile (e.g., ./Dockerfile or ./docker/Dockerfile.prod)")
	BuildCmd.Flags().BoolVarP(&flags.PreviewFlag, "preview", "r", false, "Save build for preview deployments")
}
