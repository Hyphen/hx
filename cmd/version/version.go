package version

import (
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
)

// Version is set via ldflags
var Version string

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Hyphen",
	Args:  cobra.NoArgs,
	Long:  `All software has versions. This is Hyphen's`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersionInfo()
	},
}

func printVersionInfo() {
	version := GetVersion()
	cprint.PrintHeader("Hyphen Version Information")
	cprint.PrintDetail("Version", version)
}

// GetVersion returns the current version of the CLI
func GetVersion() string {
	if Version == "" {
		Version = "unknown" // Default if not set by ldflags
	}
	return Version
}
