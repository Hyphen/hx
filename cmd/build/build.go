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
	Run: func(cmd *cobra.Command, args []string) {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		service := build.NewService()
		build, err := service.RunBuild(printer, "", flags.VerboseFlag)

		if err != nil {
			printer.Error(cmd, err)
			return
		}

		url := hyphenapp.ApplicationBuildLink(build.Organization.ID, build.Project.ID, build.App.ID, build.Id)

		printer.Info("Build successful: " + url)
	},
}
