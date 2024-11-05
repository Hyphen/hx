package setproject

import (
	"fmt"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var (
	globalFlag bool
	printer    *cprint.CPrinter
)

var SetProjectCmd = &cobra.Command{
	Use:   "set-project <id>",
	Short: "Set the project ID",
	Long:  `Set the project ID for the Hyphen CLI to use.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID := args[0]
		var err error

		if globalFlag {
			err = manifest.UpsertGlobalProjectID(projectID)
		} else {
			err = manifest.UpsertProjectID(projectID)
		}

		if err != nil {
			return fmt.Errorf("failed to update project ID: %w", err)
		}
		printProjectUpdateSuccess(projectID, globalFlag)
		return nil
	},
}

func init() {
	SetProjectCmd.Flags().BoolVar(&globalFlag, "global", false, "Set the project ID globally")
	printer = cprint.NewCPrinter(flags.VerboseFlag)
}

func printProjectUpdateSuccess(projectID string, isGlobal bool) {
	printer.PrintHeader("--- Project Update ---")
	if isGlobal {
		printer.Success("Successfully updated global project ID")
	} else {
		printer.Success("Successfully updated project ID")
	}
	printer.PrintDetail("New Project ID", projectID)
	fmt.Println()
	if isGlobal {
		printer.GreenPrint("Hyphen CLI is now set to use the new project globally.")
	} else {
		printer.GreenPrint("Hyphen CLI is now set to use the new project.")
	}
}
