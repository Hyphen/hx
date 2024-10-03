package setorg

import (
	"fmt"

	"github.com/Hyphen/cli/internal/manifest"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/spf13/cobra"
)

var (
	globalFlag bool
)

var SetOrgCmd = &cobra.Command{
	Use:   "set-org <id>",
	Short: "Set the organization ID",
	Long:  `Set the organization ID for the Hyphen CLI to use.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgID := args[0]
		var err error

		if globalFlag {
			err = manifest.UpsertGlobalOrganizationId(orgID)
		} else {
			err = manifest.UpsertOrganizationID(orgID)
		}

		if err != nil {
			return fmt.Errorf("failed to update organization ID: %w", err)
		}
		printOrgUpdateSuccess(orgID, globalFlag)
		return nil
	},
}

func init() {
	SetOrgCmd.Flags().BoolVar(&globalFlag, "global", false, "Set the organization ID globally")
}

func printOrgUpdateSuccess(orgID string, isGlobal bool) {
	cprint.PrintHeader("--- Organization Update ---")
	if isGlobal {
		cprint.Success("Successfully updated global organization ID")
	} else {
		cprint.Success("Successfully updated organization ID")
	}
	cprint.PrintDetail("New Organization ID", orgID)
	fmt.Println()
	if isGlobal {
		cprint.GreenPrint("Hyphen CLI is now set to use the new organization globally.")
	} else {
		cprint.GreenPrint("Hyphen CLI is now set to use the new organization.")
	}
}
