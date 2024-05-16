package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is set via ldflags
var version string

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Hyphen",
	Long:  `All software has versions. This is Hyphen's`,
	Run: func(cmd *cobra.Command, args []string) {
		if version == "" {
			version = "unknown" // Default if not set by ldflags
		}
		fmt.Printf("Hyphen Version %s\n", version)
	},
}
