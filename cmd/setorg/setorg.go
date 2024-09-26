package setorg

import (
	"fmt"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
)

var SetOrgCmd = &cobra.Command{
	Use:   "set-org <id>",
	Short: "Set the organization ID",
	Long:  `Set the organization ID for the Hyphen CLI to use.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgID := args[0]
		err := manifest.UpdateOrganizationID(orgID)
		if err != nil {
			return fmt.Errorf("failed to update organization ID: %w", err)
		}
		printOrgUpdateSuccess(orgID)
		return nil
	},
}

func printOrgUpdateSuccess(orgID string) {
	cprint.PrintHeader("--- Organization Update ---")
	cprint.Success("Successfully updated organization ID")
	cprint.PrintDetail("New Organization ID", orgID)
	fmt.Println()
	cprint.GreenPrint("Hyphen CLI is now set to use the new organization.")
}
