package build

import (
	"github.com/Hyphen/cli/internal/build"
	hyphenapp "github.com/Hyphen/cli/internal/hyphenApp"
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
The deploy command runs a deployment for a given deployment name.

Usage:
	hyphen deploy <deployment> [flags]

Examples:
hyphen deploy deploy-dev

Use 'hyphen link --help' for more information about available flags.
`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		service := build.NewService()
		build, err := service.RunBuild(printer, "", flags.VerboseFlag)

		if err != nil {
			printer.Error(cmd, err)
			return
		}

		url := hyphenapp.ApplicationLink(build.Organization.ID, build.Project.ID, build.App.ID)

		printer.Info("Build successful: " + url)
	},
}
