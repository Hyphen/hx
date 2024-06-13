package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set via ldflags
var Version string

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Hyphen",
	Long:  `All software has versions. This is Hyphen's`,
	Run: func(cmd *cobra.Command, args []string) {
		if Version == "" {
			Version = "unknown" // Default if not set by ldflags
		}
		fmt.Printf("Hyphen Version %s\n", Version)
	},
}

// GetVersion returns the current version of the CLI
func GetVersion() string {
	if Version == "" {
		Version = "unknown" // Default if not set by ldflags
	}
	return Version
}
