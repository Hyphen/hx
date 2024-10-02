package setproject

import (
	"fmt"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
)

var (
	globalFlag bool
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
			err = manifest.UpsertGlobalProjectId(projectID)
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
}

func printProjectUpdateSuccess(projectID string, isGlobal bool) {
	cprint.PrintHeader("--- Project Update ---")
	if isGlobal {
		cprint.Success("Successfully updated global project ID")
	} else {
		cprint.Success("Successfully updated project ID")
	}
	cprint.PrintDetail("New Project ID", projectID)
	fmt.Println()
	if isGlobal {
		cprint.GreenPrint("Hyphen CLI is now set to use the new project globally.")
	} else {
		cprint.GreenPrint("Hyphen CLI is now set to use the new project.")
	}
}
