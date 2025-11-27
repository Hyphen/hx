package version

import (
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

// Version is set via ldflags
var Version string
var printer *cprint.CPrinter

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Hyphen",
	Args:  cobra.NoArgs,
	Long:  `All software has versions. This is Hyphen's`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer = cprint.NewCPrinter(flags.VerboseFlag)
		printVersionInfo()
		return nil
	},
}

func printVersionInfo() {
	version := GetVersion()
	printer.PrintHeader("Hyphen Version Information")
	printer.PrintDetail("Version", version)
}

// GetVersion returns the current version of the CLI
func GetVersion() string {
	if Version == "" {
		Version = "unknown" // Default if not set by ldflags
	}
	return Version
}
