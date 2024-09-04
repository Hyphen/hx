package version

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Version is set via ldflags
var Version string

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Hyphen",
	Long:  `All software has versions. This is Hyphen's`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersionInfo()
	},
}

// Color definitions
var (
	cyan  = color.New(color.FgCyan).SprintFunc()
	white = color.New(color.FgWhite, color.Bold).SprintFunc()
)

func printVersionInfo() {
	version := GetVersion()
	fmt.Println("\n--- Hyphen Version Information ---")
	fmt.Printf("%s %s\n", white("Version:"), cyan(version))
}

func GetVersion() string {
	if Version == "" {
		Version = "unknown"
	}
	return Version
}
