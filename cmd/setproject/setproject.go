package setproject

import (
	"fmt"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
)

var SetProjectCmd = &cobra.Command{
	Use:   "set-project <id>",
	Short: "Set the project ID",
	Long:  `Set the project ID for the Hyphen CLI to use.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID := args[0]
		err := manifest.UpsertProjectID(projectID)
		if err != nil {
			return fmt.Errorf("failed to update project ID: %w", err)
		}
		printProjectUpdateSuccess(projectID)
		return nil
	},
}

func printProjectUpdateSuccess(projectID string) {
	cprint.PrintHeader("--- Project Update ---")
	cprint.Success("Successfully updated project ID")
	cprint.PrintDetail("New Project ID", projectID)
	fmt.Println()
	cprint.GreenPrint("Hyphen CLI is now set to use the new project.")
}
